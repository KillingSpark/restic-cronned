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

//LoadJobFromFile loads  job from a file
func LoadJobFromFile(file *os.File) (*Job, error) {
	var job = newJob()
	jsonParser := json.NewDecoder(file)
	err := jsonParser.Decode(job)
	if err != nil {
		return nil, err
	}
	job.retrieveAndStorePassword()
	if len(job.RegularTimer) > 0 {
		job.regTimerSchedule, err = cron.Parse(job.RegularTimer)
		if err != nil {
			log.WithFields(log.Fields{"Job": job.JobName, "Error": err.Error()}).Warning("Decoding error for the RegularTimer")
			return nil, err
		}
	}
	if len(job.RetryTimer) > 0 {
		job.retryTimerSchedule, err = cron.Parse(job.RetryTimer)
		if err != nil {
			log.WithFields(log.Fields{"Job": job.JobName, "Error": err.Error()}).Warning("Decoding error for the RetryTimer")
			return nil, err
		}
	}
	return job, nil
}

//FindJobs loads all jobs from the path
func FindJobs(dirPath string) ([]*Job, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Fatal("Error opening the directory: " + err.Error())
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
		} else {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}
