# Yes-as-a-Service

An HTTP API with exactly one opinion, expressed 100 different ways across
10 categories. No rate limit. No is not implemented.

```
$ curl https://100xyes.com/v1/yes?category=dao
{
  "category": "dao",
  "text": "The way opens; walk it.",
  "confidence": 1.0
}
```

## Endpoints

| Method | Path                    | Description                        |
|--------|--------------------------|-------------------------------------|
| GET    | `/`                      | the landing page                    |
| GET    | `/vendor/…`              | self-hosted fonts and stylesheet    |
| GET    | `/health`                | liveness probe                      |
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

The one-page site lives in `web/index.html` and is embedded into the binary
with `go:embed`, so it is served at `/` by the same process as the API. No
build step, no separate static host, one container to deploy.

Fonts (Space Mono, Rock Salt) are self-hosted under `web/vendor/` and served
from `/vendor/`, so loading the page sends no request to `fonts.googleapis.com`
or `fonts.gstatic.com` and discloses no visitor IP to a third party. The page
has **no external requests at all**.

To refresh the fonts, re-fetch the Google Fonts CSS with a browser
`User-Agent` (otherwise Google serves older formats than woff2), download each
`url()` target into `web/vendor/fonts/`, and point the `url()` paths in
`web/vendor/css/fonts.css` at `../fonts/`.

## Configuration

| Variable | Default | Description                  |
|----------|---------|------------------------------|
| `PORT`   | `8080`  | port the HTTP server binds to |

## Deploying

The service is stateless with negligible resource needs, so any small VM or
container host is enough. It runs at **https://100xyes.com**, hosted on
mittwald mStudio (project `p-c3xmwk`):

1. A push to `main` builds and publishes the image to
   `ghcr.io/dkd-dobberkau/100xyes:latest` via GitHub Actions. The GHCR package
   must be public, otherwise mStudio needs a registry pull secret.
2. In the mStudio project, create a container from that image exposing
   port 8080.
3. Attach an ingress for `100xyes.com` pointing at the container.

Health checks target `GET /health`, which returns `{"status":"yes"}`. The
Docker `HEALTHCHECK` runs `/yesapi -healthcheck`, which probes that endpoint
from inside the container — the distroless runtime has no shell or `curl`.

## License

MIT, see `LICENSE`.
