package jobs

import (
	"encoding/json"
	"errors"
	"github.com/robfig/cron"
	"io/ioutil"
	"os"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type JobDescription struct {
	JobName  string   `json:"JobName"`
	NextJobs []string `json:"NextJobs"`

	Username        string           `json:"Username"`
	Service         string           `json:"Service"`
	ResticPath      string           `json:"ResticPath"`
	ResticArguments []string         `json:"ResticArguments"`
	Preconditions   JobPreconditions `json:"Preconditions"`
}

type TimedTriggerDescription struct {
	JobToTrigger    string `json:"JobToTrigger"`
	RegularTimer    string `json:"regularTimer"`
	RetryTimer      string `json:"retryTimer"`
	WaitGranularity string `json:"waitGranularity"`

	MaxFailedRetries int `json:"maxFailedRetries"`

	CheckPrecondsEvery    int `json:"CheckPrecondsEvery"`
	CheckPrecondsMaxTimes int `json:"CheckPrecondsMaxTimes"`
}

//LoadJobFromFile loads  job from a file
func LoadJobFromFile(file *os.File) (*Job, error) {
	var jobDesc = &JobDescription{}
	jsonParser := json.NewDecoder(file)
	err := jsonParser.Decode(jobDesc)
	if err != nil {
		return nil, err
	}

	job := &Job{}
	job.JobName = jobDesc.JobName
	job.ResticPath = jobDesc.ResticPath
	job.ResticArguments = jobDesc.ResticArguments
	job.Username = jobDesc.Username
	job.Service = jobDesc.Service
	job.Preconditions = jobDesc.Preconditions
	job.NextJobs = jobDesc.NextJobs //need to be resolved by the queue
	job.retrieveAndStorePassword()

	return job, nil
}

func LoadTimedTriggerFromFile(file *os.File) (*TimedTrigger, error) {
	var desc = &TimedTriggerDescription{}
	jsonParser := json.NewDecoder(file)
	err := jsonParser.Decode(desc)
	if err != nil {
		return nil, err
	}

	tr := &TimedTrigger{}
	tr.JobToTrigger = desc.JobToTrigger

	if len(desc.RegularTimer) > 0 {
		tr.regTimerSchedule, err = cron.Parse(desc.RegularTimer)
		if err != nil {
			log.WithFields(log.Fields{"File": file.Name(), "Error": err.Error()}).Warning("Decoding error for the RegularTimer")
			return nil, err
		}
	}
	if len(desc.RetryTimer) > 0 {
		tr.retryTimerSchedule, err = cron.Parse(desc.RetryTimer)
		if err != nil {
			log.WithFields(log.Fields{"File": file.Name(), "Error": err.Error()}).Warning("Decoding error for the RetryTimer")
			return nil, err
		}
	}
	return tr, nil
}

//FindJobs loads all jobs from the path
func FindJobs(dirPath string) ([]*Job, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Error("Error opening the directory: " + err.Error())
		return make([]*Job, 0), errors.New(dirPath + " is no directory")
	}

	jobs := make([]*Job, 0)

	for _, f := range files {
		if f.IsDir() || !(strings.HasSuffix(f.Name(), ".json")) {
			continue
		}

		file, err := os.Open(path.Join(dirPath, f.Name()))
		if err != nil {
			log.WithFields(log.Fields{"File": f.Name(), "Error": err.Error()}).Warning("File error")
			continue
		}

		job, err := LoadJobFromFile(file)
		if err != nil {
			log.WithFields(log.Fields{"File": f.Name(), "Error": err.Error()}).Warning("Decoding error")
			continue
		}
		if job.JobName+".json" != f.Name() {
			return nil, errors.New("Job has wrong name: " + job.JobName + " Should be: " + file.Name()[:len(file.Name())-5])
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

//FindTriggers loads all triggers from the path
func FindTriggers(dirPath string) ([]*TimedTrigger, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Error("Error opening the directory: " + err.Error())
		return nil, errors.New(dirPath + " is no directory")
	}

	trs := make([]*TimedTrigger, 0)

	for _, f := range files {
		if f.IsDir() || !(strings.HasSuffix(f.Name(), ".json")) {
			continue
		}

		file, err := os.Open(path.Join(dirPath, f.Name()))
		if err != nil {
			log.WithFields(log.Fields{"File": f.Name(), "Error": err.Error()}).Warning("File error")
			continue
		}

		tr, err := LoadTimedTriggerFromFile(file)
		if err != nil {
			log.WithFields(log.Fields{"File": f.Name(), "Error": err.Error()}).Warning("Decoding error")
			continue
		} else {
			trs = append(trs, tr)
		}
	}

	return trs, nil
}
