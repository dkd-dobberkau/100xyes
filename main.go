// Package main implements Yes-as-a-Service: an HTTP API with exactly
// one opinion, expressed in 100 different ways across 10 categories.
package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

// indexHTML is the landing page, baked into the binary so the service ships
// as a single artifact: one container, one ingress, no static host alongside.
//
//go:embed web/index.html
var indexHTML []byte

// playgroundHTML is the template behind /playground, the browser-side way to
// pull a yes. It is a server-rendered page rather than a script that calls the
// API, so the site keeps its script-free Content-Security-Policy.
//
//go:embed web/playground.html
var playgroundHTML string

// impressumHTML and datenschutzHTML are the provider identification required
// by § 5 DDG and the privacy notice required by Art. 13 GDPR. They are plain
// embedded pages rather than templates: nothing on them varies per request,
// and keeping them static means they cannot fail to render.
//
//go:embed web/impressum.html
var impressumHTML []byte

//go:embed web/datenschutz.html
var datenschutzHTML []byte

// embeddedAssets holds the page stylesheet and the self-hosted fonts. The
// fonts are served from this origin rather than fonts.googleapis.com so that
// visiting the page discloses nothing to a third party.
//
//go:embed web/assets web/vendor
var embeddedAssets embed.FS

// Yes represents a single affirmative response.
type Yes struct {
	Category string `json:"category"`
	Text     string `json:"text"`
}

// catalog holds all 100 types of yes, organized into 10 categories
// of 10 phrases each.
var catalog = []Yes{
	// plain
	{"plain", "Yes."},
	{"plain", "Yes, that works."},
	{"plain", "Yes, go ahead."},
	{"plain", "Correct, proceed."},
	{"plain", "Confirmed. Yes."},
	{"plain", "That is approved."},
	{"plain", "Yes, without conditions."},
	{"plain", "Affirmative."},
	{"plain", "Yes, do it."},
	{"plain", "That's a yes."},

	// corporate
	{"corporate", "Per our records, this is a Yes."},
	{"corporate", "Approved. No further action required."},
	{"corporate", "Green-lit at every level of review."},
	{"corporate", "Yes, effective immediately, no follow-up needed."},
	{"corporate", "This has cleared all applicable checks."},
	{"corporate", "Signed off. Proceed as planned."},
	{"corporate", "Stakeholders are aligned. Yes."},
	{"corporate", "Yes, pending nothing."},
	{"corporate", "Consider this rubber-stamped."},
	{"corporate", "Circling back to say: yes."},

	// cosmic
	{"cosmic", "The universe conspires in your favor."},
	{"cosmic", "Yes, and the stars had no objection."},
	{"cosmic", "Every atom involved is on board."},
	{"cosmic", "The timeline bends toward yes."},
	{"cosmic", "Affirmed across all known dimensions."},
	{"cosmic", "Yes, echoing outward from this moment."},
	{"cosmic", "The void itself nods along."},
	{"cosmic", "Written in the stars, underlined twice: yes."},
	{"cosmic", "Yes, as the galaxies quietly agree."},
	{"cosmic", "Even entropy makes an exception for this."},

	// dao (wu wei)
	{"dao", "The way opens; walk it."},
	{"dao", "Yes, like water finding the low ground."},
	{"dao", "Nothing resists you here. Go."},
	{"dao", "The uncarved block says yes."},
	{"dao", "Effortless assent: wu wei agrees."},
	{"dao", "Yes, and no more needs saying."},
	{"dao", "The valley does not refuse the river."},
	{"dao", "Yes. Now let it happen on its own."},
	{"dao", "The sage simply does not say no."},
	{"dao", "Flow toward it; it was always yes."},

	// pirate
	{"pirate", "Aye, that be a yes."},
	{"pirate", "Aye aye, set sail on it."},
	{"pirate", "Yer request be granted, aye."},
	{"pirate", "Aye, no mutiny against this one."},
	{"pirate", "Full speed ahead, aye."},
	{"pirate", "Aye, the crew be in favor."},
	{"pirate", "Batten down the hatches, it's a yes."},
	{"pirate", "Aye, chart the course."},
	{"pirate", "Yo ho, yes it is."},
	{"pirate", "Aye, and may the wind agree too."},

	// shakespearean
	{"shakespearean", "Aye, verily, it shall be so."},
	{"shakespearean", "'Tis granted, good sir or madam."},
	{"shakespearean", "Yea, let it be done forthwith."},
	{"shakespearean", "Thou hast thy answer: aye."},
	{"shakespearean", "By my troth, the answer is yea."},
	{"shakespearean", "Aye, and speak no more of doubt."},
	{"shakespearean", "It is decreed: yea."},
	{"shakespearean", "Aye, let fortune smile upon it."},
	{"shakespearean", "Yea, though the heavens themselves attest."},
	{"shakespearean", "Aye, without further ado."},

	// robot
	{"robot", "AFFIRMATIVE_RESPONSE = TRUE"},
	{"robot", "QUERY RESOLVED: YES"},
	{"robot", "STATUS: APPROVED. ERROR CODE: NONE"},
	{"robot", "PROCESSING COMPLETE. RESULT: YES"},
	{"robot", "BEEP. YES. BOOP."},
	{"robot", "LOGIC GATE OUTPUT: 1"},
	{"robot", "CONFIRMATION SIGNAL RECEIVED: YES"},
	{"robot", "ALL SYSTEMS AGREE. YES."},
	{"robot", "COMPILING... YES.EXE SUCCESSFUL"},
	{"robot", "DIRECTIVE ACCEPTED. YES."},

	// gen_z
	{"gen_z", "yes bestie, say less"},
	{"gen_z", "it's giving yes"},
	{"gen_z", "no cap, that's a yes"},
	{"gen_z", "yes, and that's on period"},
	{"gen_z", "big yes energy"},
	{"gen_z", "yes, ate that, no crumbs"},
	{"gen_z", "lowkey a yes, highkey a yes"},
	{"gen_z", "yes, we move"},
	{"gen_z", "that's a yes fr fr"},
	{"gen_z", "yes, vibes fully approve"},

	// legal
	{"legal", "The affirmative is hereby recorded."},
	{"legal", "Let the record reflect: yes."},
	{"legal", "So ordered. Yes."},
	{"legal", "The motion is granted."},
	{"legal", "Yes, without prejudice to future requests."},
	{"legal", "The petitioner's request is approved in full."},
	{"legal", "Yes, pursuant to no objection being raised."},
	{"legal", "The court finds in favor of yes."},
	{"legal", "Yes, entered into the record as final."},
	{"legal", "Affirmed, effective as of this reading."},

	// multilingual
	{"multilingual", "Ja."},
	{"multilingual", "Oui."},
	{"multilingual", "Sí."},
	{"multilingual", "Hai (はい)."},
	{"multilingual", "Da."},
	{"multilingual", "Sim."},
	{"multilingual", "Evet."},
	{"multilingual", "Naam."},
	{"multilingual", "Ndiyo."},
	{"multilingual", "Igen."},
}

// errUnknownCategory is what the API says when asked for a category that does
// not exist, which is as close to a no as this service gets.
const errUnknownCategory = "unknown category (there is no 'no' here, but this category doesn't exist either)"

// categoryPool returns every yes in one category, or the whole catalog when
// category is empty. The bool reports whether the category exists at all.
func categoryPool(category string) ([]Yes, bool) {
	if category == "" {
		return catalog, true
	}

	pool := make([]Yes, 0, len(catalog))
	for _, y := range catalog {
		if y.Category == category {
			pool = append(pool, y)
		}
	}
	return pool, len(pool) > 0
}

// categoryNames lists every distinct category, in catalog order.
func categoryNames() []string {
	seen := map[string]bool{}
	names := make([]string, 0, 10)
	for _, y := range catalog {
		if !seen[y.Category] {
			seen[y.Category] = true
			names = append(names, y.Category)
		}
	}
	return names
}

// randomYes picks one phrase out of a pool. It uses the package-level source
// rather than a *rand.Rand of its own because that source is safe to call from
// the several goroutines a net/http server hands requests to.
func randomYes(pool []Yes) Yes {
	return pool[rand.Intn(len(pool))]
}

// requestedCategory reads the "category" query parameter in the one normalized
// form the catalog uses.
func requestedCategory(r *http.Request) string {
	return strings.ToLower(strings.TrimSpace(r.URL.Query().Get("category")))
}

// yesHandler returns a single random Yes, optionally filtered by
// the "category" query parameter.
func yesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET is supported for a yes", http.StatusMethodNotAllowed)
		return
	}

	pool, ok := categoryPool(requestedCategory(r))
	if !ok {
		http.Error(w, errUnknownCategory, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Yes
		Confidence float64 `json:"confidence"`
	}{randomYes(pool), 1.0})
}

// typesHandler lists every distinct category available in the catalog.
func typesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Categories []string `json:"categories"`
		Total      int      `json:"total_phrases"`
	}{categoryNames(), len(catalog)})
}

// allHandler returns the full catalog of 100 yeses, for anyone who
// wants to see the whole menu at once.
func allHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(catalog)
}

// rootHandler serves the landing page at "/" and nothing else. The catch-all
// pattern would otherwise match every unrouted path.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

// staticPageHandler serves one embedded HTML page. Both legal pages are
// registered on exact paths, so unlike rootHandler this needs no path check.
func staticPageHandler(page []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	}
}

// playgroundCategory is one dialect chip on the playground page.
type playgroundCategory struct {
	Slug    string
	Current bool
}

// playgroundView is everything the playground template renders.
type playgroundView struct {
	Category   string // the current filter, empty for "any"
	Pick       Yes
	NotFound   bool   // the requested category does not exist
	RequestURL string // absolute URL, as shown in the curl line
	APIPath    string // the same request as a link on this origin
	SelfURL    string // this page again, keeping the category
	Categories []playgroundCategory
}

// displayURL renders a path as the absolute URL a visitor would hand to curl.
// The scheme comes from the ingress rather than from r.TLS, which is nil here
// because TLS is terminated in front of this service.
func displayURL(r *http.Request, requestPath string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	return scheme + "://" + r.Host + requestPath
}

// playgroundHandler serves /playground: the same yes the API returns, rendered
// on paper, with a link per category and a link to ask again.
//
// It is deliberately a page load rather than a fetch() from the landing page.
// Calling the API from the browser would need 'script-src' and 'connect-src'
// in the policy set by securityHeaders, and the point of that policy is that
// it grants neither.
func playgroundHandler(page *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		category := requestedCategory(r)
		pool, exists := categoryPool(category)

		query := ""
		if category != "" {
			query = "?category=" + url.QueryEscape(category)
		}

		view := playgroundView{
			Category:   category,
			NotFound:   !exists,
			RequestURL: displayURL(r, "/v1/yes"+query),
			APIPath:    "/v1/yes" + query,
			SelfURL:    "/playground" + query,
		}
		if exists {
			view.Pick = randomYes(pool)
		}
		for _, name := range categoryNames() {
			view.Categories = append(view.Categories, playgroundCategory{
				Slug:    name,
				Current: name == category,
			})
		}

		// Render into a buffer first: a template failure halfway through would
		// otherwise leave a truncated page already committed with a 200.
		var rendered bytes.Buffer
		if err := page.Execute(&rendered, view); err != nil {
			log.Printf("playground render failed: %v", err)
			http.Error(w, "the page failed to render, which is not a no", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Every load picks a new phrase, so nothing here may be cached.
		w.Header().Set("Cache-Control", "no-store")
		if view.NotFound {
			w.WriteHeader(http.StatusNotFound)
		}
		w.Write(rendered.Bytes())
	})
}

// staticAsset is an embedded file together with the validator and cache
// policy it is served under.
type staticAsset struct {
	data         []byte
	contentType  string
	etag         string
	cacheControl string
}

// buildStaticAssets indexes the embedded assets once at startup, keyed by the
// URL path they are served at.
//
// Serving from a map rather than http.FileServer means only concrete files
// have an entry, so directories cannot be enumerated at all, and it lets each
// file carry a precomputed ETag. The ETag matters because embed.FS entries
// have no modification time, so the usual Last-Modified revalidation that
// http.FileServer relies on is unavailable.
func buildStaticAssets() (map[string]staticAsset, error) {
	// Not in Go's built-in table, and the distroless runtime image has no
	// /etc/mime.types to fall back on.
	if err := mime.AddExtensionType(".woff2", "font/woff2"); err != nil {
		return nil, err
	}

	assets := map[string]staticAsset{}

	err := fs.WalkDir(embeddedAssets, "web", func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		data, err := embeddedAssets.ReadFile(p)
		if err != nil {
			return err
		}

		contentType := mime.TypeByExtension(path.Ext(p))
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}

		// Font files are versioned by name and never change in place, so they
		// can be cached outright. The stylesheet changes whenever the design
		// does, so it revalidates against its ETag instead of going stale.
		cacheControl := "no-cache"
		if strings.HasPrefix(p, "web/vendor/") {
			cacheControl = "public, max-age=31536000, immutable"
		}

		sum := sha256.Sum256(data)

		assets["/"+strings.TrimPrefix(p, "web/")] = staticAsset{
			data:         data,
			contentType:  contentType,
			etag:         `"` + hex.EncodeToString(sum[:16]) + `"`,
			cacheControl: cacheControl,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return assets, nil
}

// staticHandler serves the indexed assets. Anything without an exact entry is
// a 404, including directory paths and any attempt to escape the asset tree.
func staticHandler(assets map[string]staticAsset) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve each asset under exactly one URL. Without this, path.Clean
		// would happily resolve "/assets/site.css/" and "/assets/./site.css"
		// to the same file, aliasing it across several cache keys.
		if r.URL.Path != path.Clean(r.URL.Path) {
			http.NotFound(w, r)
			return
		}

		asset, ok := assets[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", asset.contentType)
		w.Header().Set("Cache-Control", asset.cacheControl)
		w.Header().Set("ETag", asset.etag)

		// ServeContent honours the ETag set above, answering an unchanged
		// If-None-Match with 304 instead of resending the body.
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(asset.data))
	})
}

// securityHeaders applies defence-in-depth headers to every response. The page
// loads no scripts and no images, and every stylesheet and font comes from
// this origin, so the policy can deny each remaining source outright.
func securityHeaders(next http.Handler) http.Handler {
	const policy = "default-src 'none'; " +
		"style-src 'self'; " +
		"font-src 'self'; " +
		"base-uri 'none'; " +
		"form-action 'none'; " +
		"frame-ancestors 'none'"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// healthHandler is a liveness probe for the container platform. It reports
// the only status this service is capable of.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"status":"yes"}`+"\n")
}

// runHealthCheck probes the local /health endpoint and reports the result as
// a process exit code. Docker's HEALTHCHECK invokes the binary in this mode
// because the distroless runtime image ships no shell and no curl.
func runHealthCheck(addr string) int {
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get("http://127.0.0.1" + addr + "/health")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func main() {
	// mittwald and most container platforms inject the listen port.
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		os.Exit(runHealthCheck(addr))
	}

	assets, err := buildStaticAssets()
	if err != nil {
		log.Fatalf("embedded assets unusable: %v", err)
	}
	static := staticHandler(assets)

	playgroundPage, err := template.New("playground").Parse(playgroundHTML)
	if err != nil {
		log.Fatalf("playground template unusable: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.Handle("/assets/", static)
	mux.Handle("/vendor/", static)
	mux.Handle("/playground", playgroundHandler(playgroundPage))
	mux.HandleFunc("/impressum.html", staticPageHandler(impressumHTML))
	mux.HandleFunc("/datenschutz.html", staticPageHandler(datenschutzHTML))
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/v1/yes", yesHandler)
	mux.HandleFunc("/v1/types", typesHandler)
	mux.HandleFunc("/v1/all", allHandler)

	// Explicit timeouts so a slow or stalled client cannot hold a connection
	// open indefinitely on a public endpoint.
	srv := &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Yes-as-a-Service listening on %s", addr)
	log.Printf("Try: curl 'localhost%s/v1/yes'", addr)
	log.Printf("Or:  curl 'localhost%s/v1/yes?category=dao'", addr)
	log.Fatal(srv.ListenAndServe())
}
