package objectstore

import (
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
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
	case "Retry":
		desc := &RetryTriggererDescription{}
		err := json.Unmarshal(objdesc.Spec, desc)
		if err != nil {
			return err
		}
		s.Triggerers[desc.ID()] = desc

		desc2 := &RetryTriggerableDescription{}
		err = json.Unmarshal(objdesc.Spec, desc2)
		if err != nil {
			return err
		}
		s.Triggerables[desc2.ID()] = desc2

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
		s.Triggerables[desc2.ID()] = desc2
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

func LoadFlowForest(filePath string) (*FlowForest, error) {
	ff := &FlowForest{}

	marshflow, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	err = ff.Load(marshflow)
	if err != nil {
		return nil, err
	}
	return ff, nil
}

func LoadAllFlowForrests(dirPath string) (*FlowForest, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, errors.New(dirPath + " is no directory")
	}

	ff := &FlowForest{}

	for _, f := range files {

		//follow symlinks
		if f.Mode()&os.ModeSymlink != 0 {
			trgt, err := os.Readlink(path.Join(dirPath, f.Name()))
			if err != nil {
				return nil, err
			}
			f, err = os.Stat(trgt)
			if err != nil {
				return nil, err
			}
		}

		if f.IsDir() {
			//recurse into directories to allow for better separation of triggers/jobs/flows without
			//imposing a fixed directory structure
			newff, err := LoadAllFlowForrests(path.Join(dirPath, f.Name()))
			if err != nil {
				return nil, err
			}
			ff, err = ff.Merge(newff)
			if err != nil {
				return nil, err
			}

			continue
		}

		if !(strings.HasSuffix(f.Name(), ".flow")) {
			//ignore none-flow files
			continue
		}

		marshflow, err := ioutil.ReadFile(path.Join(dirPath, f.Name()))
		if err != nil {
			return nil, err
		}

		newff := &FlowForest{}
		err = json.Unmarshal(marshflow, newff)
		if err != nil {
			return nil, err
		}

		ff, err = ff.Merge(newff)
		if err != nil {
			return nil, err
		}
	}
	return ff, nil
}

func (s *ObjectStore) LoadAllObjects(dirPath string) error {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return errors.New(dirPath + " is no directory")
	}

	s.Triggerables = make(map[string]TriggerableDescription)
	s.Triggerers = make(map[string]TriggererDescription)

	for _, f := range files {
		//follow symlinks
		if f.Mode()&os.ModeSymlink != 0 {
			trgt, err := os.Readlink(path.Join(dirPath, f.Name()))
			if err != nil {
				return err
			}
			f, err = os.Stat(trgt)
			if err != nil {
				return err
			}
		}

		if f.IsDir() {
			//recurse into directories to allow for better separation of triggers/jobs/flows without
			//imposing a fixed directory structure
			err := s.LoadAllObjects(path.Join(dirPath, f.Name()))
			if err != nil {
				return err
			}
			continue
		}
		if !(strings.HasSuffix(f.Name(), ".json")) {
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

		if len(objdesc.Kind.Name) == 0 && len(objdesc.Spec) == 0 {
			//skip files that are obviously no objectdescriptions
			log.WithFields(log.Fields{"File": path.Join(dirPath, f.Name())}).Warn("Skipping file. Not a objectdescription")
			continue
		}

		//tries to find a probider and adds it to the correct map if found
		err = s.LoadObject(objdesc)
		if err != nil {
			return err
		}
	}

	return nil
}
