package jobs

import (
	"encoding/json"
	"errors"
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
	job.RetrieveAndStorePassword()

	return job, nil
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
