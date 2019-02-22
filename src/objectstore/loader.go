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
	Triggerers           map[string]TriggererDescription
	registeredTriggerers map[string]TriggererRegisterFunc

	Triggerables           map[string]TriggerableDescription
	registeredTriggerables map[string]TriggerableRegisterFunc
}

type TriggererRegisterFunc func(json.RawMessage) (TriggererDescription, error)
type TriggerableRegisterFunc func(json.RawMessage) (TriggerableDescription, error)

func NewObjectStore() *ObjectStore {
	s := &ObjectStore{}
	s.registeredTriggerables = make(map[string]TriggerableRegisterFunc)
	s.registeredTriggerers = make(map[string]TriggererRegisterFunc)
	return s
}

func (s *ObjectStore) RegisterTriggererType(name string, f TriggererRegisterFunc) error {
	if _, ok := s.registeredTriggerers[name]; ok {
		return errors.New("Type already registered as triggerer: " + name)
	}
	s.registeredTriggerers[name] = f
	return nil
}
func (s *ObjectStore) RegisterTriggerableType(name string, f TriggerableRegisterFunc) error {
	if _, ok := s.registeredTriggerables[name]; ok {
		return errors.New("Type already registered as triggerable: " + name)
	}
	s.registeredTriggerables[name] = f
	return nil
}

func (s *ObjectStore) LoadObject(objdesc *ObjectDescription) error {
	triggabledesc, taok := s.registeredTriggerables[objdesc.Kind.Name]
	triggerdesc, trok := s.registeredTriggerers[objdesc.Kind.Name]

	if taok {
		desc, err := triggabledesc(objdesc.Spec)
		if err != nil {
			return err
		}
		s.Triggerables[desc.ID()] = desc
	}
	if trok {
		desc, err := triggerdesc(objdesc.Spec)
		if err != nil {
			return err
		}
		s.Triggerers[desc.ID()] = desc
	}

	if !taok && !trok {
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
				return errors.New(f.Name() + ":" + err.Error())
			}
			f, err = os.Stat(trgt)
			if err != nil {
				return errors.New(f.Name() + ":" + err.Error())
			}
		}

		if f.IsDir() {
			//recurse into directories to allow for better separation of triggers/jobs/flows without
			//imposing a fixed directory structure
			err := s.LoadAllObjects(path.Join(dirPath, f.Name()))
			if err != nil {
				return errors.New(f.Name() + ":" + err.Error())
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
			return errors.New(f.Name() + ":" + err.Error())
		}

		if len(objdesc.Kind.Name) == 0 && len(objdesc.Spec) == 0 {
			//skip files that are obviously no objectdescriptions
			log.WithFields(log.Fields{"File": path.Join(dirPath, f.Name())}).Warn("Skipping file. Not a objectdescription")
			continue
		}

		//tries to find a probider and adds it to the correct map if found
		err = s.LoadObject(objdesc)
		if err != nil {
			return errors.New(f.Name() + ":" + err.Error())
		}
	}

	return nil
}
