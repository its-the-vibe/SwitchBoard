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
				Names: "/some-container-name",
				Labels: map[string]string{
					"com.docker.compose.project.working_dir": "/path/to/repo/github-dispatcher",
				},
			},
			expected: "github-dispatcher",
		},
		{
			name: "Extract from working directory with trailing slash",
			container: DockerContainer{
				Names: "/another-name",
				Labels: map[string]string{
					"com.docker.compose.project.working_dir": "/path/to/repo/RediFire/",
				},
			},
			expected: "RediFire",
		},
		{
			name: "Fallback to container name when no label",
			container: DockerContainer{
				Names:  "/github-dispatcher",
				Labels: map[string]string{},
			},
			expected: "github-dispatcher",
		},
		{
			name: "Fallback to container name when label is empty",
			container: DockerContainer{
				Names: "/RediFire",
				Labels: map[string]string{
					"com.docker.compose.project.working_dir": "",
				},
			},
			expected: "RediFire",
		},
		{
			name: "Extract from root path",
			container: DockerContainer{
				Names: "/container",
				Labels: map[string]string{
					"com.docker.compose.project.working_dir": "/service-name",
				},
			},
			expected: "service-name",
		},
		{
			name: "Container name without leading slash",
			container: DockerContainer{
				Names:  "my-service",
				Labels: map[string]string{},
			},
			expected: "my-service",
		},
		{
			name: "Working directory is root slash - fallback to container name",
			container: DockerContainer{
				Names: "/my-container",
				Labels: map[string]string{
					"com.docker.compose.project.working_dir": "/",
				},
			},
			expected: "my-container",
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
