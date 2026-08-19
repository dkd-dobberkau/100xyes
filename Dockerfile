# --- Build stage ---
# Pinned to the machine doing the building rather than to the target platform:
# the service is pure Go with cgo off, so it cross-compiles to any target from
# here. Without this the arm64 image would be built under QEMU emulation on an
# amd64 runner, which is the same result several minutes slower.
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY main.go ./
COPY web/ ./web/

# Supplied by buildx, one value per --platform entry.
ARG TARGETOS=linux
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/yesapi .

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/yesapi /yesapi

EXPOSE 8080

USER nonroot:nonroot

# The distroless image has no shell and no curl, so the binary probes itself.
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/yesapi", "-healthcheck"]

ENTRYPOINT ["/yesapi"]
