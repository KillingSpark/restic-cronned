package objectstore

import (
	"context"
	"github.com/killingspark/restic-cronned/src/jobs"
)

type OneshotTriggererDescription struct {
	Name string `Json:"Name"`
}

type OneshotTriggerableDescription struct {
	Name string `Json:"Name"`
}

func (d *OneshotTriggererDescription) ID() string {
	return d.Name
}

func (d *OneshotTriggererDescription) Instantiate(unique string) (Triggerer, error) {
	tr := &OneshotTrigger{}

	tr.Name = unique + "__" + d.Name

	return tr, nil
}

func (d *OneshotTriggerableDescription) ID() string {
	return d.Name
}

func (d *OneshotTriggerableDescription) Instantiate(unique string) (Triggerable, error) {
	tr := &OneshotTrigger{}

	tr.Name = unique + "__" + d.Name

	return tr, nil
}

type OneshotTrigger struct {
	Name           string
	Targets        []Triggerable
	TriggerCounter int
}

func (tt *OneshotTrigger) ID() string {
	return tt.Name
}

func (tt *OneshotTrigger) AddTarget(ta Triggerable) error {
	tt.Targets = append(tt.Targets, ta)
	return nil
}

func (tt *OneshotTrigger) Run(ctx context.Context) error {
	for _, t := range tt.Targets {
		t.Trigger(&ctx)
	}
	return nil
}

func (tt *OneshotTrigger) Trigger(ctx *context.Context) jobs.JobReturn {
	tt.TriggerCounter++
	var r ReturnValue
	for _, t := range tt.Targets {
		if x := t.Trigger(ctx); x > r {
			r = x
		}
	}
	return r
}
