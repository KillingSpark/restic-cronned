package jobs

import (
	log "github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
	"sync"
	"time"
)

type Triggerable interface {
	Trigger() JobReturn
	Name() string
	CheckPreconditions() bool
}

type TimedTrigger struct {
	JobToTrigger string
	ToTrigger    Triggerable
	Lock         sync.Mutex //protects access on ToTrigger
	Kill         chan int

	regTimerSchedule   cron.Schedule
	retryTimerSchedule cron.Schedule
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
	if dur == 0 {
		//Nothing
	}
}

func (tt *TimedTrigger) loop() {
	if tt.regTimerSchedule == nil {
		//dont loop on triggers if no timer present
		log.WithFields(log.Fields{"Job": tt.ToTrigger.Name()}).Info("Skipping looping, because no timer was set")
		return
	}
	//wait for first regular trigger before looping
	log.WithFields(log.Fields{"Job": tt.ToTrigger.Name()}).Info("Waiting before the first run of the job")
	tt.NextTriggerWithDelay(tt.durationTillNextRegularTrigger())
	for {
		select {
		case <-tt.Kill:
			tt.Kill <- 0
			return
		default: //proceed
		}

		log.WithFields(log.Fields{"Job": tt.ToTrigger.Name()}).Info("Waiting finished an no kill received")

		if !tt.waitPreconds() {
			tt.failPreconds() // --> wait till next regular
			tt.NextTriggerWithDelay(tt.durationTillNextRegularTrigger())
			continue
		}

		tt.Lock.Lock()
		result := tt.ToTrigger.Trigger()
		tt.Lock.Unlock()

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

func (tt *TimedTrigger) waitPreconds() bool {
	if tt.CheckPrecondsMaxTimes > 0 {
		preconds := false
		for i := 0; !preconds && i < tt.CheckPrecondsMaxTimes; i++ {
			preconds = tt.ToTrigger.CheckPreconditions()
			if !preconds {
				time.Sleep(time.Duration(tt.CheckPrecondsEvery) * time.Second)
			}
		}
		return preconds
	}
	return true
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
}

func (tt *TimedTrigger) success() {
	log.WithFields(log.Fields{"Job": tt.ToTrigger.Name(), "Retries": tt.CurrentRetry}).Info("successful")
}
