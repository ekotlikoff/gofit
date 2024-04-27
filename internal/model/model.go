package model

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"
)

const (
	Standing    = Position("standing")
	Ground      = Position("ground")
	Strength    = Modality("strength")
	Flexibility = Modality("flexiblity")
	Mobility    = Modality("mobility")
	Hip         = Focus("hip")
	Back        = Focus("back")
	Knee        = Focus("knee")
	Shoulder    = Focus("shoulder")
	Wrist       = Focus("wrist")
	Ankle       = Focus("ankle")
	Mat         = Requirement("mat")
	Chair       = Requirement("chair")
	Band        = Requirement("band")
	Low         = Effort("low")
	Medium      = Effort("medium")
	High        = Effort("high")

	DefaultMovementCount int = 8
)

var (
	//go:embed static/movements.json
	movementsFS  []byte
	movementBank []Movement
)

type (
	// Position is the general position of the body during the movement.
	Position string
	// Modality of a movement.
	Modality string
	// Focus is a focus area for a movement.
	Focus string
	// Requirement for a movement.
	Requirement string
	// Effort required for a movement.
	Effort string
	// Movement is a specific movement.
	Movement struct {
		Name             string
		Reps             int
		Duration         time.Duration
		IterationsPerRep int
		IterationNames   []string
		Position         Position
		Modality         Modality
		Focus            []Focus
		SwitchSides      bool
		Requirement      []Requirement
		Effort           Effort
	}
	// Workout structure generally follows these patterns
	// First standing, then ground
	// First low/medium warmup, then high effort movements if applicable, then low/medium cooldown
	Workout struct {
		Movements []Movement
		Done      int
	}
	WorkoutPreferences struct {
		Focus     []Focus
		Intensity IntensityPreference
		Other     OtherPreference
	}
	IntensityPreference struct {
		// If set, strictly limits workout duration
		MaxDuration *time.Duration
		// DurationMultiplier is a multiplier applied to movement durations
		DurationMultiplier float32
		// RepMultiplier is a multiplier applied to rep counts
		RepMultiplier float32
		// MovementCountMultiplier is a multiplier applied to workout movement count
		MovementCountMultiplier float32
		// MaxHighPct is the maximum percent of the workout spent on high effort movements
		MaxHighPct *int
		//MaxHighMovements is the maximum number of high effort movements in the workout
		MaxHighMovements *int
	}
	OtherPreference struct {
		PreferredModality *Modality
		PreferredPosition *Position
		RequirementFree   bool
	}
)

func MakeWorkout() Workout {
	loadMovementBank()
	// TODO set workout preferences based on user's experience
	return Workout{Movements: movementSelection(WorkoutPreferences{}), Done: 0}
}

func movementSelection(workoutPreferences WorkoutPreferences) []Movement {
	// TODO use workout preferences and general workout structure
	selection := make([]Movement, 0, DefaultMovementCount)
	for i := 0; i < DefaultMovementCount; i++ {
		selection = append(selection, movementBank[rand.IntN(len(movementBank))])
	}
	return selection
}

func loadMovementBank() {
	err := json.Unmarshal(movementsFS, &movementBank)
	if err != nil {
		fmt.Println("ERROR:", err)
	}
}
