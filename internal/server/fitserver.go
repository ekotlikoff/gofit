package server

import (
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
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
			// TODO add exercise list as static file and model code to choose a subset for the workout
			w.WriteHeader(http.StatusOK)
		}
	}
	return http.HandlerFunc(handler)
}
