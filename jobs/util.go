package jobs

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func FindJobs(path string) []*Job {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		println("error opening the directory: " + err.Error())
	}

	jobs := make([]*Job, 0)

	for _, f := range files {
		if f.IsDir() || !(strings.HasSuffix(f.Name(), ".json")) {
			continue
		}

		file, err := os.Open(f.Name())
		if err != nil {
			println("error opening the file " + f.Name() + " : " + err.Error())
			continue
		}

		var job = Job{}
		jsonParser := json.NewDecoder(file)
		err = jsonParser.Decode(&job)
		if err != nil {
			println("error decoding the file " + f.Name() + " : " + err.Error())
			continue
		}
		job.retrieveAndStorePassword()
		job.RegularTimer *= time.Second
		job.RetryTimer *= time.Second
		jobs = append(jobs, &job)
	}

	return jobs
}
