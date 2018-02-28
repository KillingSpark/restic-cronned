package output

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/killingspark/restic-cronned/jobs"
)

//StartServer blockingly starts the server that serves information abut the queue
func StartServer(queue *jobs.JobQueue, port string) {
	http.HandleFunc("/", func(wr http.ResponseWriter, r *http.Request) {
		encodeQueue(queue, wr)
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
