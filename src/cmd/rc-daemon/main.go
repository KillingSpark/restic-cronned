package main

import (
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/killingspark/restic-cronned/src/jobs"
	"github.com/killingspark/restic-cronned/src/output"
	"github.com/rshmelev/lumberjack"
	"github.com/spf13/viper"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	port       = kingpin.Flag("port", "Which port the server should listen on (if any)").Short('p').String()
	jobpath    = kingpin.Flag("jobpath", "Which directory contains the job descriptions").Short('j').String()
	configpath = kingpin.Flag("configpath", "Which directory contains the config file").Short('c').String()
)

func setupLogging() {
	log.SetFormatter(&log.TextFormatter{})
	logpath := os.ExpandEnv(viper.GetString("LogDir"))
	os.MkdirAll(logpath, 0700) //readwrite for user only
	log.SetOutput(&lumberjack.Logger{
		Filename: path.Join(logpath, "restic-cronned.log"),
		MaxSize:  viper.GetInt("LogMaxSize"), // megabytes
		MaxAge:   viper.GetInt("LogMaxAge"),  //days
	})
}

func startDaemon() {
	queue, err := jobs.NewJobQueue(*jobpath)
	if err != nil {
		println(err.Error())
		return
	}
	queue.StartQueue()

	if len(*port) > 2 {
		go output.StartServer(queue, *port)
	} else {
		println("no valid port specified -> no status server started")
	}

	queue.WaitForAllJobs()
	log.Info("All Jobs stopped")
}

func loadConfig() {
	kingpin.Parse()

	if *configpath != "" {
		viper.AddConfigPath(*configpath) // call multiple times to add many search paths
		println("ConfigPath: " + *configpath)
	}

	viper.SetConfigName("config")                       // name of config file (without extension)
	viper.AddConfigPath("/etc/restic-cronned/")         // path to look for the config file in
	viper.AddConfigPath("$HOME/.config/restic-cronned") // call multiple times to add many search paths

	viper.SetDefault("JobPath", os.ExpandEnv("$HOME/.config/restic-cronned/jobs/"))
	viper.SetDefault("ServerPort", ":8080")
	viper.SetDefault("LogMaxAge", 30)
	viper.SetDefault("LogMaxSize", 10)
	viper.SetDefault("LogDir", os.ExpandEnv("$HOME/.cache/restic-cronned"))

	viper.ReadInConfig()

	if *jobpath == "" {
		*jobpath = os.ExpandEnv(viper.GetString("JobPath"))
	}
	if *port == "" {
		*port = viper.GetString("ServerPort")
	}

	println("JobPath: " + *jobpath)
	println("Port: " + *port)
}

func main() {
	loadConfig()
	setupLogging()
	startDaemon()
}
