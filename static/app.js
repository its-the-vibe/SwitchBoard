// SwitchBoard App - Control Panel JavaScript

let config = null;
let pollInterval = null;

// Initialize the application
async function init() {
    try {
        // Load configuration
        const configResponse = await fetch('/api/config');
        config = await configResponse.json();
        
        // Render services
        renderServices(config.services);
        
        // Start polling for status updates
        startPolling();
        
        updateSystemStatus('ONLINE');
    } catch (error) {
        console.error('Failed to initialize:', error);
        updateSystemStatus('ERROR', true);
    }
}

// Render service controls
function renderServices(services) {
    const container = document.getElementById('servicesContainer');
    container.innerHTML = '';
    
    services.forEach(service => {
        const serviceRow = document.createElement('div');
        serviceRow.className = 'service-row';
        serviceRow.id = `service-${service.name}`;
        
        serviceRow.innerHTML = `
            <div class="service-info">
                <div class="service-name">${service.displayName}</div>
                <div class="service-status" data-service="${service.name}">Checking status...</div>
            </div>
            <div class="service-controls">
                <div class="status-light unknown" data-light="${service.name}"></div>
                <div class="toggle-switch" data-switch="${service.name}" onclick="toggleService('${service.name}')">
                    <div class="toggle-handle"></div>
                </div>
            </div>
        `;
        
        container.appendChild(serviceRow);
    });
}

// Update system status
function updateSystemStatus(status, isError = false) {
    const statusElement = document.getElementById('systemStatus');
    statusElement.textContent = status;
    
    if (isError) {
        statusElement.classList.add('error');
    } else {
        statusElement.classList.remove('error');
    }
}

// Fetch and update service statuses
async function updateStatuses() {
    try {
        const response = await fetch('/api/status');
        
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        
        const statuses = await response.json();
        
        statuses.forEach(service => {
            updateServiceUI(service);
        });
        
        updateSystemStatus('ONLINE');
    } catch (error) {
        console.error('Failed to fetch statuses:', error);
        updateSystemStatus('OFFLINE', true);
    }
}

// Update UI for a single service
function updateServiceUI(service) {
    const statusElement = document.querySelector(`[data-service="${service.name}"]`);
    const lightElement = document.querySelector(`[data-light="${service.name}"]`);
    const switchElement = document.querySelector(`[data-switch="${service.name}"]`);
    
    if (!statusElement || !lightElement || !switchElement) {
        return;
    }
    
    // Update status text
    statusElement.textContent = service.status;
    
    // Update status light
    lightElement.className = 'status-light';
    if (service.state === 'running') {
        lightElement.classList.add('running');
    } else if (service.state === 'exited' || service.state === 'stopped') {
        lightElement.classList.add('stopped');
    } else {
        lightElement.classList.add('unknown');
    }
    
    // Update toggle switch
    switchElement.className = 'toggle-switch';
    if (service.state === 'running') {
        switchElement.classList.add('on');
    }
}

// Toggle a service on or off
async function toggleService(serviceName) {
    const switchElement = document.querySelector(`[data-switch="${serviceName}"]`);
    const lightElement = document.querySelector(`[data-light="${serviceName}"]`);
    
    // Prevent multiple clicks while toggling
    if (switchElement.classList.contains('disabled')) {
        return;
    }
    
    switchElement.classList.add('disabled');
    
    try {
        const isCurrentlyOn = switchElement.classList.contains('on');
        const action = isCurrentlyOn ? 'down' : 'up';
        
        const payload = {};
        payload[action] = serviceName;
        
        const response = await fetch('/api/toggle', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(payload)
        });
        
        if (!response.ok) {
            throw new Error(`Failed to toggle service: ${response.statusText}`);
        }
        
        // Optimistically update UI
        if (isCurrentlyOn) {
            switchElement.classList.remove('on');
            lightElement.className = 'status-light stopped';
        } else {
            switchElement.classList.add('on');
            lightElement.className = 'status-light running';
        }
        
        // Fetch updated status after a short delay
        setTimeout(updateStatuses, 2000);
        
    } catch (error) {
        console.error('Failed to toggle service:', error);
        updateSystemStatus('TOGGLE ERROR', true);
        
        // Reset status after error
        setTimeout(() => {
            updateSystemStatus('ONLINE');
        }, 3000);
    } finally {
        switchElement.classList.remove('disabled');
    }
}

// Start polling for status updates
function startPolling() {
    // Initial fetch
    updateStatuses();
    
    // Set up periodic polling
    const intervalSeconds = config.pollIntervalSeconds || 5;
    pollInterval = setInterval(updateStatuses, intervalSeconds * 1000);
}

// Stop polling (cleanup)
function stopPolling() {
    if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
    }
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', init);

// Clean up on page unload
window.addEventListener('beforeunload', stopPolling);
