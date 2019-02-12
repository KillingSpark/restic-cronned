package triggers

import (
	"bufio"
	"context"
	"github.com/killingspark/restic-cronned/src/objectstore"
	"net"
)

type UnixTriggererDescription struct {
	Name       string `json:"Name"`
	SocketPath string `json:"SocketPath"`
}

func (d *UnixTriggererDescription) ID() string {
	return d.Name
}

func (d *UnixTriggererDescription) Instantiate(unique string) (objectstore.Triggerer, error) {
	tr := &UnixTriggerer{}

	tr.Name = unique + "__" + d.Name
	tr.SocketPath = d.SocketPath

	return tr, nil
}

type UnixTriggerer struct {
	Name           string
	Targets        []objectstore.Triggerable
	SocketPath     string
	TriggerCounter int
}

func (tt *UnixTriggerer) ID() string {
	return tt.Name
}

func (tt *UnixTriggerer) AddTarget(ta objectstore.Triggerable) error {
	tt.Targets = append(tt.Targets, ta)
	return nil
}

func (tt *UnixTriggerer) Run(ctx *context.Context) error {
	conn, err := net.Dial("unix", tt.SocketPath)
	if err != nil {
		return err
	}
	br := bufio.NewReader(conn)
	for {
		//wait for signal
		_, err := br.ReadString('\n')
		if err != nil {
			return err
		}

		//trigger all targets
		for _, t := range tt.Targets {
			t.Trigger(ctx)
		}

	}
}
