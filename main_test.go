package main

import (
	"testing"
)

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name      string
		container DockerContainer
		expected  string
	}{
		{
			name: "Extract from working directory label",
			container: DockerContainer{
				Names:  "/some-container-name",
				Labels: "com.docker.compose.config-hash=abc123,com.docker.compose.project.working_dir=/path/to/repo/github-dispatcher,com.docker.compose.service=github-dispatcher",
			},
			expected: "github-dispatcher",
		},
		{
			name: "Extract from working directory with trailing slash",
			container: DockerContainer{
				Names:  "/another-name",
				Labels: "com.docker.compose.project.working_dir=/path/to/repo/RediFire/,com.docker.compose.service=RediFire",
			},
			expected: "RediFire",
		},
		{
			name: "Fallback to container name when no label",
			container: DockerContainer{
				Names:  "/github-dispatcher",
				Labels: "",
			},
			expected: "github-dispatcher",
		},
		{
			name: "Fallback to container name when label is empty",
			container: DockerContainer{
				Names:  "/RediFire",
				Labels: "com.docker.compose.service=RediFire",
			},
			expected: "RediFire",
		},
		{
			name: "Extract from root path",
			container: DockerContainer{
				Names:  "/container",
				Labels: "com.docker.compose.project.working_dir=/service-name",
			},
			expected: "service-name",
		},
		{
			name: "Container name without leading slash",
			container: DockerContainer{
				Names:  "my-service",
				Labels: "",
			},
			expected: "my-service",
		},
		{
			name: "Working directory is root slash - fallback to container name",
			container: DockerContainer{
				Names:  "/my-container",
				Labels: "com.docker.compose.project.working_dir=/",
			},
			expected: "my-container",
		},
		{
			name: "Real Docker Compose example",
			container: DockerContainer{
				Names:  "innergate-innergate-1",
				Labels: "com.docker.compose.config-hash=4c5fb3dc43516abe85b37034d9be65a18e197be190ca7bdd320a6eff010a2a00,com.docker.compose.container-number=1,com.docker.compose.depends_on=,com.docker.compose.image=sha256:a622cea473bd5b7a44aef657d0e41930c937e1dd12c1e5ea714586c2cf6ae350,com.docker.compose.oneoff=False,com.docker.compose.project.config_files=/path/to/repo/InnerGate/docker-compose.yml,com.docker.compose.project.working_dir=/path/to/repo/InnerGate,com.docker.compose.project=innergate,com.docker.compose.service=innergate,com.docker.compose.version=5.0.2",
			},
			expected: "InnerGate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName(tt.container)
			if result != tt.expected {
				t.Errorf("extractServiceName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildUnknownStatuses(t *testing.T) {
	services := []ServiceConfig{
		{Name: "service1", DisplayName: "Service 1"},
		{Name: "service2", DisplayName: "Service 2"},
	}

	statuses := buildUnknownStatuses(services, "docker", "Test error")

	if len(statuses) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(statuses))
	}

	for i, status := range statuses {
		if status.Name != services[i].Name {
			t.Errorf("Expected name %s, got %s", services[i].Name, status.Name)
		}
		if status.DisplayName != services[i].DisplayName {
			t.Errorf("Expected display name %s, got %s", services[i].DisplayName, status.DisplayName)
		}
		if status.State != "unknown" {
			t.Errorf("Expected state 'unknown', got %s", status.State)
		}
		if status.Status != "Test error" {
			t.Errorf("Expected status 'Test error', got %s", status.Status)
		}
		if status.Type != "docker" {
			t.Errorf("Expected type 'docker', got %s", status.Type)
		}
	}
}
