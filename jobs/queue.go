package jobs

import (
	"errors"
	"io/ioutil"
	"os"
	"sync"

	log "github.com/Sirupsen/logrus"
)

//JobQueue managing the jobs
type JobQueue struct {
	Jobs      []*Job `json:"Jobs"`
	Wg        *sync.WaitGroup
	Directory string
}

func (queue *JobQueue) StopJob(name string) error {
	job := queue.findJob(name)
	if job != nil {
		job.Stop()
	} else {
		return errors.New("No such Job")
	}
	return nil
}

func (queue *JobQueue) RestartJob(name string) error {
	job := queue.findJob(name)
	if job != nil && job.Status == statusFinished {
		queue.startJob(job)
	} else {
		return errors.New("No such Job")
	}
	return nil
}

func (queue *JobQueue) ReloadJob(name string) error {
	oldJob := queue.findJob(name)

	if oldJob == nil {
		return errors.New("No such job")
	}

	files, err := ioutil.ReadDir(queue.Directory)
	if err != nil {
		log.Fatal("Error opening the directory: " + err.Error())
	}

	for _, f := range files {
		if f.IsDir() || !(f.Name() == name+".json") {
			continue
		}

		file, err := os.Open(f.Name())
		if err != nil {
			log.WithFields(log.Fields{"File": f.Name(), "Error": err.Error()}).Warning("File error")
			continue
		}

		job, err := LoadJobFromFile(file)
		if err != nil {
			log.WithFields(log.Fields{"File": f.Name(), "Error": err.Error()}).Warning("Decoding error")
			continue
		}
		oldJob.Stop()
		*oldJob = *job
		queue.startJob(oldJob)
		return nil
	}
	log.WithFields(log.Fields{"Job": name}).Warning("No file for job")
	return errors.New("File could not be found")
}

func (queue *JobQueue) findJob(name string) *Job {
	for _, job := range queue.Jobs {
		if job.JobName == name {
			return job
		}
	}
	return nil
}

func (queue *JobQueue) startJob(job *Job) {
	queue.Wg.Add(1)
	go job.loop(queue.Wg)
}

func (queue *JobQueue) JobExists(name string) bool {
	return queue.findJob(name) != nil
}

func (queue *JobQueue) AddJobs(jobs []*Job) {
	for _, job := range jobs {
		if j := queue.findJob(job.JobName); j != nil {
			queue.StopJob(j.JobName)
		}
		queue.startJob(job)
	}
	queue.Jobs = append(queue.Jobs, jobs...)
}

func NewJobQueue() *JobQueue {
	return &JobQueue{Wg: new(sync.WaitGroup), Jobs: make([]*Job, 0)}
}
