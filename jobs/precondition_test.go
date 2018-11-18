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

	pc.HostsMustRoute = []HostRoutePrecond{HostRoutePrecond{Host: "localhost"}}
	if !pc.CheckAll() {
		t.Error("Couldnt route localhost?")
	}

	err := os.MkdirAll("/tmp/backup/backup", 0777)
	if err == nil {
		pc.PathesMust = []PathPrecond{PathPrecond{Path: "/tmp/backup"}}
		if !pc.CheckAll() {
			t.Error("Couldnt find /tmp/backup but it exists?")
		}
	} else {
		t.Error("Couldnt create /tmp/backup")
	}

}
