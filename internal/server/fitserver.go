package server

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/ekotlikoff/gofit/internal/model"
)

// FitServer handles http requests from clients
type FitServer struct {
	BasePath string
	Port     int
}

// Serve the http server
func (fitServer *FitServer) Serve() {
	bp := fitServer.BasePath
	if len(bp) > 0 && (bp[len(bp)-1:] == "/" || bp[0:1] != "/") {
		panic("Invalid gateway base path")
	}
	mux := http.NewServeMux()
	mux.Handle(bp+"/api/workout", makeWorkoutHandler())
	log.Println("HTTP server listening on port", fitServer.Port, "...")
	http.ListenAndServe(":"+strconv.Itoa(fitServer.Port), mux)
}

// SetQuiet logging
func SetQuiet() {
	log.SetOutput(ioutil.Discard)
}

func makeWorkoutHandler() http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			workout := model.MakeWorkout()
			if err := json.NewEncoder(w).Encode(workout); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
	return http.HandlerFunc(handler)
}
