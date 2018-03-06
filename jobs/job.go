package jobs

//Job a job to be run periodically
import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	lastSuccess      int64
	stop             chan bool
	stopAnswer       chan bool
	trigger          chan TriggerType
	queue            *JobQueue
	JobNameToTrigger string `json:"NextJob"`
	JobName          string `json:"JobName"`
	Username         string `json:"Username"`
	Service          string `json:"Service"`
	password         string
	ResticPath       string   `json:"ResticPath"`
	ResticArguments  []string `json:"ResticArguments"`
}

func newJob() *Job {
	return &Job{
		Status:     statusReady,
		Progress:   0,
		stop:       make(chan bool),
		stopAnswer: make(chan bool),
		trigger:    make(chan TriggerType),
	}
}

//JobReturn status returns from jobs
type JobReturn int

const (
	returnStop  JobReturn = 0
	returnOk    JobReturn = 1
	returnRetry JobReturn = 2
)

//TriggerType extern triggers only followup jobs but does not retrigger himself
type TriggerType int

const (
	triggerIntern TriggerType = 0
	triggerExtern TriggerType = 1
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

//SendTrigger makes the job  run immediatly (if waiting or immediatly again if working right now)
func (job *Job) SendTrigger(trigType TriggerType) {
	if job.Status == statusWaiting || job.Status == statusWorking {
		job.trigger <- trigType
	}
}

//SendTriggerWithDelay makes the job run after "dur" nanoseconds
func (job *Job) SendTriggerWithDelay(dur time.Duration) {
	if dur < 0 {
		return
	}
	job.WaitStart = time.Duration(time.Now().UnixNano())
	job.WaitEnd = job.WaitStart + dur

	time.Sleep(dur)

	job.SendTrigger(triggerIntern)
}

func (job *Job) triggerNextJob() {
	toTrigger, _ := job.queue.findJob(job.JobNameToTrigger)
	if toTrigger != nil {
		toTrigger.SendTrigger(triggerIntern)
	}
}

func (job *Job) loop() {
	defer func() { job.Status = statusStopped }()
	defer job.finish()
	job.Status = statusWaiting
	for {
		select {
		case <-job.trigger:
		case <-job.stop:
			job.stopAnswer <- true
			return
		}

		result := job.run()
		switch result {
		case returnRetry:
			if job.CurrentRetry < job.MaxFailedRetries {
				job.retry()
			} else {
				job.fail()
			}
			break
		case returnOk:
			job.success()
			break
		case returnStop:
			job.fail()
			break
		}
	}
}

func (job *Job) start(queue *JobQueue) {
	job.queue = queue
	job.lastSuccess = time.Now().UnixNano()
	go job.SendTriggerWithDelay(job.RegularTimer)
	go job.loop()
}

func (job *Job) retry() {
	job.CurrentRetry++
	go job.SendTriggerWithDelay(job.RetryTimer)
}

func (job *Job) success() {
	log.WithFields(log.Fields{"Job": job.JobName, "Retries": job.CurrentRetry}).Info("successful")
	job.CurrentRetry = 0

	if job.RegularTimer > 0 {
		timeTaken := time.Duration(time.Now().UnixNano()-job.lastSuccess) - job.RegularTimer
		job.lastSuccess = time.Now().UnixNano()
		toWait := job.RegularTimer - timeTaken
		go job.SendTriggerWithDelay(toWait)
	}

	go job.triggerNextJob()
}

//Stop stops a job it will exit after if has finished if currently running (this may take a while!) or exit immediatly if waiting
func (job *Job) Stop() {
	log.WithFields(log.Fields{"Job": job.JobName}).Info("Stopped externally")
	job.stop <- true
	<-job.stopAnswer
}

func (job *Job) fail() {
	log.WithFields(log.Fields{"Job": job.JobName, "Retries": job.CurrentRetry}).Error("Failed")
	job.stop <- true
	<-job.stopAnswer
}

func (job *Job) finish() {
	job.Status = statusStopped
	job.queue.Wg.Done()
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
