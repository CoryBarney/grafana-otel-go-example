package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestGenerateSentence(t *testing.T) {
	tests := []struct {
		name            string
		input           Input
		expectedStatus  int
		expectedOutput  string
		isErrorResponse bool
	}{
		{"Empty request body", Input{}, http.StatusBadRequest, "Text input cannot be empty", true},
		{"Valid input", Input{Text: "Hello, world!"}, http.StatusOK, "Your input was: Hello, world!", false},
		{"Empty input text", Input{Text: ""}, http.StatusBadRequest, "Text input cannot be empty", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			req, err := http.NewRequest("POST", "/api/v1/sentence", bytes.NewBuffer(body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(GenerateSentence)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v, want %v", status, tt.expectedStatus)
			}

			if tt.isErrorResponse {
				// For error responses, check if the response body contains the expected error message
				if !strings.Contains(rr.Body.String(), tt.expectedOutput) {
					t.Errorf("Unexpected error message: got %v, want it to contain %v", rr.Body.String(), tt.expectedOutput)
				}
			} else {
				var response Output
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Sentence != tt.expectedOutput {
					t.Errorf("Unexpected sentence: got %v, want %v", response.Sentence, tt.expectedOutput)
				}
			}
		})
	}
}
func TestRandomDelay(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/random-delay", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(RandomDelay)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v, want %v", status, http.StatusOK)
	}

	var response DelayedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Message != "Response after random delay" {
		t.Errorf("Unexpected message: got %v, want %v", response.Message, "Response after random delay")
	}

	if response.Delay < 0 || response.Delay > 500 {
		t.Errorf("Delay was outside expected range: %v", response.Delay)
	}
}

func TestFailingEndpoint(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/fail", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(FailingEndpoint)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Handler returned wrong status code: got %v, want %v", status, http.StatusInternalServerError)
	}

	expectedBody := "Internal Server Error\n"
	if rr.Body.String() != expectedBody {
		t.Errorf("Unexpected response body: got %q, want %q", rr.Body.String(), expectedBody)
	}
}

func TestMain(m *testing.M) {
	// Initialize a no-op tracer for all tests
	tracer = noop.NewTracerProvider().Tracer("")
	// Run the tests
	m.Run()
}
