#!/bin/bash

# Required variables
export LOCALAGI_MODEL="gemma-3-12b-it" # Change this to your desired model
export LOCALAGI_LLM_API_URL="http://localhost:11434" # Change to your API URL
export LOCALAGI_API_URL="http://localhost:11434" # This is also checked in the code

# Optional: OpenRouter API key for remote models
# export OPENROUTER_API_KEY=""
# export OPENROUTER_API_URL=""

# Optional variables with defaults
export LOCALAGI_TIMEOUT="5m"
export LOCALAGI_STATE_DIR="$(pwd)/pool"

# Run the application
go run main.go
