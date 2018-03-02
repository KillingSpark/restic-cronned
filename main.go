package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/killingspark/restic-cronned/jobs"
	"github.com/killingspark/restic-cronned/output"
)

func setupLogging() error {
	log.SetFormatter(&log.TextFormatter{})
	logfile, err := os.OpenFile("/tmp/restic-cronned.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		//log.SetOutput(logfile)
		_ = logfile
	} else {
		log.SetOutput(os.Stdout)
		return err
	}
	return nil
}

func startDaemon() {
	if len(os.Args) < 2 {
		println("No directory specified")
		return
	}

	jobDirPath := os.Args[1]
	queue, err := jobs.NewJobQueue(jobDirPath)
	if err != nil {
		println(err.Error())
		return
	}
	queue.StartQueue()

	if len(os.Args) > 2 {
		port := os.Args[2]
		go output.StartServer(queue, port)
	}

	queue.WaitForAllJobs()
}

func main() {
	err := setupLogging()
	if err != nil {
		println(err.Error())
		return
	}
	startDaemon()
}
