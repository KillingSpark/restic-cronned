package jobs

import (
	"os"
	"testing"
)

func TestPreconds(t *testing.T) {
	pc := JobPreconditions{}
	pc.HostsMustConnect = []HostTCPPrecond{HostTCPPrecond{Host: "google.de", Port: 80}}

	if !pc.CheckAll() {
		t.Error("Couldnt connect to google?")
	}

	pc.HostsMustRoute = []HostRoutePrecond{HostRoutePrecond("localhost")}
	if !pc.CheckAll() {
		t.Error("Couldnt route localhost?")
	}

	err := os.MkdirAll("/tmp/backup/backup", 0777)
	if err == nil {
		pc.PathesMust = []PathPrecond{PathPrecond("/tmp/backup")}
		if !pc.CheckAll() {
			t.Error("Couldnt find /tmp/backup but it exists?")
		}
	} else {
		t.Error("Couldnt create /tmp/backup")
	}

	js, _ := FindJobs("../JobJsons")
	for _, j := range js {
		if j.JobName != "ExampleBackup" {
			continue
		}
		if len(j.Preconditions.PathesMust) <= 0 {
			t.Error("Preconditions not correctly unmarshalled")
		}
		if len(j.Preconditions.HostsMustRoute) <= 0 {
			t.Error("Preconditions not correctly unmarshalled")
		}
		if len(j.Preconditions.HostsMustConnect) <= 0 {
			t.Error("Preconditions not correctly unmarshalled")
		}
	}
}
