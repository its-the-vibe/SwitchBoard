package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	DockerServices      []ServiceConfig `json:"dockerServices"`
	SystemdServices     []ServiceConfig `json:"systemdServices"`
	DockerStatusURL     string          `json:"dockerStatusUrl"`
	SystemdStatusURL    string          `json:"systemdStatusUrl"`
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
	Command string `json:"Command"`
	State   string `json:"State"`
	Status  string `json:"Status"`
	Names   string `json:"Names"`
	Labels  string `json:"Labels"`
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	State       string `json:"state"`
	Status      string `json:"status"`
	Type        string `json:"type"` // "docker" or "systemd"
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
	// Labels are comma-separated key=value pairs
	if container.Labels != "" {
		labels := strings.Split(container.Labels, ",")
		for _, label := range labels {
			parts := strings.SplitN(label, "=", 2)
			if len(parts) == 2 && parts[0] == "com.docker.compose.project.working_dir" {
				workDir := parts[1]
				// Extract the base name (last path segment)
				serviceName := filepath.Base(workDir)
				// filepath.Base returns "." for empty paths and "/" for root
				if serviceName != "" && serviceName != "." && serviceName != "/" {
					return serviceName
				}
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
	// Combine docker and systemd services for backward compatibility
	allServices := make([]ServiceConfig, 0, len(config.DockerServices)+len(config.SystemdServices))
	allServices = append(allServices, config.DockerServices...)
	allServices = append(allServices, config.SystemdServices...)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"services":            allServices,
		"dockerServices":      config.DockerServices,
		"systemdServices":     config.SystemdServices,
		"pollIntervalSeconds": config.PollIntervalSeconds,
	})
}

// handleStatus fetches and returns the current status of all services
func handleStatus(w http.ResponseWriter, r *http.Request) {
	statuses := make([]ServiceStatus, 0)
	
	// Fetch Docker service statuses
	dockerStatuses := fetchDockerStatuses()
	statuses = append(statuses, dockerStatuses...)
	
	// Fetch systemd service statuses
	systemdStatuses := fetchSystemdStatuses()
	statuses = append(statuses, systemdStatuses...)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// fetchDockerStatuses fetches the current status of Docker services
func fetchDockerStatuses() []ServiceStatus {
	if config.DockerStatusURL == "" {
		return []ServiceStatus{}
	}
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(config.DockerStatusURL)
	if err != nil {
		log.Printf("Error fetching docker status: %v", err)
		// Return unknown status for configured services
		return buildUnknownStatuses(config.DockerServices, "docker", "Failed to fetch Docker status")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		return buildUnknownStatuses(config.DockerServices, "docker", "Failed to read Docker status")
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
	statuses := make([]ServiceStatus, 0, len(config.DockerServices))
	for _, svc := range config.DockerServices {
		status := ServiceStatus{
			Name:        svc.Name,
			DisplayName: svc.DisplayName,
			State:       "unknown",
			Status:      "Not found",
			Type:        "docker",
		}

		// Check if container exists in docker ps output
		if container, found := containerMap[svc.Name]; found {
			status.State = container.State
			status.Status = container.Status
		}

		statuses = append(statuses, status)
	}
	
	return statuses
}

// fetchSystemdStatuses fetches the current status of systemd services
func fetchSystemdStatuses() []ServiceStatus {
	if config.SystemdStatusURL == "" {
		return []ServiceStatus{}
	}
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(config.SystemdStatusURL)
	if err != nil {
		log.Printf("Error fetching systemd status: %v", err)
		return buildUnknownStatuses(config.SystemdServices, "systemd", "Failed to fetch systemd status")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading systemd response: %v", err)
		return buildUnknownStatuses(config.SystemdServices, "systemd", "Failed to read systemd status")
	}

	// Parse systemd status JSON output
	var systemdMap map[string]string
	if err := json.Unmarshal(body, &systemdMap); err != nil {
		log.Printf("Error parsing systemd JSON: %v", err)
		return buildUnknownStatuses(config.SystemdServices, "systemd", "Failed to parse systemd status")
	}

	// Build status response for configured services
	statuses := make([]ServiceStatus, 0, len(config.SystemdServices))
	for _, svc := range config.SystemdServices {
		status := ServiceStatus{
			Name:        svc.Name,
			DisplayName: svc.DisplayName,
			State:       "unknown",
			Status:      "Not found",
			Type:        "systemd",
		}

		// Check if service exists in systemd status output
		if state, found := systemdMap[svc.Name]; found {
			status.State = state
			// Map systemd states to user-friendly status messages
			switch state {
			case "active":
				status.Status = "Active"
			case "inactive":
				status.Status = "Inactive"
			case "failed":
				status.Status = "Failed"
			case "activating":
				status.Status = "Starting"
			case "deactivating":
				status.Status = "Stopping"
			default:
				status.Status = state
			}
		}

		statuses = append(statuses, status)
	}
	
	return statuses
}

// buildUnknownStatuses creates a list of ServiceStatus with unknown state
func buildUnknownStatuses(services []ServiceConfig, serviceType string, statusMsg string) []ServiceStatus {
	statuses := make([]ServiceStatus, 0, len(services))
	for _, svc := range services {
		statuses = append(statuses, ServiceStatus{
			Name:        svc.Name,
			DisplayName: svc.DisplayName,
			State:       "unknown",
			Status:      statusMsg,
			Type:        serviceType,
		})
	}
	return statuses
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
