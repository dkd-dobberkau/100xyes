# --- Build stage ---
FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY main.go ./
COPY web/ ./web/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/yesapi .

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/yesapi /yesapi

EXPOSE 8080

USER nonroot:nonroot

# The distroless image has no shell and no curl, so the binary probes itself.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/yesapi", "-healthcheck"]

ENTRYPOINT ["/yesapi"]
