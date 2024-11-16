#!/bin/bash

# Loki server URL
LOKI_URL="http://localhost:3100/loki/api/v1/push"

# Labels for log stream in JSON format
LABELS='{"job": "random-logs"}'

# Generate random logs
generate_log() {
  local LOG_LEVELS=("INFO" "WARNING" "ERROR" "DEBUG")
  local MESSAGES=("User login successful" "User login failed" "File uploaded" "File download failed" "Database query executed" "Cache miss" "Unexpected error occurred")

  # Pick a random log level and message
  local LEVEL=${LOG_LEVELS[$RANDOM % ${#LOG_LEVELS[@]}]}
  local MESSAGE=${MESSAGES[$RANDOM % ${#MESSAGES[@]}]}

  # Current timestamp in nanoseconds
  local TIMESTAMP=$(date +%s%N)

  # Formatted log line
  echo "{\"streams\": [{\"stream\": $LABELS, \"values\": [[\"$TIMESTAMP\", \"$LEVEL: $MESSAGE\"]]}]}"
}

# Function to send log entry to Loki
send_log() {
  local LOG=$(generate_log)
  curl -s -X POST -H "Content-Type: application/json" --data "$LOG" "$LOKI_URL" -v
}

# Main loop to generate and send logs
while true; do
  send_log
  sleep 1  # Adjust the sleep interval as desired
done
