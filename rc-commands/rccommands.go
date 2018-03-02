package main

import (
	"io"
	"net/http"
	"os"
)

const (
	cmdStopAll string = "stopall"
	cmdStop    string = "stop"
	cmdRestart string = "restart"
	cmdReload  string = "reload"
)

func printUsage() {
	println("rccommands ip:port command name")
}

func main() {
	if len(os.Args) < 3 {
		printUsage()
		return
	}
	var resp *http.Response
	var err error
	switch os.Args[2] {
	case cmdStopAll:
		req, err := http.NewRequest("GET", "http://"+os.Args[1]+"/"+cmdStopAll, nil)
		if err != nil {
			println(err.Error())
			return
		}
		client := http.Client{}
		resp, err = client.Do(req)
		break
	case cmdStop:
		if len(os.Args) < 4 {
			printUsage()
			return
		}
		req, err := http.NewRequest("GET", "http://"+os.Args[1]+"/"+cmdStop+"?name="+os.Args[3], nil)
		if err != nil {
			println(err.Error())
			return
		}
		client := http.Client{}
		resp, err = client.Do(req)
		break
	case cmdRestart:
		if len(os.Args) < 4 {
			printUsage()
			return
		}
		req, err := http.NewRequest("GET", "http://"+os.Args[1]+"/"+cmdRestart+"?name="+os.Args[3], nil)
		if err != nil {
			println(err.Error())
			return
		}
		client := http.Client{}
		resp, err = client.Do(req)
		break
	case cmdReload:
		if len(os.Args) < 4 {
			printUsage()
			return
		}
		req, err := http.NewRequest("GET", "http://"+os.Args[1]+"/"+cmdReload+"?name="+os.Args[3], nil)
		if err != nil {
			println(err.Error())
			return
		}
		client := http.Client{}
		resp, err = client.Do(req)
		break
	}
	if err != nil {
		println(err.Error())
	} else {
		if resp.Body != nil {
			io.Copy(os.Stdout, resp.Body)
			println("")
		}
	}
}
