package jobs

import (
	"github.com/killingspark/restic-cronned/src/objectstore"
)

type JobDescription struct {
	JobName  string   `json:"Name"`
	NextJobs []string `json:"NextJobs"`

	Username        string   `json:"Username"`
	Service         string   `json:"Service"`
	ResticPath      string   `json:"ResticPath"`
	ResticArguments []string `json:"ResticArguments"`

	CheckPrecondsMaxTimes   int              `json:"CheckPrecondsMaxTimes"`
	CheckPrecondsEveryMilli int              `json:"CheckPrecondsEveryMilli"`
	Preconditions           JobPreconditions `json:"Preconditions"`
}

func (jd *JobDescription) ID() string {
	return jd.JobName
}

func (jd *JobDescription) Instantiate(unique string) (objectstore.Triggerable, error) {
	job := &Job{}
	job.JobName = unique + "__" + jd.JobName
	job.ResticPath = jd.ResticPath
	job.ResticArguments = jd.ResticArguments
	job.Username = jd.Username
	job.Service = jd.Service

	job.Preconditions = jd.Preconditions
	job.CheckPrecondsMaxTimes = jd.CheckPrecondsMaxTimes
	job.CheckPrecondsEveryMilli = jd.CheckPrecondsEveryMilli
	job.RetrieveAndStorePassword()

	return job, nil
}
