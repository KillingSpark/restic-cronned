package objectstore

import (
	"context"
	log "github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
	"sync"
	"time"
)

type TimedTriggerDescription struct {
	Name            string `Json:"Name"`
	JobToTrigger    string `json:"JobToTrigger"`
	RegularTimer    string `json:"regularTimer"`
	RetryTimer      string `json:"retryTimer"`
	WaitGranularity string `json:"waitGranularity"`

	MaxFailedRetries int `json:"maxFailedRetries"`

	CheckPrecondsEvery    int `json:"CheckPrecondsEvery"`
	CheckPrecondsMaxTimes int `json:"CheckPrecondsMaxTimes"`
}

func (d *TimedTriggerDescription) ID() string {
	return d.Name
}

func (d *TimedTriggerDescription) Instantiate() (Triggerer, error) {
	tr := &TimedTrigger{}
	var err error

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

func (tt *TimedTrigger) AddTarget(ta Triggerable) error {
	tt.Targets = append(tt.Targets, ta)
	return nil
}

func (tt *TimedTrigger) Run(ctx context.Context) error {
	//tt.loop()
	return nil
}
