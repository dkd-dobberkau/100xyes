# --- Build stage ---
FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/yesapi .

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/yesapi /yesapi

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/yesapi"]
