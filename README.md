# Yes-as-a-Service

An HTTP API with exactly one opinion, expressed 100 different ways across
10 categories. No rate limit. No is not implemented.

```
$ curl "https://100xyes.com/v1/yes?category=dao"
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
| GET    | `/playground`            | try the API in the browser          |
| GET    | `/playground?category=…` | …in one category                    |
| GET    | `/assets/…`              | the page stylesheet                 |
| GET    | `/vendor/…`              | self-hosted fonts                   |
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
statically compiled, runs as `nonroot`, and lands at roughly 9 MB.

### Multiple architectures

`linux/amd64` and `linux/arm64` are both published, so `:latest` is a manifest
list and every host pulls the variant matching itself — an Apple Silicon
laptop included. Nothing here is architecture-specific: cgo is off, the binary
is pure Go, and `distroless/static-debian12` exists for both.

The build stage is pinned to `$BUILDPLATFORM` and reads buildx's `TARGETARCH`,
so the foreign architecture is cross-compiled by Go rather than emulated under
QEMU. To build one locally:

```
docker buildx build --platform linux/arm64 -t yesapi:arm64 --load .
```

`--load` only accepts a single platform. Building both at once produces a
manifest list, which needs a registry to push to:

```
docker buildx build --platform linux/amd64,linux/arm64 -t <registry>/yesapi --push .
```

## Landing page

The one-page site lives in `web/index.html` and is embedded into the binary
with `go:embed`, so it is served at `/` by the same process as the API. No
build step, no separate static host, one container to deploy.

## Playground

`/playground` is the browser-side way to pull a yes: it shows the `curl` line
and the JSON response the API would return, with a link per category and a
link to ask again. Its template is `web/playground.html`, also embedded.

It is server-rendered on purpose. Doing the same thing with a modal that
`fetch()`es the API would require `script-src` and `connect-src` in the policy
below, and the point of that policy is that it grants neither — so every
answer here is a plain page load instead. Responses carry `Cache-Control:
no-store`, since each load picks a new phrase.

Fonts (Space Mono, Rock Salt) are self-hosted under `web/vendor/` and served
from `/vendor/`, so loading the page sends no request to `fonts.googleapis.com`
or `fonts.gstatic.com` and discloses no visitor IP to a third party. The page
has **no external requests at all**.

Because everything the page loads comes from this origin, and it loads no
scripts and no images, every response carries a strict policy:

```
Content-Security-Policy: default-src 'none'; style-src 'self'; font-src 'self';
                         base-uri 'none'; form-action 'none'; frame-ancestors 'none'
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
```

This is why the page styles live in `web/assets/site.css` rather than an inline
`<style>` block, why no element carries a `style=` attribute, and why the
playground is a page load rather than a fetch: each of those would need the
policy widened, and defeat the point. **Keep it that way when editing the
pages.**

TLS and HSTS are the ingress layer's job and are not set here.

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
mittwald mStudio (project `p-c3xmwk`).

A push to `main` builds and publishes the image for `linux/amd64` and
`linux/arm64` to `ghcr.io/dkd-dobberkau/100xyes:latest` via GitHub Actions. The
GHCR package is public, so mStudio pulls it without a registry secret, and the
manifest list means it keeps working if the platform ever moves the container
to arm hardware.

That push does **not** redeploy anything. To roll the new image out:

```
mw stack deploy -s <stack-id> -c docker-compose.yml
```

`docker-compose.yml` pins the service to `:latest`, so a deploy always picks
up the most recent build. Find the stack id with `mw stack list -p p-c3xmwk`.

Routing is already in place and only needs redoing if the container is
recreated with a new id:

| Hostname                 | Target                        |
|--------------------------|-------------------------------|
| `100xyes.com`            | container, port `8080/tcp`    |
| `p-c3xmwk.project.space` | container, port `8080/tcp`    |
| `www.100xyes.com`        | 301 to `https://100xyes.com`  |

```
mw domain virtualhost update <virtualhost-id> \
  --path-to-container /:<container-uuid>:8080/tcp
```

Note that `virtualhost update` replaces *all* paths of a virtual host, and
that it needs mittwald CLI 1.20 or newer. TLS and HSTS are handled by the
mStudio ingress, not by this service.

Health checks target `GET /health`, which returns `{"status":"yes"}`. The
Docker `HEALTHCHECK` runs `/yesapi -healthcheck`, which probes that endpoint
from inside the container — the distroless runtime has no shell or `curl`.

## License

MIT, see `LICENSE`.
