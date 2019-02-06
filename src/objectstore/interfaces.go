package objectstore

import (
	"context"
	"github.com/killingspark/restic-cronned/src/jobs"
)

type ReturnValue = jobs.JobReturn

type Triggerable interface {
	Trigger(*context.Context) ReturnValue
}

type Triggerer interface {
	AddTarget(Triggerable) error
	Run(context.Context) error
}

type TriggererDescription interface {
	Instantiate() (Triggerer, error)
}

type TriggerableDescription interface {
	Instantiate() (Triggerable, error)
	ID() string
}
