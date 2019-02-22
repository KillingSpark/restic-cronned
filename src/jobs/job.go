package jobs

//Job a job to be run periodically
import (
	"bytes"
	"context"
	"github.com/killingspark/restic-cronned/src/objectstore"
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

//Job represents one job that will be run in a Queue
type Job struct {
	lock sync.Mutex //locks the run method so the job can only be run by one trigger at a time

	CheckPrecondsMaxTimes   int
	CheckPrecondsEveryMilli int

	//the progress of the running restic command. not working.
	Progress float64 `json:"progress"`

	//generic data from the config file
	JobName         string           `json:"JobName"`
	Username        string           `json:"Username"`
	Service         string           `json:"Service"`
	ResticPath      string           `json:"ResticPath"`
	ResticArguments []string         `json:"ResticArguments"`
	Preconditions   JobPreconditions `json:"Preconditions"`

	//retrieved from a system keyring
	password string

	TriggerCounter int
}

func (job *Job) Trigger(ctx context.Context) JobReturn {
	job.TriggerCounter++
	return job.run()
}

func (job *Job) ID() string {
	return job.JobName
}

func (job *Job) CheckPreconditions() bool {
	return job.Preconditions.CheckAll()
}

func (job *Job) Name() string {
	return job.JobName
}

func newJob() *Job {
	return &Job{
		Progress: 0,
	}
}

//JobReturn status returns from jobs
type JobReturn = objectstore.ReturnValue

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

func (job *Job) RetrieveAndStorePassword() {
	key, err := keyring.Get(job.Service, job.Username)
	if err != nil {
		log.WithFields(log.Fields{"Job": job.JobName, "User": job.Username, "Service": job.Service}).Warning("couldn't retrieve password.")
	} else {
		log.WithFields(log.Fields{"Job": job.JobName}).Info("retrieved password.")
		job.password = key
	}
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
	//Lock this func so multiple concurrent triggers do not run this job at the same time
	job.lock.Lock()
	defer job.lock.Unlock()

	if !job.waitPreconds() {
		return returnRetry
	}

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

func (job *Job) waitPreconds() bool {
	if job.CheckPrecondsMaxTimes > 0 {
		preconds := false
		for i := 0; !preconds && i < job.CheckPrecondsMaxTimes; i++ {
			preconds = job.CheckPreconditions()
			if !preconds {
				time.Sleep(time.Duration(job.CheckPrecondsEveryMilli) * time.Millisecond)
			}
		}
		return preconds
	}
	return true
}
