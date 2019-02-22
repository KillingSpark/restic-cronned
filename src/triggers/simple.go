package triggers

import (
	"context"
	"github.com/killingspark/restic-cronned/src/jobs"
	"github.com/killingspark/restic-cronned/src/objectstore"
	"math"
	"sync"
)

type SimpleTriggererDescription struct {
	Name     string `Json:"Name"`
	Parallel bool   `Json:"Parallel"`
}

type SimpleTriggerableDescription SimpleTriggererDescription

func (d *SimpleTriggererDescription) ID() string {
	return d.Name
}

func (d *SimpleTriggererDescription) Instantiate(unique string) (objectstore.Triggerer, error) {
	tr := &SimpleTrigger{}

	tr.Name = unique + "__" + d.Name
	tr.Parallel = d.Parallel

	return tr, nil
}

func (d *SimpleTriggerableDescription) ID() string {
	return d.Name
}

func (d *SimpleTriggerableDescription) Instantiate(unique string) (objectstore.Triggerable, error) {
	tr := &SimpleTrigger{}

	tr.Name = unique + "__" + d.Name
	tr.Parallel = d.Parallel

	return tr, nil
}

type SimpleTrigger struct {
	Name     string
	Parallel bool

	Targets        []objectstore.Triggerable
	TriggerCounter int
}

func (tt *SimpleTrigger) ID() string {
	return tt.Name
}

func (tt *SimpleTrigger) AddTarget(ta objectstore.Triggerable) error {
	tt.Targets = append(tt.Targets, ta)
	return nil
}

func (tt *SimpleTrigger) Run(ctx context.Context) error {
	tt.Trigger(ctx)
	return nil
}

func (tt *SimpleTrigger) triggerSeq(ctx context.Context) jobs.JobReturn {
	var r objectstore.ReturnValue
	for _, t := range tt.Targets {
		if x := t.Trigger(ctx); x > r {
			r = x
		}
	}
	return r
}

func (tt *SimpleTrigger) triggerPar(ctx context.Context) jobs.JobReturn {
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

func (tt *SimpleTrigger) Trigger(ctx context.Context) jobs.JobReturn {
	tt.TriggerCounter++
	if tt.Parallel {
		return tt.triggerPar(ctx)
	} else {
		return tt.triggerSeq(ctx)
	}
}
