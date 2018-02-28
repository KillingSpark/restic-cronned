package main

import (
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/killingspark/restic-cronned/jobs"
	"github.com/killingspark/restic-cronned/keyutil"
	"github.com/killingspark/restic-cronned/output"
)

func startDaemon() {
	log.SetFormatter(&log.TextFormatter{})
	logfile, err := os.OpenFile("/tmp/restic-cronned.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		log.SetOutput(logfile)
	} else {
		println(err.Error())
		os.Exit(1)
		log.SetOutput(os.Stdout)
	}
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
