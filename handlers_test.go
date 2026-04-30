package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleConfigReturnsServices verifies that /api/config returns all configured services.
func TestHandleConfigReturnsServices(t *testing.T) {
	originalConfig := config
	defer func() { config = originalConfig }()

	config = Config{
		DockerServices: []ServiceConfig{
			{Name: "svc-a", DisplayName: "Service A"},
		},
		SystemdServices: []ServiceConfig{
			{Name: "svc-b", DisplayName: "Service B"},
		},
		PollIntervalSeconds: 10,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()
	handleConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	services, ok := resp["services"].([]interface{})
	if !ok {
		t.Fatal("expected 'services' array in response")
	}
	if len(services) != 2 {
		t.Errorf("expected 2 combined services, got %d", len(services))
	}

	poll, ok := resp["pollIntervalSeconds"].(float64)
	if !ok || int(poll) != 10 {
		t.Errorf("expected pollIntervalSeconds=10, got %v", resp["pollIntervalSeconds"])
	}
}

// TestHandleStatusCombinesDockerAndSystemd checks that /api/status merges both service types.
func TestHandleStatusCombinesDockerAndSystemd(t *testing.T) {
	originalConfig := config
	defer func() { config = originalConfig }()

	// Docker status mock
	dockerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// docker ps --format json output (one JSON object per line)
		w.Write([]byte(`{"Command":"./app","State":"running","Status":"Up 5 minutes","Names":"/svc-a","Labels":""}` + "\n"))
	}))
	defer dockerServer.Close()

	// Systemd status mock
	systemdServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"svc-b": "active"})
	}))
	defer systemdServer.Close()

	config = Config{
		DockerServices:   []ServiceConfig{{Name: "svc-a", DisplayName: "Service A"}},
		SystemdServices:  []ServiceConfig{{Name: "svc-b", DisplayName: "Service B"}},
		DockerStatusURL:  dockerServer.URL,
		SystemdStatusURL: systemdServer.URL,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var statuses []ServiceStatus
	if err := json.Unmarshal(w.Body.Bytes(), &statuses); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}
}

// TestHandleToggleMethodNotAllowed ensures non-POST requests are rejected.
func TestHandleToggleMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/toggle", nil)
	w := httptest.NewRecorder()
	handleToggle(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// TestHandleToggleBadBody checks that a malformed body returns 400.
func TestHandleToggleBadBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/toggle", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	handleToggle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestHandleToggleForwardsRequest verifies successful toggle forwarding.
func TestHandleToggleForwardsRequest(t *testing.T) {
	originalConfig := config
	defer func() { config = originalConfig }()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	config = Config{ToggleServiceURL: backend.URL}

	body := `{"up":"svc-a"}`
	req := httptest.NewRequest(http.MethodPost, "/api/toggle", strings.NewReader(body))
	w := httptest.NewRecorder()
	handleToggle(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestHandleToggleBackendError checks that a backend failure returns a 5xx response.
func TestHandleToggleBackendError(t *testing.T) {
	originalConfig := config
	defer func() { config = originalConfig }()

	// Use an address that will refuse connections
	config = Config{ToggleServiceURL: "http://127.0.0.1:0"}

	body := `{"down":"svc-a"}`
	req := httptest.NewRequest(http.MethodPost, "/api/toggle", strings.NewReader(body))
	w := httptest.NewRecorder()
	handleToggle(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// TestFetchDockerStatusesNoURL verifies empty result when DockerStatusURL is unset.
func TestFetchDockerStatusesNoURL(t *testing.T) {
	originalConfig := config
	defer func() { config = originalConfig }()

	config = Config{
		DockerServices:  []ServiceConfig{{Name: "svc", DisplayName: "Svc"}},
		DockerStatusURL: "",
	}

	statuses := fetchDockerStatuses()
	if len(statuses) != 0 {
		t.Errorf("expected 0 statuses when URL is empty, got %d", len(statuses))
	}
}

// TestFetchDockerStatusesRunning verifies a running container is parsed correctly.
func TestFetchDockerStatusesRunning(t *testing.T) {
	originalConfig := config
	defer func() { config = originalConfig }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		line := `{"Command":"./app","State":"running","Status":"Up 10 minutes","Names":"/my-service","Labels":""}` + "\n"
		w.Write([]byte(line))
	}))
	defer server.Close()

	config = Config{
		DockerServices:  []ServiceConfig{{Name: "my-service", DisplayName: "My Service"}},
		DockerStatusURL: server.URL,
	}

	statuses := fetchDockerStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].State != "running" {
		t.Errorf("expected state 'running', got '%s'", statuses[0].State)
	}
	if statuses[0].Type != "docker" {
		t.Errorf("expected type 'docker', got '%s'", statuses[0].Type)
	}
}

// TestFetchDockerStatusesNotFound verifies unknown state when container is absent.
func TestFetchDockerStatusesNotFound(t *testing.T) {
	originalConfig := config
	defer func() { config = originalConfig }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return an empty body (no containers)
		w.Header().Set("Content-Type", "application/json")
	}))
	defer server.Close()

	config = Config{
		DockerServices:  []ServiceConfig{{Name: "missing-service", DisplayName: "Missing"}},
		DockerStatusURL: server.URL,
	}

	statuses := fetchDockerStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].State != "unknown" {
		t.Errorf("expected state 'unknown', got '%s'", statuses[0].State)
	}
}

// TestFetchDockerStatusesServerError verifies unknown statuses on HTTP failure.
func TestFetchDockerStatusesServerError(t *testing.T) {
	originalConfig := config
	defer func() { config = originalConfig }()

	// Use an address that refuses connections
	config = Config{
		DockerServices:  []ServiceConfig{{Name: "svc", DisplayName: "Svc"}},
		DockerStatusURL: "http://127.0.0.1:0",
	}

	statuses := fetchDockerStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].State != "unknown" {
		t.Errorf("expected state 'unknown', got '%s'", statuses[0].State)
	}
}

// TestFetchDockerStatusesWithLabelExtraction verifies container name extraction via labels.
func TestFetchDockerStatusesWithLabelExtraction(t *testing.T) {
	originalConfig := config
	defer func() { config = originalConfig }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		line := `{"Command":"./app","State":"exited","Status":"Exited (1) 2 hours ago","Names":"/random-name-1","Labels":"com.docker.compose.project.working_dir=/opt/services/my-service"}` + "\n"
		w.Write([]byte(line))
	}))
	defer server.Close()

	config = Config{
		DockerServices:  []ServiceConfig{{Name: "my-service", DisplayName: "My Service"}},
		DockerStatusURL: server.URL,
	}

	statuses := fetchDockerStatuses()
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].State != "exited" {
		t.Errorf("expected state 'exited', got '%s'", statuses[0].State)
	}
}
