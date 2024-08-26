package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "simple_http_sentence/docs"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

// @title Simple HTTP Sentence API
// @version 1.0
// @description This is a simple API that takes an input and returns a sentence, includes a route with random delay, and exposes Prometheus metrics.
// @host localhost:8080
// @BasePath /api/v1

type Input struct {
	Text string `json:"text" example:"Hello, world!"`
}

type Output struct {
	Sentence string `json:"sentence" example:"Your input was: Hello, world!"`
}

type DelayedResponse struct {
	Message string `json:"message"`
	Delay   int    `json:"delay_ms"`
}

var (
	tracer trace.Tracer
	rng    *rand.Rand // New global random number generator

	// Prometheus metrics
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(requestCounter)
	prometheus.MustRegister(requestDuration)

	// Initialize the random number generator with a time-based seed
	source := rand.NewSource(time.Now().UnixNano())
	rng = rand.New(source)
}

func initTracer() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint("otel-collector:4318"),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("simple_http_sentence"),
			attribute.String("environment", "local"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

// GenerateSentence godoc
// @Summary Generate a sentence from input
// @Description Takes an input text and returns a sentence
// @Accept json
// @Produce json
// @Param input body Input true "Input text"
// @Success 200 {object} Output
// @Failure 400 {string} string "Bad Request"
// @Router /sentence [post]
func GenerateSentence(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues("POST", "/api/v1/sentence").Observe(duration)
	}()

	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input format", http.StatusBadRequest)
		requestCounter.WithLabelValues("POST", "/api/v1/sentence", "400").Inc()
		return
	}

	if input.Text == "" {
		http.Error(w, "Text input cannot be empty", http.StatusBadRequest)
		requestCounter.WithLabelValues("POST", "/api/v1/sentence", "400").Inc()
		return
	}

	output := Output{
		Sentence: "Your input was: " + input.Text,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
	requestCounter.WithLabelValues("POST", "/api/v1/sentence", "200").Inc()
}

// RandomDelay godoc
// @Summary Respond after a random delay
// @Description Waits for a random time between 0-500ms and then responds
// @Produce json
// @Success 200 {object} DelayedResponse
// @Router /random-delay [get]
func RandomDelay(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var span trace.Span
	if tracer != nil {
		_, span = tracer.Start(ctx, "RandomDelay")
		defer span.End()
	}

	delay := rng.Intn(501) // Random delay between 0-500ms
	time.Sleep(time.Duration(delay) * time.Millisecond)

	response := DelayedResponse{
		Message: "Response after random delay",
		Delay:   delay,
	}

	if span != nil {
		span.SetAttributes(attribute.Int("delay.ms", delay))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// FailingEndpoint godoc
// @Summary Endpoint that always fails
// @Description This endpoint always returns a 500 Internal Server Error
// @Produce json
// @Success 500 {string} string "Internal Server Error"
// @Router /fail [get]
func FailingEndpoint(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		requestDuration.WithLabelValues("GET", "/api/v1/fail").Observe(duration)
	}()

	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	requestCounter.WithLabelValues("GET", "/api/v1/fail", "500").Inc()

}

// loggingMiddleware is a custom middleware for logging requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom ResponseWriter to capture the status code
		lrw := newLoggingResponseWriter(w)

		// Call the next handler
		next.ServeHTTP(lrw, r)

		// Log the request details
		duration := time.Since(start)
		log.Printf(
			"Method: %s | Path: %s | Status: %d | Duration: %v",
			r.Method,
			r.URL.Path,
			lrw.statusCode,
			duration,
		)
	})
}

// loggingResponseWriter is a custom ResponseWriter that captures the status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	tracer = tp.Tracer("simple_http_sentence")

	r := mux.NewRouter()

	// Apply the logging middleware to all routes
	r.Use(loggingMiddleware)

	r.Use(otelmux.Middleware("simple_http_sentence"))

	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/sentence", GenerateSentence).Methods(http.MethodPost)
	api.HandleFunc("/random-delay", RandomDelay).Methods(http.MethodGet)
	api.HandleFunc("/fail", FailingEndpoint).Methods(http.MethodGet)

	// Add Prometheus metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("Server is running on http://localhost:8080")
		log.Println("Swagger documentation is available at http://localhost:8080/swagger/index.html")
		log.Println("Prometheus metrics are available at http://localhost:8080/metrics")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
