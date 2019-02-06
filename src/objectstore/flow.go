package objectstore

import (
	"encoding/json"
	"errors"
)

type FlowNode struct {
	Name    string //correpsonds to an ID in the objectstore
	Targets []*FlowNode
}

type Flow struct {
	Name string
	Root *FlowNode
}

type FlowForest struct {
	Flows map[string]*Flow
}

func (ff *FlowForest) Load(raw json.RawMessage) error {
	return json.Unmarshal(raw, ff)
}

func (ff *FlowForest) recBuild(tr Triggerer, store *ObjectStore, node *FlowNode) error {
	for _, target := range node.Targets {
		triggdesc, ok := store.Triggerables[target.Name]
		if !ok {
			return errors.New("No triggerable with name: " + target.Name)
		}
		triggable, err := triggdesc.Instantiate()
		if err != nil {
			return err
		}
		tr.AddTarget(triggable)

		//need to recurse
		if len(target.Targets) > 0 {
			triggdesc, ok := store.Triggerers[target.Name]
			if !ok {
				return errors.New("No triggerer with name: " + target.Name)
			}
			triggerer, err := triggdesc.Instantiate()
			if err != nil {
				return err
			}
			if err = ff.recBuild(triggerer, store, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ff *FlowForest) Build(name string, store *ObjectStore) (Triggerer, error) {
	flow, ok := ff.Flows[name]
	if !ok {
		return nil, errors.New("No flow with that name found")
	}
	roottrdesc, ok := store.Triggerers[flow.Root.Name]
	if !ok {
		return nil, errors.New("No triggerable with name: " + flow.Root.Name)
	}

	roottr, err := roottrdesc.Instantiate()
	if err != nil {
		return nil, err
	}

	ff.recBuild(roottr, store, flow.Root)

	return roottr, nil
}
