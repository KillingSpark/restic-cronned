package objectstore_test

import (
	"github.com/killingspark/restic-cronned/src/jobs"
	"github.com/killingspark/restic-cronned/src/objectstore"
	"io/ioutil"
	"strings"
	"testing"
)

func TestMyShit(t *testing.T) {
	ost := &objectstore.ObjectStore{}

	err := ost.LoadAllObjects("./testfiles/")
	if err != nil {
		t.Error(err.Error())
		t.Fail()
	}

	if _, ok := ost.Triggerers["oneshot"]; !ok {
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

	if !strings.HasSuffix(root.ID(), "__parshot") {
		t.Error("Didnt load triggerer name correctly: " + root.ID())
		t.Fail()
	}

	//check names/ids
	parshot := root.(*objectstore.OneshotTrigger)
	if !parshot.Parallel {
		t.Error("Didnt load Parallel attribute correctly: " + parshot.ID())
		t.Fail()
	}
	for _, child := range parshot.Targets {
		if !strings.HasSuffix(child.ID(), "___oneshot") {
			t.Error("Didnt load middle-triggerer name correctly: " + child.ID())
			t.Fail()
		}
		childoneshot := child.(*objectstore.OneshotTrigger)
		for _, job := range childoneshot.Targets {
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
		childoneshot := child.(*objectstore.OneshotTrigger)
		if childoneshot.TriggerCounter != runs {
			t.Errorf("Middletrigger didnt run right amount of times: %d", childoneshot.TriggerCounter)
			t.Fail()
		}
		for _, child2 := range childoneshot.Targets {
			job := child2.(*jobs.Job)
			if job.TriggerCounter != runs {
				t.Errorf("Backup %s didnt run right amount of times: %d", job.JobName, job.TriggerCounter)
				t.Fail()
			}
		}
	}
}
