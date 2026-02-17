# SwitchBoard
A webapp for switching off and switching on services

## Overview

SwitchBoard is a retro 60's control panel-inspired web application for managing Docker and systemd service states. It provides a tactile, visually engaging interface with switches and status lights for toggling services on and off.

## Features

- 🎛️ **Retro Control Panel UI** - Classic 60's aesthetic with glowing status lights and toggle switches
- 🔄 **Live Status Polling** - Automatically polls service status in the background
- 🚦 **Visual Status Indicators** - Color-coded status lights (green=running/active, red=stopped/inactive, magenta=failed, orange=unknown)
- ⚡ **Service Toggle Controls** - Turn services on and off with physical switch-style controls
- 🐳 **Docker Support** - Monitor and control Docker containers
- ⚙️ **Systemd Support** - Monitor and control systemd services
- 📋 **Configuration-Driven** - Define services and endpoints via JSON configuration

## Quick Start

### Configuration

Edit `config.json` to configure your services and endpoints:

```json
{
  "dockerServices": [
    {
      "name": "github-dispatcher",
      "displayName": "GitHub Dispatcher"
    },
    {
      "name": "RediFire",
      "displayName": "RediFire"
    }
  ],
  "systemdServices": [
    {
      "name": "poppit",
      "displayName": "Poppit"
    },
    {
      "name": "poppit-builder",
      "displayName": "Poppit Builder"
    },
    {
      "name": "thisisfine",
      "displayName": "This Is Fine"
    },
    {
      "name": "ttyd",
      "displayName": "TTYD Terminal"
    }
  ],
  "dockerStatusUrl": "http://docker-service-url/ps",
  "systemdStatusUrl": "http://systemd-service-url/status",
  "toggleServiceUrl": "http://turner-of-and-on-service/messages",
  "pollIntervalSeconds": 5
}
```

**Configuration Fields:**

- `dockerServices`: Array of Docker services to monitor
- `systemdServices`: Array of systemd services to monitor
- `dockerStatusUrl`: Endpoint that returns Docker container status (see Docker Integration below)
- `systemdStatusUrl`: Endpoint that returns systemd service status (see Systemd Integration below)
- `toggleServiceUrl`: Endpoint for sending service toggle commands
- `pollIntervalSeconds`: How often to poll for status updates

**Note:** You can configure only Docker services, only systemd services, or both. Leave the URL field empty (or omit it) for service types you don't need.

### Running the Application

#### Option 1: Using Go directly

```bash
# Build and run
go build -o switchboard
./switchboard

# Or run directly
go run main.go
```

The application will start on port 8080 by default. Open http://localhost:8080 in your browser.

To use a different port, set the `PORT` environment variable:

```bash
PORT=3000 go run main.go
```

#### Option 2: Using Docker

```bash
# Build the Docker image
docker build -t switchboard .

# Run the container
docker run -p 8080:8080 -v $(pwd)/config.json:/root/config.json switchboard
```

#### Option 3: Using Docker Compose

```bash
# Start the application
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the application
docker-compose down
```

## API Endpoints

### Backend API

- `GET /api/config` - Returns configuration (services and poll interval)
- `GET /api/status` - Returns current status of all configured services
- `POST /api/toggle` - Toggle a service on or off

### Toggle Request Format

To turn a service on:
```json
{
  "up": "service-name"
}
```

To turn a service off:
```json
{
  "down": "service-name"
}
```

## Architecture

The application consists of:

1. **Go Backend** (`main.go`) - HTTP server that:
   - Serves the static frontend
   - Proxies requests to the Docker status endpoint
   - Forwards toggle requests to the service controller

2. **Frontend** (`static/`) - Retro-styled web interface with:
   - `index.html` - Main HTML structure
   - `style.css` - 60's control panel styling
   - `app.js` - Status polling and toggle logic

3. **Configuration** (`config.json`) - Service definitions and endpoint URLs

## Docker Integration

SwitchBoard expects the Docker status endpoint to return `docker ps --output json` format:

```json
{"Command":"\"./github-dispatcher\"", "State":"running", "Status":"Up 41 minutes", "Names":"/github-dispatcher"}
{"Command":"\"/app/redifire\"", "State":"running", "Status":"Up About an hour", "Names":"/RediFire"}
```

## Systemd Integration

SwitchBoard polls the systemd status endpoint with a `services` query parameter containing a comma-separated list of service names.

**Request Format:**
```
GET http://systemd-service-url/status?services=poppit,poppit-builder,thisisfine,ttyd
```

**Expected Response:**
The endpoint should return a JSON object with service names as keys and their states as values:

```json
{
  "poppit": "active",
  "poppit-builder": "active",
  "thisisfine": "active",
  "ttyd": "inactive"
}
```

**Supported systemd states:**
- `active` - Service is running (displayed with green light)
- `inactive` - Service is stopped (displayed with red light)
- `failed` - Service has failed (displayed with magenta light)
- `activating` - Service is starting
- `deactivating` - Service is stopping

The UI will visually distinguish between Docker and systemd services with labeled badges.

## License

MIT
