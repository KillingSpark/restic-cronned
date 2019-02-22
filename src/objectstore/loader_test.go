package objectstore_test

import (
	"encoding/json"
	"github.com/killingspark/restic-cronned/src/jobs"
	"github.com/killingspark/restic-cronned/src/objectstore"
	"github.com/killingspark/restic-cronned/src/triggers"
	"io/ioutil"
	"strings"
	"testing"
)

func registerTypes(s *objectstore.ObjectStore) {
	s.RegisterTriggerableType("Job", func(raw json.RawMessage) (objectstore.TriggerableDescription, error) {
		desc := &jobs.JobDescription{}
		err := json.Unmarshal(raw, desc)
		if err != nil {
			return nil, err
		}
		return desc, nil
	})

	s.RegisterTriggerableType("Simple", func(raw json.RawMessage) (objectstore.TriggerableDescription, error) {
		desc := &triggers.SimpleTriggerableDescription{}
		err := json.Unmarshal(raw, desc)
		if err != nil {
			return nil, err
		}
		return desc, nil
	})
	s.RegisterTriggerableType("Retry", func(raw json.RawMessage) (objectstore.TriggerableDescription, error) {
		desc := &triggers.RetryTriggerableDescription{}
		err := json.Unmarshal(raw, desc)
		if err != nil {
			return nil, err
		}
		return desc, nil
	})

	s.RegisterTriggererType("Simple", func(raw json.RawMessage) (objectstore.TriggererDescription, error) {
		desc := &triggers.SimpleTriggererDescription{}
		err := json.Unmarshal(raw, desc)
		if err != nil {
			return nil, err
		}
		return desc, nil
	})
	s.RegisterTriggererType("Retry", func(raw json.RawMessage) (objectstore.TriggererDescription, error) {
		desc := &triggers.RetryTriggererDescription{}
		err := json.Unmarshal(raw, desc)
		if err != nil {
			return nil, err
		}
		return desc, nil
	})

	s.RegisterTriggererType("Timer", func(raw json.RawMessage) (objectstore.TriggererDescription, error) {
		desc := &triggers.TimedTriggerDescription{}
		err := json.Unmarshal(raw, desc)
		if err != nil {
			return nil, err
		}
		return desc, nil
	})
}

func TestMyShit(t *testing.T) {
	ost := objectstore.NewObjectStore()
	registerTypes(ost)

	err := ost.LoadAllObjects("./testfiles/")
	if err != nil {
		t.Error(err.Error())
		t.Fail()
	}

	if _, ok := ost.Triggerers["simple"]; !ok {
		t.Error("Test triggerer not found")
		t.Fail()
	}

	if _, ok := ost.Triggerables["backup"]; !ok {
		t.Error("Test job not found")
		t.Fail()
	}

	marshflow, err := ioutil.ReadFile("./testfiles/my.flow")
	if err != nil {
		t.Error(err.Error())
		t.Fail()
	}

	ff := &objectstore.FlowForest{}
	err = ff.Load(marshflow)
	if err != nil {
		t.Error(err.Error())
		t.Fail()
	}

	root, err := ff.Build("Flow1", ost)
	if err != nil {
		t.Error(err.Error())
		t.Fail()
	}

	if !strings.HasSuffix(root.ID(), "__parsimple") {
		t.Error("Didnt load triggerer name correctly: " + root.ID())
		t.Fail()
	}

	//check names/ids
	parshot := root.(*triggers.SimpleTrigger)
	if !parshot.Parallel {
		t.Error("Didnt load Parallel attribute correctly: " + parshot.ID())
		t.Fail()
	}
	for _, child := range parshot.Targets {
		if !strings.HasSuffix(child.ID(), "___simple") {
			t.Error("Didnt load middle-triggerer name correctly: " + child.ID())
			t.Fail()
		}
		childSimple := child.(*triggers.SimpleTrigger)
		for _, job := range childSimple.Targets {
			if !strings.HasSuffix(job.ID(), "___backup") {
				t.Error("Didnt load backup name correctly: " + job.ID())
				t.Fail()
			}
		}
	}

	runs := 10
	for i := 0; i < runs; i++ {
		root.Run(nil)
	}

	for _, child := range parshot.Targets {
		childSimple := child.(*triggers.SimpleTrigger)
		if childSimple.TriggerCounter != runs {
			t.Errorf("Middletrigger didnt run right amount of times: %d", childSimple.TriggerCounter)
			t.Fail()
		}
		for _, child2 := range childSimple.Targets {
			job := child2.(*jobs.Job)
			if job.TriggerCounter != runs {
				t.Errorf("Backup %s didnt run right amount of times: %d", job.JobName, job.TriggerCounter)
				t.Fail()
			}
		}
	}
}
