package main

import (
	"encoding/json"
	"os"
	"path"

	"github.com/KillingSpark/restic-cronned/jobs"
	"github.com/KillingSpark/restic-cronned/output"
	log "github.com/Sirupsen/logrus"
	"github.com/rshmelev/lumberjack"
)

type serverConfig struct {
	Port string `json:"Port"`
}

type loggingConfig struct {
	MaxSize int    `json:"MaxSize"`
	MaxAge  int    `json:"MaxAge"`
	LogDir  string `json:"LogDir"`
}

type config struct {
	JobPath string        `json:"JobPath"`
	SrvConf serverConfig  `json:"SrvConf"`
	LogConf loggingConfig `json:"LogConf"`
}

func setupLogging(conf loggingConfig) {
	log.SetFormatter(&log.TextFormatter{})
	logpath := os.ExpandEnv(conf.LogDir)
	os.MkdirAll(logpath, 0700) //readwrite for user only
	log.SetOutput(&lumberjack.Logger{
		Filename: path.Join(logpath, "restic-cronned.log"),
		MaxSize:  conf.MaxSize, // megabytes
		MaxAge:   conf.MaxAge,  //days
	})
}

func startDaemon(conf config) {

	var jobDirPath = os.ExpandEnv(conf.JobPath)
	var port = conf.SrvConf.Port

	if len(os.Args) > 1 {
		jobDirPath = os.Args[1]
	}

	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	println("Path: " + jobDirPath)
	println("Port: " + port)

	queue, err := jobs.NewJobQueue(jobDirPath)
	if err != nil {
		println(err.Error())
		return
	}
	queue.StartQueue()

	if len(port) > 2 {
		go output.StartServer(queue, port)
	} else {
		println("no valid port specified -> no status server started")
	}

	queue.WaitForAllJobs()
	log.Info("All Jobs stopped")
}

func loadConfig() config {
	conf := config{}
	//default config
	conf.JobPath = os.ExpandEnv("$HOME/.config/restic-cronned/jobs/")
	conf.SrvConf.Port = ":8080"
	conf.LogConf.MaxAge = 30
	conf.LogConf.MaxSize = 10
	conf.LogConf.LogDir = os.ExpandEnv("$HOME/.cache/restic-cronned")
	confPath := os.ExpandEnv("$HOME") + "/.config/restic-cronned/config"

	confFile, err := os.Open(confPath)
	if err != nil {
		println("config file not found at: " + confPath + " -> using default config")
	} else {
		err = json.NewDecoder(confFile).Decode(&conf)
		if err != nil {
			println(err.Error())
		}
	}
	return conf
}

func main() {
	conf := loadConfig()
	setupLogging(conf.LogConf)
	startDaemon(conf)
}
