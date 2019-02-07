package objectstore

import (
	"context"
	"github.com/killingspark/restic-cronned/src/jobs"
)

type ReturnValue = jobs.JobReturn

//ugly private interfaces. Needed because go doesnt allow overlapping of interfaces
type __Triggerable interface {
	Trigger(*context.Context) ReturnValue
}
type __Triggerer interface {
	AddTarget(Triggerable) error
	Run(context.Context) error
}
type __IDable interface {
	ID() string
}

type Triggerable interface {
	__Triggerable
	__IDable
}

type Triggerer interface {
	__Triggerer
	__IDable
}

type TriggerableTriggerer interface {
	__Triggerer
	__Triggerable
	__IDable
}

type TriggererDescription interface {
	Instantiate(unique string) (Triggerer, error)
	__IDable
}

type TriggerableDescription interface {
	Instantiate(unique string) (Triggerable, error)
	__IDable
}
