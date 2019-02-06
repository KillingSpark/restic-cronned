package objectstore

import (
	"github.com/killingspark/restic-cronned/src/jobs"
)

type JobDescription struct {
	JobName  string   `json:"Name"`
	NextJobs []string `json:"NextJobs"`

	Username        string                `json:"Username"`
	Service         string                `json:"Service"`
	ResticPath      string                `json:"ResticPath"`
	ResticArguments []string              `json:"ResticArguments"`
	Preconditions   jobs.JobPreconditions `json:"Preconditions"`
}

func (jd *JobDescription) ID() string {
	return jd.JobName
}

func (jd *JobDescription) Instantiate(unique string) (Triggerable, error) {
	job := &jobs.Job{}
	job.JobName = unique + "__" + jd.JobName
	job.ResticPath = jd.ResticPath
	job.ResticArguments = jd.ResticArguments
	job.Username = jd.Username
	job.Service = jd.Service
	job.Preconditions = jd.Preconditions
	job.NextJobs = jd.NextJobs //need to be resolved by the queue
	job.RetrieveAndStorePassword()

	return job, nil
}
