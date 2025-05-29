#!/bin/bash

# Set default environment variables
export ARCANE_HOST=${ARCANE_HOST:-"localhost"}
export ARCANE_PORT=${ARCANE_PORT:-"8080"}
export AGENT_ID=${AGENT_ID:-"agent-$(hostname)-$(date +%s)"}

# Start the agent
./bin/arcane-agent