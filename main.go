package main

import (
	"os"
	"path"

	"github.com/KillingSpark/restic-cronned/jobs"
	"github.com/KillingSpark/restic-cronned/output"
	log "github.com/Sirupsen/logrus"
)

func setupLogging() error {
	log.SetFormatter(&log.TextFormatter{})
	logpath := os.ExpandEnv("$HOME/.cache")
	logfile, err := os.OpenFile(path.Join(logpath, "restic-cronned.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		log.SetOutput(logfile)
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
	log.Info("All Jobs stopped")
}

func main() {
	err := setupLogging()
	if err != nil {
		println(err.Error())
		return
	}
	startDaemon()
}
