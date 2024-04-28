package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	acceptableRequestPeriodMS   = 100
	maxBurstOfRequests          = 10
	maxTimeToWaitForRateLimiter = 2 * time.Second
)

var (
	sessionCache *TTLMap

	rateLimiter = make(chan time.Time, maxBurstOfRequests)

	//go:embed static
	webStaticFS embed.FS

	serverRateLimiterMetric = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "gofit",
			Subsystem: "server",
			Name:      "rate_limiter_length",
			Help:      "Length of the rateLimiter channel.",
		},
		func() float64 {
			return float64(len(rateLimiter))
		},
	)

	serverSessionMetric = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "gofit",
			Subsystem: "server",
			Name:      "session_count",
			Help:      "Total number of sessions in the cache.",
		},
		func() float64 {
			if sessionCache == nil {
				return 0
			}
			return float64(sessionCache.Len())
		},
	)

	serverResponseMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gofit",
			Subsystem: "server",
			Name:      "request_total",
			Help:      "Total number of requests serviced.",
		},
		[]string{"uri", "method", "status"},
	)

	serverResponseDurationMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "gofit",
			Subsystem: "server",
			Name:      "request_duration",
			Help:      "Duration of requests serviced.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1,
				2.5, 5, 10},
		},
		[]string{"uri", "method", "status"},
	)
)

func init() {
	monthSecs := 30 * 24 * 60 * 60
	hourSecs := 60 * 60
	sessionCache = NewTTLMap(50, monthSecs, hourSecs)
	initAuth()

	prometheus.MustRegister(serverResponseMetric)
	prometheus.MustRegister(serverResponseDurationMetric)
	prometheus.MustRegister(serverRateLimiterMetric)
}

type (
	// Server serves static files and the api
	Server struct {
		Backend  *url.URL
		BasePath string
		Port     int
	}

	// Credentials for authentication
	Credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
)

// Serve static files and proxy to the different backends
func (gw *Server) Serve() {
	cleanupChan := make(chan struct{})
	setupRateLimiter(cleanupChan)
	mux := http.NewServeMux()
	bp := gw.BasePath
	if len(bp) > 0 && (bp[len(bp)-1:] == "/" || bp[0:1] != "/") {
		panic("Invalid base path")
	}

	middleware := func(handler http.Handler) http.HandlerFunc {
		return prometheusMiddleware(rateLimiterMiddleware(handler))
	}
	mux.Handle(bp+"/", middleware(http.HandlerFunc(gw.handleWebRoot)))
	mux.Handle(bp+"/register", middleware(http.HandlerFunc(Register)))
	mux.Handle(bp+"/session", middleware(http.HandlerFunc(Session)))
	mux.Handle(bp+"/workout", middleware(makeFetchWorkoutHandler()))
	mux.Handle(bp+"/workoutUpdate", middleware(makeWorkoutUpdateHandler()))
	// Prometheus metrics endpoint
	mux.Handle(bp+"/metrics", middleware(
		promhttp.Handler()))
	log.Println("Server server listening on port", gw.Port, "...")
	http.ListenAndServe(":"+strconv.Itoa(gw.Port), mux)
	close(cleanupChan)
}

func setupRateLimiter(cleanupChan chan struct{}) {
	for i := 0; i < maxBurstOfRequests; i++ {
		rateLimiter <- time.Now()
	}
	go func() {
		ticker := time.NewTicker(acceptableRequestPeriodMS * time.Millisecond)
		defer ticker.Stop()
		for t := range ticker.C {
			select {
			case rateLimiter <- t:
			case <-cleanupChan:
				return
			}
		}
	}()
}

func (gw *Server) handleWebRoot(w http.ResponseWriter, r *http.Request) {
	bp := gw.BasePath
	if len(bp) > 0 && len(r.URL.Path) > len(bp) && r.URL.Path[0:len(bp)] == bp {
		r.URL.Path = "/static" + r.URL.Path[len(bp):]
	} else {
		r.URL.Path = "/static" + r.URL.Path // This is a hack to get the embedded path
	}
	http.FileServer(http.FS(webStaticFS)).ServeHTTP(w, r)
}

// Register a new user, credit to https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if r.Method == http.MethodPost {
		var creds Credentials
		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			log.Println("Bad request", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := hashAndStore(creds); err != nil {
			log.Println("Bad request", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		newSession(w, r, creds)
	}
}

// Session credit to https://www.sohamkamani.com/blog/2018/03/25/golang-session-authentication/
func Session(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		getSession(w, r)
	} else if r.Method == http.MethodPost {
		var creds Credentials
		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			log.Println("Bad request", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		} else if creds.Username == "" {
			log.Println("Missing username")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing username"))
			return
		} else if creds.Password == "" {
			log.Println("Missing password")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Missing password"))
			return
		}
		newSession(w, r, creds)
	}
}

func getSession(w http.ResponseWriter, r *http.Request) {
	user := GetSession(w, r)
	if user == nil {
		return
	}
	response := SessionResponse(*user)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func newSession(w http.ResponseWriter, r *http.Request, creds Credentials) {
	err := auth(creds)
	if err != nil {
		log.Println("Bad password")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Bad password"))
		return
	}
	sessionToken, err := uuid.NewV4()
	if err != nil {
		log.Println("Failed to generate session token")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionTokenStr := sessionToken.String()
	user := GetUser(creds.Username)
	log.Println("Adding to sessionCache,", creds.Username)
	err = sessionCache.Put(sessionTokenStr, &user)
	if err != nil {
		log.Println("Failed to store session token in sessionCache")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionTokenStr,
		Expires: time.Now().Add(1800 * time.Second),
	})
}

// GetSession credit to https://www.sohamkamani.com/blog/2018/03/25/golang-session-authentication/
func GetSession(w http.ResponseWriter, r *http.Request) *UserSession {
	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			log.Println("session_token is not set")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Missing session_token"))
			return nil
		}
		log.Println("ERROR", err)
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	sessionToken := c.Value
	user, err := sessionCache.Get(sessionToken)
	if err != nil {
		log.Println("ERROR token is invalid", sessionToken)
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	} else if user == nil {
		log.Println("No user found for token ", sessionToken)
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	}
	return user
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader and record the status for instrumentation
func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// SessionResponse serializable struct to send client's session
type SessionResponse UserSession

// rateLimiterMiddleware handles the request by first blocking until the rate
// limiter says it is acceptable to proceed.
func rateLimiterMiddleware(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-rateLimiter:
			handler.ServeHTTP(w, r)
		case <-time.After(maxTimeToWaitForRateLimiter):
			// TODO once we add per-session rate limiting we should consider a
			// different status code for this global rate limiting.
			w.WriteHeader(429)
		}
	}
}

// prometheusMiddleware handles the request by passing it to the real
// handler and creating time series with the request details
func prometheusMiddleware(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := statusWriter{ResponseWriter: w}
		handler.ServeHTTP(&sw, r)
		duration := time.Since(start)
		serverResponseMetric.WithLabelValues(
			r.URL.Path, r.Method, fmt.Sprintf("%d", sw.status)).Inc()
		serverResponseDurationMetric.WithLabelValues(r.URL.Path, r.Method,
			fmt.Sprintf("%d", sw.status)).Observe(duration.Seconds())
	}
}
