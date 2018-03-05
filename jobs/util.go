package jobs

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

func LoadJobFromFile(file *os.File) (*Job, error) {
	var job = newJob()
	jsonParser := json.NewDecoder(file)
	err := jsonParser.Decode(job)
	if err != nil {
		return nil, err
	}
	job.retrieveAndStorePassword()
	job.RegularTimer *= time.Second
	job.RetryTimer *= time.Second
	return job, nil
}

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
