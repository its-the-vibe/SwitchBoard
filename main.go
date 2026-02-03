package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	Services            []ServiceConfig `json:"services"`
	DockerStatusURL     string          `json:"dockerStatusUrl"`
	ToggleServiceURL    string          `json:"toggleServiceUrl"`
	PollIntervalSeconds int             `json:"pollIntervalSeconds"`
}

// ServiceConfig represents a service in the configuration
type ServiceConfig struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// DockerContainer represents the docker ps JSON output structure
type DockerContainer struct {
	Command string            `json:"Command"`
	State   string            `json:"State"`
	Status  string            `json:"Status"`
	Names   string            `json:"Names"`
	Labels  map[string]string `json:"Labels"`
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	State       string `json:"state"`
	Status      string `json:"status"`
}

// ToggleRequest represents a request to toggle a service
type ToggleRequest struct {
	Up   string `json:"up,omitempty"`
	Down string `json:"down,omitempty"`
}

var config Config

// extractServiceName extracts the service name from a Docker container.
// It first tries to use the com.docker.compose.project.working_dir label,
// extracting the last path segment. If that's not available, it falls back
// to using the container name.
func extractServiceName(container DockerContainer) string {
	// Try to extract from working directory label
	if workDir, ok := container.Labels["com.docker.compose.project.working_dir"]; ok && workDir != "" {
		// Extract the last path segment
		parts := strings.Split(strings.TrimSuffix(workDir, "/"), "/")
		if len(parts) > 0 {
			serviceName := parts[len(parts)-1]
			if serviceName != "" {
				return serviceName
			}
		}
	}

	// Fallback to container name
	return strings.TrimPrefix(container.Names, "/")
}

func main() {
	// Load configuration
	configFile, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	if err := json.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	// Set up HTTP routes
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/toggle", handleToggle)
	http.HandleFunc("/api/config", handleConfig)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting SwitchBoard on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleConfig returns the configuration to the frontend
func handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"services":            config.Services,
		"pollIntervalSeconds": config.PollIntervalSeconds,
	})
}

// handleStatus fetches and returns the current status of all services
func handleStatus(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(config.DockerStatusURL)
	if err != nil {
		log.Printf("Error fetching docker status: %v", err)
		http.Error(w, "Failed to fetch service status", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		http.Error(w, "Failed to read service status", http.StatusInternalServerError)
		return
	}

	// Parse docker ps JSON output (one JSON object per line)
	lines := strings.Split(string(body), "\n")
	containerMap := make(map[string]DockerContainer)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var container DockerContainer
		if err := json.Unmarshal([]byte(line), &container); err != nil {
			log.Printf("Error parsing container JSON: %v", err)
			continue
		}

		// Extract service name from container
		serviceName := extractServiceName(container)
		containerMap[serviceName] = container
	}

	// Build status response for configured services
	statuses := make([]ServiceStatus, 0, len(config.Services))
	for _, svc := range config.Services {
		status := ServiceStatus{
			Name:        svc.Name,
			DisplayName: svc.DisplayName,
			State:       "unknown",
			Status:      "Not found",
		}

		// Check if container exists in docker ps output
		if container, found := containerMap[svc.Name]; found {
			status.State = container.State
			status.Status = container.Status
		}

		statuses = append(statuses, status)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// handleToggle handles requests to toggle a service on or off
func handleToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var toggleReq ToggleRequest
	if err := json.NewDecoder(r.Body).Decode(&toggleReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Forward the toggle request to the service controller
	jsonData, err := json.Marshal(toggleReq)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Post(
		config.ToggleServiceURL,
		"application/json",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		log.Printf("Error toggling service: %v", err)
		http.Error(w, "Failed to toggle service", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		log.Printf("Toggle service returned status: %d", resp.StatusCode)
		http.Error(w, fmt.Sprintf("Service toggle failed with status %d", resp.StatusCode), resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}
