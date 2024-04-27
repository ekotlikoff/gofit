package gateway

import (
	"embed"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	acceptableRequestPeriodMS   = 100
	maxBurstOfRequests          = 10
	maxTimeToWaitForRateLimiter = 2 * time.Second
)

var (
	rateLimiter = make(chan time.Time, maxBurstOfRequests)

	//go:embed static
	webStaticFS embed.FS

	gatewayRateLimiterMetric = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "gochess",
			Subsystem: "gateway",
			Name:      "rate_limiter_length",
			Help:      "Length of the rateLimiter channel.",
		},
		func() float64 {
			return float64(len(rateLimiter))
		},
	)

	gatewayResponseMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gochess",
			Subsystem: "gateway",
			Name:      "request_total",
			Help:      "Total number of requests serviced.",
		},
		[]string{"uri", "method", "status"},
	)

	gatewayResponseDurationMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "gochess",
			Subsystem: "gateway",
			Name:      "request_duration",
			Help:      "Duration of requests serviced.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1,
				2.5, 5, 10},
		},
		[]string{"uri", "method", "status"},
	)
)

func init() {
	prometheus.MustRegister(gatewayResponseMetric)
	prometheus.MustRegister(gatewayResponseDurationMetric)
	prometheus.MustRegister(gatewayRateLimiterMetric)
}

type (
	// Gateway is the server that serves static files and proxies to the different
	// backends
	Gateway struct {
		Backend  *url.URL
		BasePath string
		Port     int
	}

	// Credentials for authentication
	Credentials struct {
		Username string
	}
)

// Serve static files and proxy to the different backends
func (gw *Gateway) Serve() {
	cleanupChan := make(chan struct{})
	setupRateLimiter(cleanupChan)
	backendProxy := httputil.NewSingleHostReverseProxy(gw.Backend)
	mux := http.NewServeMux()
	bp := gw.BasePath
	if len(bp) > 0 && (bp[len(bp)-1:] == "/" || bp[0:1] != "/") {
		panic("Invalid gateway base path")
	}

	middleware := func(handler http.Handler) http.HandlerFunc {
		return prometheusMiddleware(rateLimiterMiddleware(handler))
	}
	mux.Handle(bp+"/", middleware(http.HandlerFunc(gw.handleWebRoot)))
	mux.Handle(bp+"/api/", middleware(backendProxy))
	// Prometheus metrics endpoint
	mux.Handle(bp+"/metrics", middleware(
		promhttp.Handler()))
	log.Println("Gateway server listening on port", gw.Port, "...")
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

func (gw *Gateway) handleWebRoot(w http.ResponseWriter, r *http.Request) {
	bp := gw.BasePath
	if len(bp) > 0 && len(r.URL.Path) > len(bp) && r.URL.Path[0:len(bp)] == bp {
		r.URL.Path = "/static" + r.URL.Path[len(bp):]
	} else {
		r.URL.Path = "/static" + r.URL.Path // This is a hack to get the embedded path
	}
	http.FileServer(http.FS(webStaticFS)).ServeHTTP(w, r)
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
		gatewayResponseMetric.WithLabelValues(
			r.URL.Path, r.Method, fmt.Sprintf("%d", sw.status)).Inc()
		gatewayResponseDurationMetric.WithLabelValues(r.URL.Path, r.Method,
			fmt.Sprintf("%d", sw.status)).Observe(duration.Seconds())
	}
}

// SetQuiet logging
func SetQuiet() {
	log.SetOutput(ioutil.Discard)
}
