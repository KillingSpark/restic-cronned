package objectstore

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
	"sync"
	"time"
)

const (
	returnStop  ReturnValue = 0
	returnOk    ReturnValue = 1
	returnRetry ReturnValue = 2
)

type TimedTriggerDescription struct {
	Name            string `Json:"Name"`
	RegularTimer    string `json:"regularTimer"`
	RetryTimer      string `json:"retryTimer"`
	WaitGranularity int    `json:"waitGranularity"`

	MaxFailedRetries int `json:"maxFailedRetries"`
}

func (d *TimedTriggerDescription) ID() string {
	return d.Name
}

func (d *TimedTriggerDescription) Instantiate(unique string) (Triggerer, error) {
	tr := &TimedTrigger{}
	var err error

	tr.Name = unique + "__" + d.Name
	tr.MaxFailedRetries = d.MaxFailedRetries
	tr.waitGran = time.Duration(d.WaitGranularity) * time.Millisecond

	if len(d.RegularTimer) > 0 {
		tr.regTimerSchedule, err = cron.Parse(d.RegularTimer)
		if err != nil {
			log.WithFields(log.Fields{"ID": d.ID, "Error": err.Error()}).Warning("Decoding error for the RegularTimer")
			return nil, err
		}
	}
	if len(d.RetryTimer) > 0 {
		tr.retryTimerSchedule, err = cron.Parse(d.RetryTimer)
		if err != nil {
			log.WithFields(log.Fields{"ID": d.ID, "Error": err.Error()}).Warning("Decoding error for the RetryTimer")
			return nil, err
		}
	}
	return tr, nil
}

type TimedTrigger struct {
	Name    string
	Targets []Triggerable
	Lock    sync.Mutex //protects access on ToTrigger
	Kill    chan int

	regTimerSchedule   cron.Schedule
	retryTimerSchedule cron.Schedule
	waitGran           time.Duration

	CurrentRetry     int
	MaxFailedRetries int

	CheckPrecondsEvery    int
	CheckPrecondsMaxTimes int

	//times set when the wait is started
	WaitStart time.Duration
	WaitEnd   time.Duration
}

func (tt *TimedTrigger) ID() string {
	return tt.Name
}

func (tt *TimedTrigger) AddTarget(ta Triggerable) error {
	tt.Targets = append(tt.Targets, ta)
	return nil
}

func (tt *TimedTrigger) Run(ctx context.Context) error {
	tt.loop()
	return nil
}

//NextTriggerWithDelay makes the job run after "dur" nanoseconds
func (tt *TimedTrigger) NextTriggerWithDelay(dur time.Duration) {
	if dur < 0 {
		//ignore for example jobs that shouldnt be run
		log.WithFields(log.Fields{"Trigger": tt.ID(), "Duration": dur}).Info("Ignore trigger with negative duration")
	}
	if dur > 0 {
		//for frontends
		tt.WaitStart = time.Duration(time.Now().UnixNano())
		tt.WaitEnd = tt.WaitStart + dur

		sleepUntil := time.Now().Add(dur).Round(0)

		log.WithFields(log.Fields{"Trigger": tt.ID(), "Time": sleepUntil.String()}).Info("Timed trigger scheduled")

		for time.Now().Round(0).Before(sleepUntil) {
			if tt.waitGran > 0 {
				time.Sleep(tt.waitGran)
			} else {
				time.Sleep(10 * time.Second)
			}
		}
	}
	if dur == 0 {
		//Nothing
	}
}

func (tt *TimedTrigger) loop() {
	if tt.regTimerSchedule == nil {
		//dont loop on triggers if no timer present
		log.WithFields(log.Fields{"Trigger": tt.ID()}).Info("Skipping looping, because no timer was set")
		return
	}
	//wait for first regular trigger before looping
	log.WithFields(log.Fields{"Trigger": tt.ID()}).Info("Waiting before the first run of the job")
	tt.NextTriggerWithDelay(tt.durationTillNextRegularTrigger())
	for {
		select {
		case <-tt.Kill:
			tt.Kill <- 0
			return
		default: //proceed
		}

		log.WithFields(log.Fields{"Trigger": tt.ID()}).Info("Waiting finished an no kill received")

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
				tt.NextTriggerWithDelay(tt.durationTillNextRetryTrigger())
			} else {
				tt.fail()
				tt.NextTriggerWithDelay(tt.durationTillNextRegularTrigger())
			}
		case returnOk:
			tt.success()
			tt.NextTriggerWithDelay(tt.durationTillNextRegularTrigger())
		case returnStop:
			tt.fail()
			return
		}
	}
}

func (tt *TimedTrigger) durationTillNextRegularTrigger() time.Duration {
	dur := time.Duration(-1)

	if tt.regTimerSchedule != nil {
		dur = tt.regTimerSchedule.Next(time.Now()).Sub(time.Now())
	}
	return dur
}

func (tt *TimedTrigger) durationTillNextRetryTrigger() time.Duration {
	dur := time.Duration(-1)

	if tt.retryTimerSchedule != nil {
		dur = tt.retryTimerSchedule.Next(time.Now()).Sub(time.Now())
	}
	return dur
}

func (tt *TimedTrigger) retry() {
	log.WithFields(log.Fields{"Trigger": tt.ID(), "Retries": tt.CurrentRetry}).Info("Start next retry")
	tt.CurrentRetry++
}

func (tt *TimedTrigger) fail() {
	log.WithFields(log.Fields{"Trigger": tt.ID(), "Retries": tt.CurrentRetry}).Error("Failed. Will try again at next regular trigger")
}

func (tt *TimedTrigger) failPreconds() {
	log.WithFields(log.Fields{"Trigger": tt.ID()}).Error("Failed Preconditions. Will try again at next regular trigger")
}

func (tt *TimedTrigger) success() {
	log.WithFields(log.Fields{"Trigger": tt.ID(), "Retries": tt.CurrentRetry}).Info("successful")
}
