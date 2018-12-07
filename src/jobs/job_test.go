package jobs

import (
	"sync"
	"testing"
	"time"
)

type goTestSuite struct {
	test  *testing.T
	job1  *Job
	job2  *Job
	store TestStore
	wg    *sync.WaitGroup
}

type TestStore struct {
	jobs []*Job
}

func (store TestStore) FindJob(name string) (*Job, int) {
	switch name {
	case "A":
		return store.jobs[0], 0
	case "B":
		return store.jobs[1], 1
	default:
		return nil, 0
	}
}

func (suite *goTestSuite) SetupSuite() {
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

	suite.store = TestStore{[]*Job{suite.job1, suite.job2}}
	suite.wg = new(sync.WaitGroup)
}

func (suite *goTestSuite) TearDownSuite() {
}

func (suite *goTestSuite) SetupTest() {
}

func (suite *goTestSuite) TearDownTest() {
}

func (suite *goTestSuite) TestStati() {
	if suite.job1.Status != statusReady {
		suite.test.Error("Wrong state, should be ready")
	}
	suite.wg.Add(1)
	suite.job1.start(suite.store, func() { suite.wg.Done() })
	time.Sleep(1 * time.Millisecond)
	if suite.job1.Status != statusWaiting {
		suite.test.Error("Wrong state, should be working|waiting")
	}
	suite.job1.Stop()
	if suite.job1.Status != statusStopped {
		suite.test.Error("Wrong state, should be stopped")
	}
	var b = true
	go func() { suite.wg.Wait(); b = false }()
	time.Sleep(1 * time.Millisecond)
	if b {
		suite.test.Error("didnt release waitgroup")
	}
}

func (suite *goTestSuite) TestFollowupTrigger() {
	suite.wg.Add(1)
	suite.job1.start(suite.store, func() { suite.wg.Done() })
	suite.job2.start(suite.store, func() { suite.wg.Done() })
	time.Sleep(100 * time.Millisecond)
	suite.job1.SendTrigger(triggerIntern)
	time.Sleep(100 * time.Millisecond)
	if suite.job2.CurrentRetry != 1 {
		suite.test.Error("job2 wasnt triggered")
	}
}

func (suite *goTestSuite) TestFail() {
	suite.wg.Add(1)
	suite.job2.start(suite.store, func() { suite.wg.Done() })

	time.Sleep(100 * time.Millisecond)
	suite.job2.SendTrigger(triggerIntern)
	time.Sleep(100 * time.Millisecond)
	if suite.job2.CurrentRetry != 1 {
		suite.test.Error("didnt record failed try")
	}

	suite.job2.SendTrigger(triggerIntern)
	time.Sleep(100 * time.Millisecond)
	if suite.job2.CurrentRetry != 2 {
		suite.test.Error("didnt record failed try")
	}

	suite.job2.SendTrigger(triggerIntern)
	time.Sleep(100 * time.Millisecond)
	if suite.job2.Status != statusStopped {
		suite.test.Error("didnt stop after max fails" + suite.job2.Status)
	}

	var b = true
	go func() { suite.wg.Wait(); b = false }()
	time.Sleep(1 * time.Millisecond)
	if b {
		suite.test.Error("didnt release waitgroup")
	}
}

func TestGoTestSuite(t *testing.T) {
	tests := goTestSuite{test: t}
	tests.SetupSuite()
	tests.TestStati()
	tests.SetupSuite()
	tests.TestFail()
	tests.SetupSuite()
	tests.TestFollowupTrigger()
}
