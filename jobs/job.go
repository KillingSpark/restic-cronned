package jobs

//Job a job to be run periodically
import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
	keyring "github.com/zalando/go-keyring"
)

//Job represents one job that will be run in a Queue
type Job struct {
	//timers that get waited
	RegularTimer       string `json:"regularTimer"`
	regTimerSchedule   cron.Schedule
	RetryTimer         string `json:"retryTimer"`
	retryTimerSchedule cron.Schedule
	//Retry counter/limit
	CurrentRetry     int `json:"CurrentRetry"`
	MaxFailedRetries int `json:"maxFailedRetries"`
	//statemachine status
	Status JobStatus `json:"status"`
	//the progress of the running restic command. not working.
	Progress float64 `json:"progress"`
	//times set when the wait is started
	WaitStart time.Duration `json:"WaitStart"`
	WaitEnd   time.Duration `json:"WaitEnd"`

	//channels used for stopping the loop/answering to the caller
	stop       chan bool
	stopAnswer chan bool
	//channel to trigger the loop to run once
	trigger chan TriggerType
	//interface to the queue that lats you query for jobs. used for triggerNext
	jobstore JobStore
	//generic data from the config files
	JobNameToTrigger      string `json:"NextJob"`
	JobName               string `json:"JobName"`
	Username              string `json:"Username"`
	Service               string `json:"Service"`
	password              string
	ResticPath            string           `json:"ResticPath"`
	ResticArguments       []string         `json:"ResticArguments"`
	Preconditions         JobPreconditions `json:"Preconditions"`
	CheckPrecondsEvery    int              `json:"CheckPrecondsEvery"`
	CheckPrecondsMaxTimes int              `json:"CheckPrecondsMaxTimes"`
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
		log.WithFields(log.Fields{"Job": job.JobName}).Info("retrieved password.")
		job.password = key
	}
}

//SendTrigger makes the job  run immediatly (if waiting or immediatly again if working right now)
func (job *Job) SendTrigger(trigType TriggerType) {
	if job.Status == statusWaiting || job.Status == statusWorking {
		log.WithFields(log.Fields{"Job": job.JobName}).Info("Trigger try")
		job.trigger <- trigType
	}
}

//SendTriggerWithDelay makes the job run after "dur" nanoseconds
func (job *Job) SendTriggerWithDelay(dur time.Duration) {
	if dur < 0 {
		//ignore for example jobs that shouldnt be run
		log.WithFields(log.Fields{"Job": job.JobName, "Duration": dur}).Info("Ignore trigger with negative duration")
		return
	}
	if dur > 0 {
		//for frontends
		job.WaitStart = time.Duration(time.Now().UnixNano())
		job.WaitEnd = job.WaitStart + dur

		sleepUntil := time.Now().Add(dur).Round(0)

		log.WithFields(log.Fields{"Job": job.JobName, "Time": sleepUntil.String()}).Info("Trigger scheduled")

		for time.Now().Round(0).Before(sleepUntil) {
			time.Sleep(10 * time.Second)
		}
	}
	job.SendTrigger(triggerIntern)
}

func (job *Job) triggerNextJob() {
	if len(job.JobNameToTrigger) <= 0 {
		log.WithFields(log.Fields{"Job": job.JobName}).Info("No follow up job")
		return
	}
	toTrigger, _ := job.jobstore.FindJob(job.JobNameToTrigger)
	if toTrigger != nil {
		toTrigger.SendTrigger(triggerIntern)
	} else {
		log.WithFields(log.Fields{"Job": job.JobName, "NextJob": job.JobNameToTrigger}).Warning("could not find next Job")
	}
}

func (job *Job) loop(finishCallback func()) {
	defer job.finish(finishCallback)
	for {
		var retrigger = false
		job.Status = statusWaiting
		log.WithFields(log.Fields{"Job": job.JobName}).Info("Await trigger/stop")
		select {
		case trigType := <-job.trigger:
			log.WithFields(log.Fields{"Job": job.JobName}).Info("Trigger received")
			switch trigType {
			case triggerIntern:
				retrigger = true
			case triggerExtern:
				retrigger = false
			}
		case <-job.stop:
			job.stopAnswer <- true
			return
		}

		preconds := true
		for i := 0; !preconds && i < job.CheckPrecondsMaxTimes; i++ {
			preconds = job.Preconditions.CheckAll()
			if !preconds {
				time.Sleep(time.Duration(job.CheckPrecondsEvery) * time.Second)
			}
		}
		if !preconds {
			job.failPreconds()
			continue
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
			job.success(retrigger)
			break
		case returnStop:
			job.fail()
			return
		}
	}
}

func (job *Job) start(store JobStore, finishCallback func()) {
	job.jobstore = store
	go job.loop(finishCallback)
	job.Status = statusWaiting
	go job.SendTriggerWithDelay(job.durationTillNextRegularTrigger())
}

var jobTriggerPersistDir = path.Join(os.ExpandEnv("$HOME"), ".local/share/restic-cronned")

type persistedJobTrigger struct {
	NextTrigger time.Time
}

func (job *Job) durationTillNextRegularTrigger() time.Duration {
	dur := time.Duration(-1)

	if job.regTimerSchedule != nil {
		dur = job.regTimerSchedule.Next(time.Now()).Sub(time.Now())
	}
	return dur
}

func (job *Job) durationTillNextRetryTrigger() time.Duration {
	dur := time.Duration(-1)

	if job.retryTimerSchedule != nil {
		dur = job.retryTimerSchedule.Next(time.Now()).Sub(time.Now())
	}
	return dur
}

func (job *Job) retry() {
	log.WithFields(log.Fields{"Job": job.JobName, "Retries": job.CurrentRetry}).Info("Start next retry")
	job.CurrentRetry++
	go job.SendTriggerWithDelay(job.durationTillNextRetryTrigger())
}

func (job *Job) success(retrigger bool) {
	log.WithFields(log.Fields{"Job": job.JobName, "Retries": job.CurrentRetry}).Info("successful")
	job.CurrentRetry = 0

	if retrigger {
		if job.regTimerSchedule != nil {
			go job.SendTriggerWithDelay(job.durationTillNextRegularTrigger())
		}
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
	log.WithFields(log.Fields{"Job": job.JobName, "Retries": job.CurrentRetry}).Error("Failed. Will try again at next regular trigger")
}

func (job *Job) failPreconds() {
	log.WithFields(log.Fields{"Job": job.JobName}).Error("Failed Preconditions. Will try again at next regular trigger")
	go job.SendTriggerWithDelay(job.durationTillNextRegularTrigger())
}

func (job *Job) finish(finishCallback func()) {
	log.WithFields(log.Fields{"Job": job.JobName}).Error("Finished")
	job.Status = statusStopped
	finishCallback()
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
	log.WithFields(log.Fields{"Job": job.JobName}).Info("Run restic")
	err := cmd.Run()
	log.WithFields(log.Fields{"Job": job.JobName}).Info("Finished running restic")

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
