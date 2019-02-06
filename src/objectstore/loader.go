package objectstore

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type ObjectDescription struct {
	Kind struct {
		Name string
	}
	Spec json.RawMessage
}

type ObjectStore struct {
	Triggerers   map[string]TriggererDescription
	Triggerables map[string]TriggerableDescription
}

func (s *ObjectStore) LoadObject(objdesc *ObjectDescription) error {
	switch objdesc.Kind.Name {
	case "Timer":
		desc := &TimedTriggerDescription{}
		err := json.Unmarshal(objdesc.Spec, desc)
		if err != nil {
			return err
		}
		s.Triggerers[desc.ID()] = desc
	case "Oneshot":
		desc := &OneshotTriggererDescription{}
		err := json.Unmarshal(objdesc.Spec, desc)
		if err != nil {
			return err
		}
		s.Triggerers[desc.ID()] = desc

		desc2 := &OneshotTriggerableDescription{}
		err = json.Unmarshal(objdesc.Spec, desc2)
		if err != nil {
			return err
		}
		s.Triggerables[desc.ID()] = desc2
	case "Parshot":
		desc := &ParallelOneshotTriggererDescription{}
		err := json.Unmarshal(objdesc.Spec, desc)
		if err != nil {
			return err
		}
		s.Triggerers[desc.ID()] = desc

		desc2 := &ParallelOneshotTriggerableDescription{}
		err = json.Unmarshal(objdesc.Spec, desc2)
		if err != nil {
			return err
		}
		s.Triggerables[desc.ID()] = desc2
	case "Job":
		desc := &JobDescription{}
		err := json.Unmarshal(objdesc.Spec, desc)
		if err != nil {
			return err
		}
		s.Triggerables[desc.ID()] = desc
	default:
		panic("Cant handle kind: " + objdesc.Kind.Name)
	}
	return nil
}

func (s *ObjectStore) LoadAllObjects(dirPath string) error {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return errors.New(dirPath + " is no directory")
	}

	s.Triggerables = make(map[string]TriggerableDescription)
	s.Triggerers = make(map[string]TriggererDescription)

	for _, f := range files {
		if f.IsDir() || !(strings.HasSuffix(f.Name(), ".json")) {
			//ignore none-json files
			continue
		}
		file, err := os.Open(path.Join(dirPath, f.Name()))
		if err != nil {
			return errors.New("cannot open file: " + path.Join(dirPath, f.Name()))
		}

		objdesc := &ObjectDescription{}
		jsonParser := json.NewDecoder(file)
		err = jsonParser.Decode(objdesc)
		if err != nil {
			return err
		}

		//tries to find a probider and adds it to the correct map if found
		err = s.LoadObject(objdesc)
		if err != nil {
			return err
		}
	}

	return nil
}
