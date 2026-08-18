# Yes-as-a-Service

An HTTP API with exactly one opinion, expressed 100 different ways across
10 categories. No rate limit. No is not implemented.

```
$ curl localhost:8080/v1/yes?category=dao
{
  "category": "dao",
  "text": "The way opens; walk it.",
  "confidence": 1.0
}
```

## Endpoints

| Method | Path                    | Description                        |
|--------|--------------------------|-------------------------------------|
| GET    | `/v1/yes`                | one random yes, any category        |
| GET    | `/v1/yes?category=dao`   | random yes from one category        |
| GET    | `/v1/types`               | list all categories                 |
| GET    | `/v1/all`                 | the full catalog of 100             |

Categories: `plain`, `corporate`, `cosmic`, `dao`, `pirate`, `shakespearean`,
`robot`, `gen_z`, `legal`, `multilingual`.

## Run locally

```
go run main.go
```

Requires Go 1.22+. No external dependencies, standard library only.

## Run with Docker

```
docker build -t yesapi .
docker run -p 8080:8080 yesapi
```

The image is a multi-stage build (`golang:1.22-alpine` → `distroless/static`),
statically compiled, runs as `nonroot`, and lands at roughly 10-15 MB.

## Landing page

A static one-page site describing the API lives in `web/index.html`. It has
no build step and can be served as-is by any static host or placed in front
of the API via a reverse proxy.

## Deploying

Any small VM or container host is enough; the service is stateless and has
negligible resource needs. For container hosting on mittwald mStudio:

1. Push the built image to a registry (Docker Hub or GHCR).
2. In the mStudio project, create a container referencing that image on
   port 8080.
3. Attach an ingress resource with the target domain once one is chosen.

Domain: TBD.

## License

MIT, see `LICENSE`.
