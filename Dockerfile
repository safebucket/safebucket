# Multi-stage build: Frontend (Node.js) + Backend (Go)
FROM node:20-alpine AS frontend-builder

# Set working directory for frontend
WORKDIR /app/web

# Copy package files and install dependencies
COPY web/package*.json ./

# Copy frontend source code
COPY web/ ./

# Build static frontend
RUN npm install && npm run build

# Backend builder stage
FROM golang:1.24-alpine AS backend-builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Copy backend source code
COPY . .

# Build the Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o safebucket main.go

# Final stage - distroless production image
FROM gcr.io/distroless/static-debian12:nonroot

# Set working directory
WORKDIR /app

# Copy built binary from backend builder
COPY --from=backend-builder /app/safebucket ./safebucket

# Copy built frontend static files from frontend builder
COPY --from=frontend-builder --chown=nonroot:nonroot /app/web/dist ./web/dist

# Copy database migrations
COPY --from=backend-builder --chown=nonroot:nonroot /app/internal/database/migrations ./internal/database/migrations

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./safebucket"]
