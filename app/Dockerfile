# ── Stage 1: Build ──────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Cache deps separately from source code
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Tests run here — if they fail, Docker build fails — this is your CI gate
RUN go test ./... -v -count=1

# Build a fully static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" \
    -o /out/server ./...

# ── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM gcr.io/distroless/static:nonroot

# Copy only the binary — nothing else
COPY --from=builder /out/server /server

# Run as non-root user
USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/server"]
