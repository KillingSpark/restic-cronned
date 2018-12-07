package jobs

import (
	"sync"
	"testing"
	"time"
)

type queueTestSuite struct {
	test  *testing.T
	job1  *Job
	job2  *Job
	queue *JobQueue
}

func (suite *queueTestSuite) SetupSuite() {
	suite.job1 = newJob()
	suite.job1.JobName = "A"
	suite.job1.JobNameToTrigger = "B"
	suite.job1.RegularTimer = -1
	suite.job1.RetryTimer = -1

	suite.job2 = newJob()
	suite.job2.JobName = "B"
	suite.job2.ResticArguments = []string{"aösdfhasödlk", "asdfwerrwefosdf"}
	suite.job2.RegularTimer = -1
	suite.job2.RetryTimer = -1
	suite.job2.MaxFailedRetries = 2

	suite.queue = &JobQueue{Wg: new(sync.WaitGroup), Jobs: make([]*Job, 0)}
}

func (suite *queueTestSuite) TearDownSuite() {
}

func (suite *queueTestSuite) SetupTest() {
}

func (suite *queueTestSuite) TearDownTest() {
}

func (suite *queueTestSuite) TestAdd() {
	suite.queue.AddJobs(suite.job1, suite.job2)
	time.Sleep(1 * time.Millisecond)
	for _, job := range suite.queue.Jobs {
		if job.Status != statusWaiting {
			suite.test.Error("Job not started: " + job.JobName)
		}
	}
}

func (suite *queueTestSuite) TestReplace() {
	suite.queue.AddJobs(suite.job1)
	suite.queue.replaceJob(suite.job2, suite.job1)
	if suite.queue.JobExists("A") {
		suite.test.Error("A is still in list")
	}
	if !suite.queue.JobExists("B") {
		suite.test.Error("B is not in list")
	}
}

func (suite *queueTestSuite) TestStop() {
	suite.queue.AddJobs(suite.job1)
	time.Sleep(1 * time.Millisecond)
	suite.queue.StopJob("A")
	time.Sleep(1 * time.Millisecond)
	if suite.job1.Status != statusStopped {
		suite.test.Error("Did not stop")
	}
}

func (suite *queueTestSuite) TestTrigger() {
	suite.queue.AddJobs(suite.job2)
	time.Sleep(1 * time.Millisecond)
	suite.queue.TriggerJob("B")
	time.Sleep(1 * time.Millisecond)
	if suite.job2.CurrentRetry != 1 {
		suite.test.Error("Did not trigger")
	}
}

func (suite *queueTestSuite) TestRestart() {
	suite.queue.AddJobs(suite.job1)
	time.Sleep(1 * time.Millisecond)
	suite.queue.StopJob("A")
	time.Sleep(1 * time.Millisecond)
	if suite.job1.Status != statusStopped {
		suite.test.Error("Did not stop")
	}
	suite.queue.RestartJob("A")
	time.Sleep(1 * time.Millisecond)
	if suite.job1.Status != statusWaiting {
		suite.test.Error("Did not restart")
	}
}

func (suite *queueTestSuite) TestStopAll() {
	suite.queue.AddJobs(suite.job1, suite.job2)
	time.Sleep(1 * time.Millisecond)
	suite.queue.StopAllJobs()
	time.Sleep(1 * time.Millisecond)
	for _, job := range suite.queue.Jobs {
		if job.Status != statusStopped {
			suite.test.Error("Job not stop: " + job.JobName)
		}
	}
}

func TestQueueTestSuite(t *testing.T) {
	tests := queueTestSuite{test: t}
	tests.SetupSuite()
	tests.TestAdd()
	tests.SetupSuite()
	tests.TestAdd()
	tests.SetupSuite()
	tests.TestAdd()
}
