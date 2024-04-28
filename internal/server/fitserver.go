package server

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/ekotlikoff/gofit/internal/model"
)

type UserSession struct {
	Username      string        `json:"username"`
	DoneForTheDay bool          `json:"doneForTheDay"`
	Workout       model.Workout `json:"workout"`
}

// GetUser creates a user object for an authenticated user
func GetUser(username string) UserSession {
	// TODO get user's state (DoneForTheDay/Done/Workout)
	return UserSession{Username: username}
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
			session := GetSession(w, r)
			session.Workout = workout
		case "POST":
			session := GetSession(w, r)
			if session.Workout.Done >= len(session.Workout.Movements) {
				session.DoneForTheDay = true
				return
			}
			session.Workout.Done++
		}
	}
	return http.HandlerFunc(handler)
}
