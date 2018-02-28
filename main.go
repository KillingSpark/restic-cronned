package main

import (
	"os"
	"strings"

	"github.com/killingspark/restic-cronned/jobs"
	"github.com/killingspark/restic-cronned/keyutil"
	"github.com/killingspark/restic-cronned/output"
)

func startDaemon() {
	queue := jobs.NewJobQueue()
	jobDirPath := os.Args[1]
	if len(os.Args) > 2 {
		port := os.Args[2]
		go output.StartServer(queue, port)
	}
	jobs := jobs.FindJobs(jobDirPath)
	queue.AddJobs(jobs)
	queue.Wg.Wait()
}

const (
	daemonSuffix  = "cronned"
	keyringSuffix = "keyutil"
)

func main() {
	//exename := os.Args[0]
	//exename := "keyutil"
	exename := "cronned"

	if strings.HasSuffix(exename, daemonSuffix) {
		startDaemon()
	}
	if strings.HasSuffix(exename, keyringSuffix) {
		keyutil.KeyRingUtil()
	}
}
