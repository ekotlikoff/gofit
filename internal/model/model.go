package model

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

const (
	Standing        = Position("standing")
	Ground          = Position("ground")
	Strength        = Modality("strength")
	Flexibility     = Modality("flexiblity")
	Mobility        = Modality("mobility")
	Hip             = Focus("hip")
	Back            = Focus("back")
	Knee            = Focus("knee")
	Shoulder        = Focus("shoulder")
	Wrist           = Focus("wrist")
	Ankle           = Focus("ankle")
	Mat             = Requirement("mat")
	Chair           = Requirement("chair")
	Band            = Requirement("band")
	Low             = Effort("low")
	Medium          = Effort("medium")
	High            = Effort("high")
	WarmupPhase     = WorkoutEffortPhase(iota)
	HighEffortPhase = iota
	CooldownPhase   = iota
	StandingPhase   = WorkoutPositionPhase(iota)
	GroundPhase     = iota

	BeginningWorkoutDuration  time.Duration = time.Duration(8 * time.Minute)
	DefaultMinStandingRatio   float64       = 1.0 / 3.0
	DefaultMaxStandingRatio   float64       = 2.0 / 3.0
	DefaultMinGroundRatio     float64       = 1.0 / 3.0
	DefaultMaxGroundRatio     float64       = 2.0 / 3.0
	DefaultMinWarmupRatio     float64       = 1.0 / 4.0
	DefaultMaxWarmupRatio     float64       = 1.0 / 3.0
	DefaultMinHighEffortRatio float64       = 1.0 / 3.0
	DefaultMaxHighEffortRatio float64       = 1.0 / 2.0
	DefaultMinCooldownRatio   float64       = 1.0 / 4.0
	DefaultMaxCooldownRatio   float64       = 1.0 / 3.0
)

var (
	//go:embed static/movements.json
	movementsFS  []byte
	movementBank []Movement
	allEfforts   = []Effort{Low, Medium, High}
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
		Name             string        `json:"name"`
		Reps             int           `json:"reps"`
		Duration         time.Duration `json:"duration"`
		IterationsPerRep int           `json:"iterationsPerRep"`
		IterationNames   []string      `json:"iterationNames"`
		Position         Position      `json:"position"`
		Modality         Modality      `json:"modality"`
		Focus            []Focus       `json:"focus"`
		SwitchSides      bool          `json:"switchSides"`
		Requirement      []Requirement `json:"requirement"`
		Effort           Effort        `json:"effort"`
	}
	Workout struct {
		Movements []Movement `json:"movements"`
		Done      int        `json:"done"`
	}
	WorkoutEffortPhase   int
	WorkoutPositionPhase int
	WorkoutPreferences   struct {
		Structure StructurePreference
		Effort    EffortPreference
		// TODO the below two are unused.
		Focus []Focus
		Other OtherPreference
	}
	StructurePreference struct {
		// Workout structure generally follows these patterns
		// First standing, then ground
		MinStandingRatio float64
		MaxStandingRatio float64
		MinGroundRatio   float64
		MaxGroundRatio   float64
		// First low/medium warmup, then high effort movements, then low/medium cooldown
		MinWarmupRatio     float64
		MaxWarmupRatio     float64
		MinHighEffortRatio float64
		MaxHighEffortRatio float64
		MinCooldownRatio   float64
		MaxCooldownRatio   float64
	}
	EffortPreference struct {
		// Soft limit to workout duration
		MaxDuration time.Duration
		// DurationMultiplier is a multiplier applied to movement durations
		DurationMultiplier float32
		// RepMultiplier is a multiplier applied to rep counts
		RepMultiplier float32
	}
	OtherPreference struct {
		PreferredModality *Modality
		PreferredPosition *Position
		RequirementFree   bool
	}
)

func (movement Movement) estimateDuration() time.Duration {
	durationPerRepWithRests := movement.Duration + (time.Second * 2)
	return durationPerRepWithRests * time.Duration(movement.Reps) * max(1, time.Duration(movement.IterationsPerRep))
}

func MakeWorkout() Workout {
	loadMovementBank()
	// TODO set workout preferences based on user's experience
	return Workout{Movements: movementSelection(defaultWorkoutPreferences()), Done: 0}
}

func defaultWorkoutPreferences() WorkoutPreferences {
	return WorkoutPreferences{
		Structure: StructurePreference{
			MinStandingRatio: DefaultMinStandingRatio, MaxStandingRatio: DefaultMaxStandingRatio,
			MinGroundRatio: DefaultMinGroundRatio, MaxGroundRatio: DefaultMaxGroundRatio,
			MinWarmupRatio: DefaultMinWarmupRatio, MinHighEffortRatio: DefaultMinHighEffortRatio,
			MinCooldownRatio: DefaultMinCooldownRatio},
		Effort: EffortPreference{MaxDuration: BeginningWorkoutDuration, DurationMultiplier: 1, RepMultiplier: 1},
	}
}

func movementSelection(workoutPreferences WorkoutPreferences) []Movement {
	selection := []Movement{}
	workoutDurations := getWorkoutDurations(workoutPreferences)
	warmupDuration, highEffortDuration := workoutDurations[0], workoutDurations[1]
	standingDuration := workoutDurations[3]
	currentDuration := time.Duration(0)
	for currentDuration < BeginningWorkoutDuration {
		efforts := effortsForPhase(getEffortPhase(currentDuration, warmupDuration, highEffortDuration))
		position := getPositionPhase(currentDuration, standingDuration)
		options := queryMovements(position, efforts)
		var thisMovement Movement
		if len(options) == 0 {
			fmt.Printf("Found no movement options for effort: %s position: %s\n", efforts, position)
			options = queryMovements(position, allEfforts)
		}
		thisMovement = options[rand.Intn(len(options))]
		selection = append(selection, thisMovement)
		estimatedRestPerMovement := time.Second * 2
		currentDuration += thisMovement.estimateDuration() + estimatedRestPerMovement
	}
	return selection
}

func effortsForPhase(phase WorkoutEffortPhase) []Effort {
	if phase == HighEffortPhase {
		return []Effort{Medium, High}
	}
	return []Effort{Low}
}

func getEffortPhase(currentDuration, warmupDuration, highEffortDuration time.Duration) WorkoutEffortPhase {
	if currentDuration > warmupDuration+highEffortDuration {
		return CooldownPhase
	} else if currentDuration > warmupDuration {
		return HighEffortPhase
	}
	return WarmupPhase
}

func getPositionPhase(currentDuration, standingDuration time.Duration) Position {
	if currentDuration > standingDuration {
		return Ground
	}
	return Standing
}

// getWorkoutRatios chooses a warmup/cooldown and standing/ground ratio.
func getWorkoutDurations(preferences WorkoutPreferences) []time.Duration {
	warmupRatio := preferences.Structure.MinWarmupRatio + ((preferences.Structure.MaxWarmupRatio - preferences.Structure.MinWarmupRatio) * rand.Float64())
	highEffortRatio := preferences.Structure.MinHighEffortRatio + ((preferences.Structure.MaxHighEffortRatio - preferences.Structure.MinHighEffortRatio) * rand.Float64())
	coolDownRatio := 1 - warmupRatio - highEffortRatio
	standingRatio := preferences.Structure.MinStandingRatio + ((preferences.Structure.MaxStandingRatio - preferences.Structure.MinStandingRatio) * rand.Float64())
	groundRatio := 1 - standingRatio
	workoutDurationMs := BeginningWorkoutDuration.Milliseconds()
	ratios := []float64{warmupRatio, highEffortRatio, coolDownRatio, standingRatio, groundRatio}
	durations := make([]time.Duration, len(ratios))
	for i, ratio := range ratios {
		durations[i] = time.Millisecond * time.Duration(int(ratio*float64(workoutDurationMs)))
	}
	return durations
}

func contains(efforts []Effort, effort Effort) bool {
	for _, e := range efforts {
		if e == effort {
			return true
		}
	}
	return false
}

func queryMovements(position Position, efforts []Effort) []Movement {
	out := []Movement{}
	for _, movement := range movementBank {
		if contains(efforts, movement.Effort) && movement.Position == position {
			out = append(out, movement)
		}
	}
	return out
}

func loadMovementBank() {
	err := json.Unmarshal(movementsFS, &movementBank)
	if err != nil {
		fmt.Println("ERROR:", err)
	}
	for i, m := range movementBank {
		movementBank[i].Duration = m.Duration * time.Second
	}
}
