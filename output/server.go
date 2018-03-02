package output

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/killingspark/restic-cronned/jobs"
)

//StartServer blockingly starts the server that serves information abut the queue
func StartServer(queue *jobs.JobQueue, port string) {
	http.HandleFunc("/queue", func(wr http.ResponseWriter, r *http.Request) {
		encodeQueue(queue, wr)
	})
	http.HandleFunc("/stop", func(wr http.ResponseWriter, r *http.Request) {
		queue.StopJob(r.URL.Query().Get("name"))
	})
	http.HandleFunc("/restart", func(wr http.ResponseWriter, r *http.Request) {
		queue.RestartJob(r.URL.Query().Get("name"))
	})
	http.HandleFunc("/reload", func(wr http.ResponseWriter, r *http.Request) {
		queue.ReloadJob(r.URL.Query().Get("name"))
	})
	http.HandleFunc("/stopall", func(wr http.ResponseWriter, r *http.Request) {
		queue.StopAllJobs()
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
