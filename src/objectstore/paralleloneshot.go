package objectstore

import (
	"context"
	"github.com/killingspark/restic-cronned/src/jobs"
	"math"
	"sync"
)

type ParallelOneshotTriggererDescription struct {
	Name string `Json:"Name"`
}

type ParallelOneshotTriggerableDescription struct {
	Name string `Json:"Name"`
}

func (d *ParallelOneshotTriggererDescription) ID() string {
	return d.Name
}

func (d *ParallelOneshotTriggererDescription) Instantiate(unique string) (Triggerer, error) {
	tr := &ParallelOneshotTrigger{}

	tr.Name = unique + "__" + d.Name

	return tr, nil
}

func (d *ParallelOneshotTriggerableDescription) ID() string {
	return d.Name
}

func (d *ParallelOneshotTriggerableDescription) Instantiate(unique string) (Triggerable, error) {
	tr := &ParallelOneshotTrigger{}

	tr.Name = unique + "__" + d.Name

	return tr, nil
}

type ParallelOneshotTrigger struct {
	Name           string
	Targets        []Triggerable
	TriggerCounter int
}

func (tt *ParallelOneshotTrigger) ID() string {
	return tt.Name
}

func (tt *ParallelOneshotTrigger) AddTarget(ta Triggerable) error {
	tt.Targets = append(tt.Targets, ta)
	return nil
}

func (tt *ParallelOneshotTrigger) Run(ctx context.Context) error {
	for _, t := range tt.Targets {
		t.Trigger(&ctx)
	}
	return nil
}

func (tt *ParallelOneshotTrigger) Trigger(ctx *context.Context) jobs.JobReturn {
	tt.TriggerCounter++
	res := make([]ReturnValue, len(tt.Targets))
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

	r := ReturnValue(math.MinInt64)
	for _, x := range res {
		if x > r {
			r = x
		}
	}
	return r
}
