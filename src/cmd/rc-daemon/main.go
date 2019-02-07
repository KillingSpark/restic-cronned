package main

import (
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/killingspark/restic-cronned/src/objectstore"
	"github.com/rshmelev/lumberjack"
	"github.com/spf13/viper"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	dirpath    = kingpin.Flag("dirpath", "Which directory contains the job/trigger/flow descriptions").Short('d').String()
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
	objs := objectstore.ObjectStore{}
	err := objs.LoadAllObjects(*dirpath)
	if err != nil {
		println(err.Error())
		return
	}

	ff, err := objectstore.LoadAllFlowForrests(*dirpath)
	if err != nil {
		println(err.Error())
		return
	}

	builtff, err := ff.BuildAll(&objs)
	if err != nil {
		println(err.Error())
		return
	}

	wg := sync.WaitGroup{}

	for n := range builtff.Roots {
		name := n
		root := builtff.Roots[name]
		wg.Add(1)
		go func() {
			log.WithFields(log.Fields{"Flow": name}).Info("Starting flow")
			root.Run(nil)
			wg.Done()
		}()
	}

	log.Info("All flows started")

	rootchan := make(chan int, 1)
	go func() {
		wg.Wait()
		rootchan <- 0
	}()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)

	select {
	case _ = <-sigchan:
		log.Info("Daemon exited after a os signal was received")
	case _ = <-rootchan:
		log.Info("Daemon exited after all flow roots exited")
	}
}

func loadConfig() error {
	kingpin.Parse()

	if *configpath != "" {
		viper.AddConfigPath(*configpath) // call multiple times to add many search paths
		println("ConfigPath: " + *configpath)
	}

	viper.AddConfigPath("/etc/restic-cronned/")         // path to look for the config file in
	viper.AddConfigPath("$HOME/.config/restic-cronned") // call multiple times to add many search paths

	viper.SetConfigType("json")
	viper.SetConfigName("config") // name of config file

	viper.SetDefault("Dir", os.ExpandEnv("$HOME/.config/restic-cronned/"))
	viper.SetDefault("LogDir", os.ExpandEnv("$HOME/.cache/restic-cronned"))
	viper.SetDefault("LogMaxAge", 30)
	viper.SetDefault("LogMaxSize", 10)

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	if *dirpath == "" {
		*dirpath = os.ExpandEnv(viper.GetString("Dir"))
	}

	println("ObjectPath: " + *dirpath)
	return nil
}

func main() {
	err := loadConfig()
	if err != nil {
		println(err.Error())
		return
	}
	setupLogging()
	startDaemon()
}
