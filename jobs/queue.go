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

//StartQueue starts all the jobs in the directory
func (queue *JobQueue) StartQueue() {
	jobs, err := FindJobs(queue.Directory)
	if err != nil {
		println(err.Error())
		return
	}
	queue.AddJobs(jobs)
}

//WaitForAllJobs does what it says it does
func (queue *JobQueue) WaitForAllJobs() {
	queue.Wg.Wait()
}

//StopJob stops the job with this name
func (queue *JobQueue) StopJob(name string) error {
	job := queue.findJob(name)
	if job != nil {
		job.Stop()
	} else {
		return errors.New("No such Job")
	}
	return nil
}

//RestartJob restarts the job with this name if it is present and in the "stopped" State
func (queue *JobQueue) RestartJob(name string) error {
	job := queue.findJob(name)
	if job != nil && job.Status == statusStopped {
		job.Status = statusReady
		err := queue.startJob(job)
		if err != nil {
			return err
		}
	} else {
		return errors.New("No such Job")
	}
	return nil
}

//StopAllJobs can take a long time depending on the jobs
func (queue *JobQueue) StopAllJobs() {
	for _, job := range queue.Jobs {
		job.Stop()
	}
}

//ReloadJob reloads the file (with all changes made to it) and replaces the old job with the new one.
//the old job is stopped (and waited for until stopped) before the new job is started
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
		queue.replaceJob(job, oldJob)
		queue.RestartJob(job.JobName)
		return nil
	}
	log.WithFields(log.Fields{"Job": name}).Warning("No file for job")
	return errors.New("File could not be found")
}

func (queue *JobQueue) replaceJob(newJob, oldJob *Job) {
	oldJob.Stop()
	*oldJob = *newJob
}

func (queue *JobQueue) findJob(name string) *Job {
	for _, job := range queue.Jobs {
		if job.JobName == name {
			return job
		}
	}
	return nil
}

func (queue *JobQueue) startJob(job *Job) error {
	if job.Status != statusReady {
		return errors.New("Illegal state")
	}
	queue.Wg.Add(1)
	job.start(queue)
	return nil
}

//JobExists Check if this job is in the queue
func (queue *JobQueue) JobExists(name string) bool {
	return queue.findJob(name) != nil
}

//AddJobs adds the jobs to its list and starts them
func (queue *JobQueue) AddJobs(jobs []*Job) {
	for _, job := range jobs {
		if oldJob := queue.findJob(job.JobName); oldJob != nil {
			queue.replaceJob(job, oldJob)
		} else {
			queue.Jobs = append(queue.Jobs, job)
		}
		queue.startJob(job)
	}
}

//NewJobQueue creates a new JobQueue for the given directory
func NewJobQueue(path string) (*JobQueue, error) {
	var err error
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("Can not open" + path)
	}
	s, err := f.Stat()
	if err != nil {
		return nil, errors.New("Can not open" + path)
	}
	if dir := s.IsDir(); !dir {
		return nil, errors.New(path + " is no directory")
	}
	return &JobQueue{Wg: new(sync.WaitGroup), Directory: path, Jobs: make([]*Job, 0)}, nil
}
