package jobs

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

func FindJobs(path string) []*Job {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal("Error opening the directory: " + err.Error())
	}

	jobs := make([]*Job, 0)

	for _, f := range files {
		if f.IsDir() || !(strings.HasSuffix(f.Name(), ".json")) {
			continue
		}

		file, err := os.Open(f.Name())
		if err != nil {
			log.WithFields(log.Fields{"File": f.Name(), "Error": err.Error()}).Warning("File error")
			continue
		}

		var job = Job{}
		jsonParser := json.NewDecoder(file)
		err = jsonParser.Decode(&job)
		if err != nil {
			log.WithFields(log.Fields{"File": f.Name(), "Error": err.Error()}).Warning("Decodinf error")
			continue
		}
		job.retrieveAndStorePassword()
		job.RegularTimer *= time.Second
		job.RetryTimer *= time.Second
		jobs = append(jobs, &job)
	}

	return jobs
}
