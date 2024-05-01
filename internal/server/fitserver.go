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
	WorkoutDay    string        `json:"workoutDay"`
}

// GetUser creates a user object for an authenticated user
func GetUser(username string) UserSession {
	return UserSession{Username: username}
}

// SetQuiet logging
func SetQuiet() {
	log.SetOutput(ioutil.Discard)
}

func makeWorkoutUpdateHandler() http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			session := GetSession(w, r)
			if session == nil {
				return
			}
			if session.Workout.Done >= len(session.Workout.Movements) {
				session.DoneForTheDay = true
				return
			}
			session.Workout.Done++
		}
	}
	return http.HandlerFunc(handler)
}

func makeFetchWorkoutHandler() http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			session := GetSession(w, r)
			if session == nil {
				return
			}
			var workoutDay string
			err := json.NewDecoder(r.Body).Decode(&workoutDay)
			if err != nil {
				log.Println("ERROR parsing workout body")
			}
			workout := model.MakeWorkout()
			if err := json.NewEncoder(w).Encode(workout); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			session.Workout = workout
			session.DoneForTheDay = false
			session.WorkoutDay = workoutDay
		}
	}
	return http.HandlerFunc(handler)
}
