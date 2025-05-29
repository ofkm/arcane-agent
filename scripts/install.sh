#!/bin/bash

set -e

echo "Installing Arcane Agent..."

# Build the agent
make build

# Create systemd service (Linux)
if command -v systemctl &> /dev/null; then
    sudo cp scripts/arcane-agent.service /etc/systemd/system/
    sudo systemctl daemon-reload
    echo "Systemd service installed. Enable with: sudo systemctl enable arcane-agent"
fi

echo "Installation complete!"