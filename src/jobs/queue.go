package jobs

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"sync"

	log "github.com/Sirupsen/logrus"
)

//JobQueue managing the jobs
type JobQueue struct {
	Jobs      map[string]*Job `json:"Jobs"`
	Triggers  map[string]([]*TimedTrigger)
	Wg        *sync.WaitGroup
	Directory string
}

//StartQueue starts all the jobs in the directory
func (queue *JobQueue) StartQueue() error {
	//TODO reimplement
	//println("Searching jobs")
	//jobs, err := FindJobs(path.Join(queue.Directory, "jobs"))
	//if err != nil {
	//	return err
	//}
	//
	//println("Searching triggers")
	//triggers, err := FindTriggers(path.Join(queue.Directory, "triggers"))
	//if err != nil {
	//	return err
	//}
	//
	////enter all jobs into the queue
	//for _, j := range jobs {
	//	if _, ok := queue.Jobs[j.JobName]; ok {
	//		return errors.New("Two jobs with same name: " + j.JobName)
	//	}
	//	queue.Jobs[j.JobName] = j
	//}
	//// NextJob resolution
	//for _, j := range queue.Jobs {
	//	for _, n := range j.NextJobs {
	//		nj, ok := queue.Jobs[n]
	//		if !ok {
	//			return errors.New("NextJob " + n + "not found for Job" + j.JobName)
	//		}
	//		j.nextTriggers = append(j.nextTriggers, nj)
	//	}
	//}
	//
	////enter all timed triggers into the queue
	//for _, t := range triggers {
	//	var ok bool
	//	t.ToTrigger, ok = queue.Jobs[t.JobToTrigger]
	//	if !ok {
	//		return errors.New("JobToTrigger not found:" + t.JobToTrigger)
	//	}
	//	queue.Triggers[t.JobToTrigger] = append(queue.Triggers[t.JobToTrigger], t)
	//}
	////start trigger loops
	//for _, trs := range queue.Triggers {
	//	for _, t := range trs {
	//		go t.loop()
	//	}
	//}
	return nil
}

func (queue *JobQueue) RestartJob(name string) error {
	triggers, ok := queue.Triggers[name]
	if !ok {
		return errors.New("No such Job")
	}
	for _, tr := range triggers {
		tr.Kill <- 0
		<-tr.Kill
	}
	err := queue.ReloadJob(name)
	if err != nil {
		return err
	}
	for _, tr := range triggers {
		go tr.loop()
	}
	return nil
}

func (queue *JobQueue) WaitForAllJobs() error {
	select {}
}

//StopJob stops all timed trigger for the job with this name
func (queue *JobQueue) StopJob(name string) error {
	triggers, ok := queue.Triggers[name]
	if !ok {
		return errors.New("No such Job")
	}
	for _, tr := range triggers {
		tr.Kill <- 0
		<-tr.Kill
	}
	return nil
}

//RemoveJob stops the job and then removes it from the queue
func (queue *JobQueue) RemoveJob(name string) error {
	err := queue.StopJob(name)
	if err != nil {
		return err
	}
	delete(queue.Jobs, name)
	delete(queue.Triggers, name)
	return nil
}

//TriggerJob triggers the job with the extern trigger so it doesnt trigger itself afterwards
func (queue *JobQueue) TriggerJob(name string) error {
	job, ok := queue.Jobs[name]
	if !ok {
		return errors.New("No such Job")
	}
	job.run()
	return nil
}

//StopAllJobs can take a long time depending on the jobs
func (queue *JobQueue) StopAllJobs() {
	for _, triggers := range queue.Triggers {
		for _, tr := range triggers {
			tr.Kill <- 0
			<-tr.Kill
		}
	}
}

//ReloadJob reloads the file (with all changes made to it) and replaces the old job with the new one.
func (queue *JobQueue) ReloadJob(name string) error {
	oldJob, ok := queue.Jobs[name]

	if !ok {
		return errors.New("No such job")
	}

	files, err := ioutil.ReadDir(path.Join(queue.Directory, "jobs"))
	if err != nil {
		log.Error("Error opening the directory: " + err.Error())
	}

	for _, f := range files {
		if f.IsDir() || !(f.Name() == name+".json") {
			continue
		}

		file, err := os.Open(path.Join(queue.Directory, f.Name()))
		if err != nil {
			log.WithFields(log.Fields{"File": f.Name(), "Error": err.Error()}).Warning("File error")
			continue
		}

		job, err := LoadJobFromFile(file)
		if err != nil {
			log.WithFields(log.Fields{"File": f.Name(), "Error": err.Error()}).Warning("Decoding error")
			continue
		}

		//job file found and parsed
		if job.JobName != oldJob.JobName {
			return errors.New("Name change not allowed")
		}
		queue.replaceJob(job, oldJob)

		return nil
	}
	log.WithFields(log.Fields{"Job": name}).Warning("No file for job")
	return errors.New("File could not be found")
}

func (queue *JobQueue) replaceJob(newJob, oldJob *Job) {
	//assume checking before that call
	triggers := queue.Triggers[oldJob.JobName]

	//patch job into all Triggers
	for _, tr := range triggers {
		tr.Lock.Lock()
		//tr.ToTrigger = newJob
		tr.Lock.Unlock()
	}

	queue.Jobs[newJob.JobName] = newJob
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
	return &JobQueue{Wg: new(sync.WaitGroup), Directory: path, Jobs: make(map[string]*Job), Triggers: make(map[string][]*TimedTrigger)}, nil
}
