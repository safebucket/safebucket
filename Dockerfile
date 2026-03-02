# Frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

COPY web/package*.json ./
COPY web/ ./

RUN npm install && npm run build

# Backend
FROM golang:1.25-bookworm AS backend-builder

# renovate: datasource=github-releases depName=upx/upx
ARG UPX_VERSION=5.1.0
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates tzdata curl xz-utils && rm -rf /var/lib/apt/lists/*
RUN curl -L -o upx.tar.xz https://github.com/upx/upx/releases/download/v${UPX_VERSION}/upx-${UPX_VERSION}-${TARGETARCH}_linux.tar.xz && \
    tar -xf upx.tar.xz && \
    mv upx-${UPX_VERSION}-${TARGETARCH}_linux/upx /usr/local/bin/ && \
    rm -rf upx.tar.xz upx-${UPX_VERSION}-${TARGETARCH}_linux

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /app/web/dist ./web/dist

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -a -o safebucket .
RUN upx --best --lzma safebucket
RUN mkdir -p /app/data

# Runtime
FROM gcr.io/distroless/cc-debian12:nonroot

WORKDIR /app

COPY --from=backend-builder --chown=nonroot:nonroot /app/safebucket ./safebucket
COPY --chown=nonroot:nonroot --from=backend-builder /app/data /app/data

EXPOSE 8080

CMD ["./safebucket"]
