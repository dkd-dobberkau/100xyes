// Package main implements Yes-as-a-Service: an HTTP API with exactly
// one opinion, expressed in 100 different ways across 10 categories.
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

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

// randSource is seeded once at startup for reasonably varied output.
var randSource = rand.New(rand.NewSource(time.Now().UnixNano()))

// yesHandler returns a single random Yes, optionally filtered by
// the "category" query parameter.
func yesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET is supported for a yes", http.StatusMethodNotAllowed)
		return
	}

	category := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("category")))

	pool := catalog
	if category != "" {
		filtered := make([]Yes, 0, len(catalog))
		for _, y := range catalog {
			if y.Category == category {
				filtered = append(filtered, y)
			}
		}
		if len(filtered) == 0 {
			http.Error(w, "unknown category (there is no 'no' here, but this category doesn't exist either)", http.StatusNotFound)
			return
		}
		pool = filtered
	}

	pick := pool[randSource.Intn(len(pool))]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Yes
		Confidence float64 `json:"confidence"`
	}{pick, 1.0})
}

// typesHandler lists every distinct category available in the catalog.
func typesHandler(w http.ResponseWriter, r *http.Request) {
	seen := map[string]bool{}
	var categories []string
	for _, y := range catalog {
		if !seen[y.Category] {
			seen[y.Category] = true
			categories = append(categories, y.Category)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Categories []string `json:"categories"`
		Total      int      `json:"total_phrases"`
	}{categories, len(catalog)})
}

// allHandler returns the full catalog of 100 yeses, for anyone who
// wants to see the whole menu at once.
func allHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(catalog)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/yes", yesHandler)
	mux.HandleFunc("/v1/types", typesHandler)
	mux.HandleFunc("/v1/all", allHandler)

	addr := ":8080"
	log.Printf("Yes-as-a-Service listening on %s", addr)
	log.Printf("Try: curl 'localhost%s/v1/yes'", addr)
	log.Printf("Or:  curl 'localhost%s/v1/yes?category=dao'", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
