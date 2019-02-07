package triggers

import (
	"context"
	"github.com/killingspark/restic-cronned/src/jobs"
	"github.com/killingspark/restic-cronned/src/objectstore"
	"math"
	"sync"
)

type OneshotTriggererDescription struct {
	Name     string `Json:"Name"`
	Parallel bool   `Json:"Parallel"`
}

type OneshotTriggerableDescription OneshotTriggererDescription

func (d *OneshotTriggererDescription) ID() string {
	return d.Name
}

func (d *OneshotTriggererDescription) Instantiate(unique string) (objectstore.Triggerer, error) {
	tr := &OneshotTrigger{}

	tr.Name = unique + "__" + d.Name
	tr.Parallel = d.Parallel

	return tr, nil
}

func (d *OneshotTriggerableDescription) ID() string {
	return d.Name
}

func (d *OneshotTriggerableDescription) Instantiate(unique string) (objectstore.Triggerable, error) {
	tr := &OneshotTrigger{}

	tr.Name = unique + "__" + d.Name
	tr.Parallel = d.Parallel

	return tr, nil
}

type OneshotTrigger struct {
	Name     string
	Parallel bool

	Targets        []objectstore.Triggerable
	TriggerCounter int
}

func (tt *OneshotTrigger) ID() string {
	return tt.Name
}

func (tt *OneshotTrigger) AddTarget(ta objectstore.Triggerable) error {
	tt.Targets = append(tt.Targets, ta)
	return nil
}

func (tt *OneshotTrigger) Run(ctx *context.Context) error {
	for _, t := range tt.Targets {
		t.Trigger(ctx)
	}
	return nil
}

func (tt *OneshotTrigger) triggerSeq(ctx *context.Context) jobs.JobReturn {
	tt.TriggerCounter++
	var r objectstore.ReturnValue
	for _, t := range tt.Targets {
		if x := t.Trigger(ctx); x > r {
			r = x
		}
	}
	return r
}

func (tt *OneshotTrigger) triggerPar(ctx *context.Context) jobs.JobReturn {
	tt.TriggerCounter++
	res := make([]objectstore.ReturnValue, len(tt.Targets))
	wg := sync.WaitGroup{}
	for i, t := range tt.Targets {
		idx := i
		trgt := t
		wg.Add(1)
		go func() {
			res[idx] = trgt.Trigger(ctx)
			wg.Done()
		}()
	}

	wg.Wait()

	//TODO result checking strategies
	r := objectstore.ReturnValue(math.MinInt64)
	for _, x := range res {
		if x > r {
			r = x
		}
	}
	return r
}

func (tt *OneshotTrigger) Trigger(ctx *context.Context) jobs.JobReturn {
	if tt.Parallel {
		return tt.triggerPar(ctx)
	} else {
		return tt.triggerSeq(ctx)
	}
}
