package jobs

import (
	log "github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
	"time"
)

type Triggerable interface {
	Trigger() JobReturn
	Name() string
	CheckPreconditions() bool
}

type TimedTrigger struct {
	ToTrigger Triggerable
	Kill      chan int

	RegularTimer       string `json:"regularTimer"`
	regTimerSchedule   cron.Schedule
	RetryTimer         string `json:"retryTimer"`
	retryTimerSchedule cron.Schedule
	WaitGranularity    string `json:"waitGranularity"`
	waitGran           time.Duration

	CurrentRetry     int `json:"CurrentRetry"`
	MaxFailedRetries int `json:"maxFailedRetries"`

	CheckPrecondsEvery    int `json:"CheckPrecondsEvery"`
	CheckPrecondsMaxTimes int `json:"CheckPrecondsMaxTimes"`

	//times set when the wait is started
	WaitStart time.Duration `json:"WaitStart"`
	WaitEnd   time.Duration `json:"WaitEnd"`
}

//NextTriggerWithDelay makes the job run after "dur" nanoseconds
func (tt *TimedTrigger) NextTriggerWithDelay(dur time.Duration) {
	if dur < 0 {
		//ignore for example jobs that shouldnt be run
		log.WithFields(log.Fields{"Job": tt.ToTrigger.Name(), "Duration": dur}).Info("Ignore trigger with negative duration")
	}
	if dur > 0 {
		//for frontends
		tt.WaitStart = time.Duration(time.Now().UnixNano())
		tt.WaitEnd = tt.WaitStart + dur

		sleepUntil := time.Now().Add(dur).Round(0)

		log.WithFields(log.Fields{"Job": tt.ToTrigger.Name(), "Time": sleepUntil.String()}).Info("Timed trigger scheduled")

		for time.Now().Round(0).Before(sleepUntil) {
			if tt.waitGran > 0 {
				time.Sleep(tt.waitGran)
			} else {
				time.Sleep(10 * time.Second)
			}
		}
	}
	if dur >= 0 {
		tt.ToTrigger.Trigger()
	}
}

func (tt *TimedTrigger) loop() {
	for {
		log.WithFields(log.Fields{"Job": tt.ToTrigger.Name()}).Info("Await trigger/stop")

		select {
		case <-tt.Kill:
			tt.Kill <- 0
			return
		default: //proceed
		}

		log.WithFields(log.Fields{"Job": tt.ToTrigger.Name()}).Info("Waiting finished an no kill received")

		if tt.CheckPrecondsMaxTimes > 0 {
			preconds := false
			for i := 0; !preconds && i < tt.CheckPrecondsMaxTimes; i++ {
				preconds = tt.ToTrigger.CheckPreconditions()
				if !preconds {
					time.Sleep(time.Duration(tt.CheckPrecondsEvery) * time.Second)
				}
			}
			if !preconds {
				tt.failPreconds()
				continue
			}
		}

		result := tt.ToTrigger.Trigger()
		switch result {
		case returnRetry:
			if tt.CurrentRetry < tt.MaxFailedRetries {
				tt.retry()
				tt.NextTriggerWithDelay(tt.durationTillNextRetryTrigger())
			} else {
				tt.fail()
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
	log.WithFields(log.Fields{"Job": tt.ToTrigger.Name(), "Retries": tt.CurrentRetry}).Info("Start next retry")
	tt.CurrentRetry++
}

func (tt *TimedTrigger) fail() {
	log.WithFields(log.Fields{"Job": tt.ToTrigger.Name(), "Retries": tt.CurrentRetry}).Error("Failed. Will try again at next regular trigger")
}

func (tt *TimedTrigger) failPreconds() {
	log.WithFields(log.Fields{"Job": tt.ToTrigger.Name()}).Error("Failed Preconditions. Will try again at next regular trigger")
	tt.NextTriggerWithDelay(tt.durationTillNextRegularTrigger())
}

func (tt *TimedTrigger) success() {
	log.WithFields(log.Fields{"Job": tt.ToTrigger.Name(), "Retries": tt.CurrentRetry}).Info("successful")
}
