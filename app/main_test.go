package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"
)

// ... previous test functions ...

func TestRandomDelay(t *testing.T) {
	// Initialize a no-op tracer for testing
	tracer = noop.NewTracerProvider().Tracer("")

	// Create a request
	req, err := http.NewRequest("GET", "/api/v1/random-delay", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler directly
	handler := http.HandlerFunc(RandomDelay)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body structure
	var response DelayedResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Message != "Response after random delay" {
		t.Errorf("unexpected message: got %v want %v", response.Message, "Response after random delay")
	}

	if response.Delay < 0 || response.Delay > 500 {
		t.Errorf("delay was outside expected range: %v", response.Delay)
	}
}

func TestMain(m *testing.M) {
	// Initialize a no-op tracer for all tests
	tracer = noop.NewTracerProvider().Tracer("")

	// Run the tests
	m.Run()
}
