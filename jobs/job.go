package jobs

//Job a job to be run periodically
import (
	"bytes"
	"io"
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
	failedRetries    int
	MaxFailedRetries int       `json:"maxFailedRetries"`
	Status           JobStatus `json:"status"`
	Progress         float64   `json:"progress"`
	stop             chan bool
	JobName          string `json:"JobName"`
	Username         string `json:"Username"`
	Service          string `json:"Service"`
	password         string
	ResticPath       string   `json:"ResticPath"`
	ResticArguments  []string `json:"ResticArguments"`
}

func newJob(regTimer, retTimer time.Duration, maxFailedRetries int, jobName, username, service, password string, resticArguments []string) *Job {
	return &Job{
		RegularTimer:     regTimer,
		RetryTimer:       retTimer,
		failedRetries:    0,
		MaxFailedRetries: maxFailedRetries,
		Status:           statusReady,
		Progress:         0,
		stop:             make(chan bool),
		JobName:          jobName,
		Username:         username,
		Service:          service,
		password:         password,
		ResticArguments:  resticArguments}
}

//JobReturn status returns from jobs
type JobReturn int

const (
	returnStop  JobReturn = 0
	returnOk    JobReturn = 1
	returnRetry JobReturn = 2
)

//JobStatus stati the jobs can be in
type JobStatus int

const (
	statusReady    JobStatus = 0
	statusWaiting  JobStatus = 1
	statusFinished JobStatus = 2
	statusWorking  JobStatus = 3
)

func (job *Job) retrieveAndStorePassword() {
	key, err := keyring.Get(job.Service, job.Username)
	if err != nil {
		log.WithFields(log.Fields{"Job": job.JobName}).Warning("couldn't retrieve password.")
	}
	job.password = key
}

//loops until the cannel sends a message. Will send a message back when actually exited
func (job *Job) loop(wg *sync.WaitGroup) {
	job.Status = statusWaiting
	defer func() { job.Status = statusFinished }()
	defer wg.Done()
	for {
		select {
		case _ = <-job.stop:
			break
		default:
		}

		startTime := time.Now()

		var result JobReturn
		job.failedRetries = 0
		result = job.run()
		for result == returnRetry && (job.MaxFailedRetries < 0 || job.failedRetries < job.MaxFailedRetries) {
			job.failedRetries++
			time.Sleep(job.RetryTimer)
			result = job.run()
		}

		if result != returnOk {
			log.WithFields(log.Fields{"Job": job.JobName, "retries": strconv.Itoa(job.failedRetries)}).Warning("Stopping")
			break //break loop for job, it did fail completely
		}

		endTime := time.Now().UnixNano()
		used := endTime - startTime.UnixNano()

		log.WithFields(log.Fields{"Job": job.JobName, "Used": strconv.FormatFloat(float64(used)/float64(time.Second), 'f', 6, 64)}).Warning("successful")

		time.Sleep(job.RegularTimer - time.Duration(used))
	}
	job.stop <- true
}

//reads the output from the command, extracts the percentage that is finished and updates the job accordingly
//doesnt work currently because retic does some fancy printing that wont be catched outside a terminal
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

	//stdout, err := cmd.StdoutPipe()
	//if err == nil {
	//	go job.updateStatus(stdout) //doesnt work right now, see comment on func
	//}
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
