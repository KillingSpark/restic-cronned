package objectstore

import (
	"context"
	"github.com/killingspark/restic-cronned/src/jobs"
)

type ReturnValue = jobs.JobReturn

type Triggerable interface {
	Trigger(*context.Context) ReturnValue
	ID() string
}

type Triggerer interface {
	AddTarget(Triggerable) error
	Run(context.Context) error
	ID() string
}

type TriggererDescription interface {
	Instantiate(unique string) (Triggerer, error)
}

type TriggerableDescription interface {
	Instantiate(unique string) (Triggerable, error)
	ID() string
}
