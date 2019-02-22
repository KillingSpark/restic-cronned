package triggers

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/killingspark/restic-cronned/src/objectstore"
	"github.com/robfig/cron"
	"time"
)

type TimedTriggerDescription struct {
	SimpleTriggererDescription

	Timer           string `json:"Timer"`
	WaitGranularity int    `json:"WaitGranularity"`
}

func (d *TimedTriggerDescription) Instantiate(unique string) (objectstore.Triggerer, error) {
	tr := &TimedTrigger{}
	var err error

	simple, err := d.SimpleTriggererDescription.Instantiate(unique)
	if err != nil {
		return nil, err
	}
	tr.SimpleTrigger = *(simple.(*SimpleTrigger))

	tr.waitGran = time.Duration(d.WaitGranularity) * time.Millisecond

	tr.Schedule, err = cron.Parse(d.Timer)
	if err != nil {
		log.WithFields(log.Fields{"ID": d.ID, "Error": err.Error()}).Warning("Decoding error for the RegularTimer")
		return nil, err
	}
	return tr, nil
}

type TimedTrigger struct {
	SimpleTrigger

	Schedule cron.Schedule
	waitGran time.Duration

	//times set when the wait is started
	WaitStart time.Duration
	WaitEnd   time.Duration
}

func (tt *TimedTrigger) Run(ctx context.Context) error {
	tt.loop(ctx)
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
				time.Sleep(1 * time.Second)
			}
		}
	}
	if dur == 0 {
		//Nothing
	}
}

func (tt *TimedTrigger) loop(ctx context.Context) {
	if tt.Schedule == nil {
		//dont loop on triggers if no timer present
		log.WithFields(log.Fields{"Trigger": tt.ID()}).Info("Skipping looping, because no timer was set")
		return
	}
	//wait for first regular trigger before looping
	log.WithFields(log.Fields{"Trigger": tt.ID()}).Info("Waiting before the first run of the job")
	tt.NextTriggerWithDelay(tt.durationTillNextTrigger())

	for {
		select {
		case <-ctx.Done():
			return
		default: //proceed
		}

		log.WithFields(log.Fields{"Trigger": tt.ID()}).Info("Waiting finished an no kill received")

		var r objectstore.ReturnValue
		for _, t := range tt.Targets {
			if x := t.Trigger(nil); x > r {
				r = x
			}
		}
		//ignore results. just trigger next time
		tt.NextTriggerWithDelay(tt.durationTillNextTrigger())
	}
}

func (tt *TimedTrigger) durationTillNextTrigger() time.Duration {
	dur := time.Duration(-1)

	if tt.Schedule != nil {
		dur = tt.Schedule.Next(time.Now()).Sub(time.Now())
	}
	return dur
}
