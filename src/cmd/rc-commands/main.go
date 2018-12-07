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
	var req *http.Request
	var err error
	if len(os.Args) > 3 {
		req, err = http.NewRequest("GET", "http://"+os.Args[1]+"/"+os.Args[2]+"?name="+os.Args[3], nil)
	} else {
		req, err = http.NewRequest("GET", "http://"+os.Args[1]+"/"+os.Args[2], nil)
	}

	if err != nil {
		println(err.Error())
		return
	}

	c := http.Client{}
	resp, err = c.Do(req)

	if err != nil {
		println(err.Error())
	} else {
		if resp != nil && resp.Body != nil {
			io.Copy(os.Stdout, resp.Body)
			println("")
		}
	}
}
