package jobs

import "sync"

//JobQueue managing the jobs
type JobQueue struct {
	Jobs []*Job `json:"Jobs"`
	Wg   *sync.WaitGroup
}

func (queue *JobQueue) AddJobs(jobs []*Job) {
	for _, job := range jobs {
		queue.Wg.Add(1)
		go job.loop(queue.Wg)
	}
	queue.Jobs = append(queue.Jobs, jobs...)
}

func NewJobQueue() *JobQueue {
	return &JobQueue{Wg: new(sync.WaitGroup), Jobs: make([]*Job, 0)}
}
