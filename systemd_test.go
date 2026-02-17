package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchSystemdStatusesWithQueryParam(t *testing.T) {
	// Save original config and restore it after test
	originalConfig := config
	defer func() { config = originalConfig }()

	// Create a test server that checks for the services query parameter
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the services parameter is present
		servicesParam := r.URL.Query().Get("services")
		if servicesParam == "" {
			t.Error("Expected 'services' query parameter to be present")
		}

		// Check that the services parameter contains the expected values
		expectedServices := "service1,service2,service3"
		if servicesParam != expectedServices {
			t.Errorf("Expected services parameter to be '%s', got '%s'", expectedServices, servicesParam)
		}

		// Return a mock response
		response := map[string]string{
			"service1": "active",
			"service2": "inactive",
			"service3": "failed",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Set up test config
	config = Config{
		SystemdServices: []ServiceConfig{
			{Name: "service1", DisplayName: "Service 1"},
			{Name: "service2", DisplayName: "Service 2"},
			{Name: "service3", DisplayName: "Service 3"},
		},
		SystemdStatusURL: server.URL,
	}

	// Call the function
	statuses := fetchSystemdStatuses()

	// Verify results
	if len(statuses) != 3 {
		t.Errorf("Expected 3 statuses, got %d", len(statuses))
	}

	// Check that states were parsed correctly
	expectedStates := map[string]string{
		"service1": "active",
		"service2": "inactive",
		"service3": "failed",
	}

	for _, status := range statuses {
		expectedState, ok := expectedStates[status.Name]
		if !ok {
			t.Errorf("Unexpected service: %s", status.Name)
			continue
		}
		if status.State != expectedState {
			t.Errorf("Service %s: expected state '%s', got '%s'", status.Name, expectedState, status.State)
		}
		if status.Type != "systemd" {
			t.Errorf("Service %s: expected type 'systemd', got '%s'", status.Name, status.Type)
		}
	}
}

func TestFetchSystemdStatusesEmptyServices(t *testing.T) {
	// Save original config and restore it after test
	originalConfig := config
	defer func() { config = originalConfig }()

	// Create a test server
	requestReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		
		// Verify no services parameter is sent for empty list
		servicesParam := r.URL.Query().Get("services")
		if servicesParam != "" {
			t.Errorf("Expected no services parameter for empty list, got: %s", servicesParam)
		}
		
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, "{}")
	}))
	defer server.Close()

	// Set up test config with no services
	config = Config{
		SystemdServices:  []ServiceConfig{},
		SystemdStatusURL: server.URL,
	}

	// Call the function
	statuses := fetchSystemdStatuses()

	// Verify results
	if len(statuses) != 0 {
		t.Errorf("Expected 0 statuses, got %d", len(statuses))
	}

	// Should still make request even with empty services list
	if !requestReceived {
		t.Error("Expected request to be made even with empty services list")
	}
}

func TestFetchSystemdStatusesWithSpecialCharacters(t *testing.T) {
	// Save original config and restore it after test
	originalConfig := config
	defer func() { config = originalConfig }()

	// Create a test server that checks for properly encoded parameters
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the services parameter is present and properly decoded
		servicesParam := r.URL.Query().Get("services")
		
		// The query should be properly decoded by the server
		expectedServices := "service-1,service@test,service+plus"
		if servicesParam != expectedServices {
			t.Errorf("Expected services parameter to be '%s', got '%s'", expectedServices, servicesParam)
		}

		// Return a mock response
		response := map[string]string{
			"service-1":      "active",
			"service@test":   "inactive",
			"service+plus":   "failed",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Set up test config with services that have special characters
	config = Config{
		SystemdServices: []ServiceConfig{
			{Name: "service-1", DisplayName: "Service 1"},
			{Name: "service@test", DisplayName: "Service Test"},
			{Name: "service+plus", DisplayName: "Service Plus"},
		},
		SystemdStatusURL: server.URL,
	}

	// Call the function
	statuses := fetchSystemdStatuses()

	// Verify results
	if len(statuses) != 3 {
		t.Errorf("Expected 3 statuses, got %d", len(statuses))
	}
}

