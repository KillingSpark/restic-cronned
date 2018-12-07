package output

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/killingspark/restic-cronned/src/jobs"
)

//StartServer blockingly starts the server that serves information abut the queue
func StartServer(queue *jobs.JobQueue, port string) {
	http.HandleFunc("/queue", func(wr http.ResponseWriter, r *http.Request) {
		encodeQueue(queue, wr)
	})
	http.HandleFunc("/stop", func(wr http.ResponseWriter, r *http.Request) {
		err := queue.StopJob(r.URL.Query().Get("name"))
		if err != nil {
			wr.Write([]byte(err.Error()))
		} else {
			wr.Write([]byte("Done"))
		}
	})
	http.HandleFunc("/restart", func(wr http.ResponseWriter, r *http.Request) {
		err := queue.RestartJob(r.URL.Query().Get("name"))
		if err != nil {
			wr.Write([]byte(err.Error()))
		}
	})
	http.HandleFunc("/reload", func(wr http.ResponseWriter, r *http.Request) {
		err := queue.ReloadJob(r.URL.Query().Get("name"))
		if err != nil {
			wr.Write([]byte(err.Error()))
		} else {
			wr.Write([]byte("Done"))
		}
	})
	http.HandleFunc("/trigger", func(wr http.ResponseWriter, r *http.Request) {
		err := queue.TriggerJob(r.URL.Query().Get("name"))
		if err != nil {
			wr.Write([]byte(err.Error()))
		} else {
			wr.Write([]byte("Done"))
		}
	})
	http.HandleFunc("/remove", func(wr http.ResponseWriter, r *http.Request) {
		err := queue.RemoveJob(r.URL.Query().Get("name"))
		if err != nil {
			wr.Write([]byte(err.Error()))
		} else {
			wr.Write([]byte("Done"))
		}
	})
	http.HandleFunc("/stopall", func(wr http.ResponseWriter, r *http.Request) {
		queue.StopAllJobs()
		wr.Write([]byte("Done"))
	})
	http.ListenAndServe(port, nil)
}

func encodeQueue(queue *jobs.JobQueue, wr io.Writer) error {
	enc := json.NewEncoder(wr)
	err := enc.Encode(queue)
	if err != nil {
		return err
	}
	return nil
}
