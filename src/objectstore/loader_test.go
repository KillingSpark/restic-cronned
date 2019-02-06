package objectstore_test

import (
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

	if _, ok := ost.Triggerers["timer"]; !ok {
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

	timer, err := ff.Build("Flow1", ost)
	if err != nil {
		t.Error(err.Error())
		t.Fail()
	}

	if !strings.HasPrefix(timer.ID(), "timer") {
		t.Error("Didnt load timer name correctly: " + timer.ID())
		t.Fail()
	}

	timer.Run(nil)
}
