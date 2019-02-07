package objectstore

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
	"time"
)

const (
	returnStop  ReturnValue = 0
	returnOk    ReturnValue = 1
	returnRetry ReturnValue = 2
)

type RetryTriggererDescription struct {
	Name            string `Json:"Name"`
	RetryTimer      string `json:"Timer"`
	WaitGranularity int    `json:"WaitGranularity"`

	MaxFailedRetries int `json:"MaxFailedRetries"`
}

type RetryTriggerableDescription RetryTriggererDescription

func (d *RetryTriggererDescription) ID() string {
	return d.Name
}

func (d *RetryTriggererDescription) Instantiate(unique string) (Triggerer, error) {
	tr := &RetryTrigger{}
	var err error

	tr.Name = unique + "__" + d.Name
	tr.MaxFailedRetries = d.MaxFailedRetries
	tr.waitGran = time.Duration(d.WaitGranularity) * time.Millisecond

	tr.Schedule, err = cron.Parse(d.RetryTimer)
	if err != nil {
		log.WithFields(log.Fields{"ID": d.ID, "Error": err.Error()}).Warning("Decoding error for the RetryTimer")
		return nil, err
	}
	return tr, nil
}

func (d *RetryTriggerableDescription) ID() string {
	return d.Name
}

func (d *RetryTriggerableDescription) Instantiate(unique string) (Triggerable, error) {
	tr := &RetryTrigger{}
	var err error

	tr.Name = unique + "__" + d.Name
	tr.MaxFailedRetries = d.MaxFailedRetries
	tr.waitGran = time.Duration(d.WaitGranularity) * time.Millisecond

	tr.Schedule, err = cron.Parse(d.RetryTimer)
	if err != nil {
		log.WithFields(log.Fields{"ID": d.ID, "Error": err.Error()}).Warning("Decoding error for the RetryTimer")
		return nil, err
	}
	return tr, nil
}

type RetryTrigger struct {
	Name    string
	Targets []Triggerable

	Schedule cron.Schedule
	waitGran time.Duration

	CurrentRetry     int
	MaxFailedRetries int
}

func (tt *RetryTrigger) ID() string {
	return tt.Name
}

func (tt *RetryTrigger) AddTarget(ta Triggerable) error {
	tt.Targets = append(tt.Targets, ta)
	return nil
}

func (tt *RetryTrigger) Run(ctx context.Context) error {
	tt.loop()
	return nil
}

func (tt *RetryTrigger) Trigger(ctx *context.Context) ReturnValue {
	return tt.loop()
}

//NextTriggerWithDelay makes the job run after "dur" nanoseconds
func (tt *RetryTrigger) NextTriggerWithDelay(dur time.Duration) {
	if dur < 0 {
		//ignore for example jobs that shouldnt be run
		log.WithFields(log.Fields{"Trigger": tt.ID(), "Duration": dur}).Info("Ignore trigger with negative duration")
	}
	if dur > 0 {
		sleepUntil := time.Now().Add(dur).Round(0)

		log.WithFields(log.Fields{"Trigger": tt.ID(), "Time": sleepUntil.String()}).Info("Timed trigger scheduled")

		for time.Now().Round(0).Before(sleepUntil) {
			if tt.waitGran > 0 {
				time.Sleep(tt.waitGran)
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	}
	if dur == 0 {
		//Nothing
	}
}

func (tt *RetryTrigger) loop() ReturnValue {
	tt.CurrentRetry = 0

	if tt.Schedule == nil {
		//dont loop on triggers if no timer present
		log.WithFields(log.Fields{"Trigger": tt.ID()}).Info("Skipping looping, because no timer was set")
		return returnStop
	}
	for {
		log.WithFields(log.Fields{"Trigger": tt.ID(), "Retry": tt.CurrentRetry}).Info("Starting try")

		//TODO result resolution
		var r ReturnValue
		for _, t := range tt.Targets {
			if x := t.Trigger(nil); x > r {
				r = x
			}
		}
		result := r

		switch result {
		case returnRetry:
			if tt.CurrentRetry < tt.MaxFailedRetries {
				tt.retry()
				tt.NextTriggerWithDelay(tt.durationTillNextTrigger())
			} else {
				tt.fail()
				return result
			}
		case returnOk:
			tt.success()
			return result
		case returnStop:
			tt.fail()
			return result
		}
	}
}

func (tt *RetryTrigger) durationTillNextTrigger() time.Duration {
	dur := time.Duration(-1)

	if tt.Schedule != nil {
		dur = tt.Schedule.Next(time.Now()).Sub(time.Now())
	}
	return dur
}

func (tt *RetryTrigger) retry() {
	log.WithFields(log.Fields{"Trigger": tt.ID(), "Retries": tt.CurrentRetry}).Info("Start next retry")
	tt.CurrentRetry++
}

func (tt *RetryTrigger) fail() {
	log.WithFields(log.Fields{"Trigger": tt.ID(), "Retries": tt.CurrentRetry}).Error("Failed. Will try again at next regular trigger")
}

func (tt *RetryTrigger) failPreconds() {
	log.WithFields(log.Fields{"Trigger": tt.ID()}).Error("Failed Preconditions. Will try again at next regular trigger")
}

func (tt *RetryTrigger) success() {
	log.WithFields(log.Fields{"Trigger": tt.ID(), "Retries": tt.CurrentRetry}).Info("successful")
}
