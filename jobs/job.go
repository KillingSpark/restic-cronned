package jobs

//Job a job to be run periodically
import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	keyring "github.com/zalando/go-keyring"
)

var (
//log = log.New()
)

//Job represents one job that will be run in a Queue
type Job struct {
	RegularTimer     time.Duration `json:"regularTimer"`
	RetryTimer       time.Duration `json:"retryTimer"`
	CurrentRetry     int           `json:"CurrentRetry"`
	MaxFailedRetries int           `json:"maxFailedRetries"`
	Status           JobStatus     `json:"status"`
	Progress         float64       `json:"progress"`
	WaitStart        time.Duration `json:"WaitStart"`
	WaitEnd          time.Duration `json:"WaitEnd"`
	stop             chan bool
	JobName          string `json:"JobName"`
	Username         string `json:"Username"`
	Service          string `json:"Service"`
	password         string
	ResticPath       string   `json:"ResticPath"`
	ResticArguments  []string `json:"ResticArguments"`
}

func newJob() *Job {
	return &Job{
		Status:   statusReady,
		Progress: 0,
		stop:     make(chan bool),
	}
}

//JobReturn status returns from jobs
type JobReturn int

const (
	returnStop  JobReturn = 0
	returnOk    JobReturn = 1
	returnRetry JobReturn = 2
)

//JobStatus stati the jobs can be in
type JobStatus string

const (
	statusReady   JobStatus = "ready"
	statusWaiting JobStatus = "waiting"
	statusStopped JobStatus = "stopped"
	statusWorking JobStatus = "working"
)

func (job *Job) retrieveAndStorePassword() {
	key, err := keyring.Get(job.Service, job.Username)
	if err != nil {
		log.WithFields(log.Fields{"Job": job.JobName}).Warning("couldn't retrieve password.")
	} else {
		job.password = key
	}
}

//Stop stops a job it will exit after if has finished if currently running (this may take a while!) or exit immediatly if waiting
func (job *Job) Stop() {
	log.WithFields(log.Fields{"Job": job.JobName}).Info("Stopping externally")
	job.stop <- true
	<-job.stop
	log.WithFields(log.Fields{"Job": job.JobName}).Info("Stopped externally")
}

//loops until there is a "true" in the stop channel. Will send a "true" back when actually exited
func (job *Job) loop(wg *sync.WaitGroup) {
	job.Status = statusWaiting
	defer func() { job.Status = statusStopped }()
	defer wg.Done()
	for {
		startTime := time.Now()

		var result JobReturn
		result = job.run()
		job.CurrentRetry = 0
		for result == returnRetry && (job.MaxFailedRetries < 0 || job.CurrentRetry < job.MaxFailedRetries) {
			if result != returnOk {
				job.CurrentRetry++
			}
			go job.delaySignal(false, job.RetryTimer)
			sig := <-job.stop
			if sig {
				job.stop <- true
				return
			}
			result = job.run()
		}

		if result != returnOk {
			log.WithFields(log.Fields{"Job": job.JobName, "retries": strconv.Itoa(job.CurrentRetry)}).Error("Stopping")
			break //break loop for job, it did fail completely
		}

		endTime := time.Now().UnixNano()
		used := endTime - startTime.UnixNano()

		log.WithFields(log.Fields{"Job": job.JobName, "Used": strconv.FormatFloat(float64(used)/float64(time.Second), 'f', 6, 64)}).Info("successful")

		go job.delaySignal(false, job.RegularTimer-time.Duration(used))
		doExit := <-job.stop
		if doExit {
			job.stop <- true
			break
		}
	}
}

func (job *Job) delaySignal(msg bool, dur time.Duration) {
	job.WaitStart = time.Duration(time.Now().UnixNano())
	job.WaitEnd = job.WaitStart + dur
	time.Sleep(dur)
	job.stop <- msg
}

//reads the output from the command, extracts the percentage that is finished and updates the job accordingly
//doesnt work currently because restic does some fancy printing that wont be catched outside a terminal
func (job *Job) updateStatus(stdout io.ReadCloser) {
	defer stdout.Close()
	buf := make([]byte, 256)
	var n, err = stdout.Read(buf)
	for !(n == 0 && err != nil) {
		var chunk = string(buf)
		chunk = strings.Replace(chunk, "\n", " ", -1)
		chunk = strings.Replace(chunk, "\t", " ", -1)
		chunk = strings.Replace(chunk, "\000", " ", 1)
		toks := strings.Split(chunk, " ")
		for _, token := range toks {
			if strings.HasSuffix(token, "%") {
				val, err := strconv.ParseFloat(token[:len(token)-1], 32)
				if err == nil {
					job.Progress = val
				}
			}
		}
		n, err = stdout.Read(buf)
	}
}
func (job *Job) getRepo() string {
	var repo string
	for idx, arg := range job.ResticArguments {
		if arg == "-r" && len(job.ResticArguments) > idx+2 {
			repo = job.ResticArguments[idx+1]
		}
	}
	return repo
}
func (job *Job) waitForRepoForLock() {
	files, err := ioutil.ReadDir(job.getRepo() + "/locks")
	if err != nil {
		return
	}
	for len(files) > 0 {
		time.Sleep(10 * time.Millisecond)
		files, err = ioutil.ReadDir(job.getRepo() + "/locks")
		if err != nil {
			return
		}
	}
}

func (job *Job) run() JobReturn {
	job.Status = statusWorking
	defer func() { job.Status = statusWaiting }()

	var cmd *exec.Cmd
	if len(job.ResticPath) > 0 {
		cmd = exec.Command(job.ResticPath, job.ResticArguments...)
	} else {
		cmd = exec.Command("restic", job.ResticArguments...)
	}

	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+job.password)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	//doesnt work right now, see comment on updateStatus
	//stdout, err := cmd.StdoutPipe()
	//if err == nil {
	//	go job.updateStatus(stdout)
	//}
	job.waitForRepoForLock()
	err := cmd.Run()

	var exitCode = 0

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get, and stderr will be
			// empty string very likely, so we use the default fail code, and format err
			// to string and set to stderr
			exitCode = 1
		}

		log.WithFields(log.Fields{"Job": job.JobName, "error": err.Error(), "message": stderr.String()}).Warning("error")
	}

	switch exitCode {
	case 0:
		return returnOk //everything fine
	default:
		return returnRetry //not fine but retryable
	}
}
