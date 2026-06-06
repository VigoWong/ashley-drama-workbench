package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ashley/drama-workbench/internal/agent"
	"github.com/ashley/drama-workbench/internal/auth"
	"github.com/ashley/drama-workbench/internal/llm"
	"github.com/ashley/drama-workbench/internal/model"
	"github.com/ashley/drama-workbench/internal/store"
)

// db is the optional persistence layer. It stays nil when DATABASE_URL is
// unset or the connection fails, in which case the server degrades gracefully:
// generation still works and the history endpoints return empty/404.
var db *store.Store

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(corsMiddleware)

	a := auth.New()
	log.Printf("auth enabled (user=%s)", a.User())

	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		s, err := store.Open(dsn)
		if err != nil {
			log.Printf("history: persistence disabled (open failed: %v)", err)
		} else if err := s.Migrate(); err != nil {
			log.Printf("history: persistence disabled (migrate failed: %v)", err)
		} else {
			db = s
			log.Printf("history: persistence enabled")
		}
	}
	if db == nil {
		log.Printf("history: persistence disabled (DATABASE_URL unset or unavailable)")
	}

	r.Post("/api/login", a.LoginHandler)
	r.With(a.Middleware).Post("/api/propose", handlePropose)
	r.With(a.Middleware).Post("/api/generate", handleGenerate)
	r.With(a.Middleware).Post("/api/refine", handleRefine)
	r.With(a.Middleware).Get("/api/history", handleHistoryList)
	r.With(a.Middleware).Get("/api/history/{id}", handleHistoryGet)
	r.With(a.Middleware).Delete("/api/history/{id}", handleHistoryDelete)
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// handlePropose returns 2-3 candidate 立意方向 for a brief as plain JSON (NOT SSE):
// {"concepts": [...]}. It is the first step of the multi-direction selection flow —
// the user picks/tweaks one, then /api/generate continues from there. This call is
// cheap (one LLM round-trip) so it does not need streaming and is not persisted.
func handlePropose(w http.ResponseWriter, r *http.Request) {
	var brief model.Brief
	if err := json.NewDecoder(r.Body).Decode(&brief); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	provider, _ := llm.FromEnv()
	concepts, err := agent.Propose(r.Context(), provider, brief)
	if err != nil {
		log.Printf("propose error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"concepts": concepts})
}

// generateRequest embeds Brief so genre/episodes/… stay at the top level (keeping
// the legacy flat body working), plus an optional Concept. When Concept is set the
// user has already picked a 立意方向 via /api/propose, so we skip the concept stage
// and run the pipeline from the bible stage onward against that chosen direction.
type generateRequest struct {
	model.Brief
	Concept *model.Concept `json:"concept"`
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	brief := req.Brief
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	provider, _ := llm.FromEnv()
	emit := func(e model.Event) {
		data, _ := json.Marshal(e)
		w.Write([]byte("data: "))
		w.Write(data)
		w.Write([]byte("\n\n"))
		flusher.Flush()
	}
	o := agent.New(provider, emit)

	var plan *model.Plan
	var err error
	if req.Concept != nil {
		// User picked (and maybe tweaked) a 立意方向: seed the plan with it and run
		// from the bible stage, skipping concept generation.
		seeded := &model.Plan{Brief: brief, Concept: *req.Concept}
		seeded.Brief.ApplyDefaults()
		plan, err = o.RunFrom(r.Context(), seeded, "bible", false, "")
	} else {
		// Legacy full run: generate the concept too.
		plan, err = o.Run(r.Context(), brief)
	}
	if err != nil {
		log.Printf("pipeline error: %v", err)
		return
	}
	// Persist asynchronously to the client: the plan has already streamed out,
	// so a storage failure is logged but never surfaces to the user.
	if db != nil && plan != nil {
		if id, err := db.Save(brief, plan); err != nil {
			log.Printf("history: save failed: %v", err)
		} else {
			log.Printf("history: saved plan %s", id)
		}
	}
}

// refineRequest is the body for /api/refine: an existing plan plus which stage
// to rerun, whether to rerun only that stage or everything from it onward, and
// an optional transient note steering the regeneration.
type refineRequest struct {
	Plan      model.Plan `json:"plan"`
	FromStage string     `json:"fromStage"`
	Only      bool       `json:"only"`
	Note      string     `json:"note"`
}

// handleRefine reruns part of the pipeline against a user-edited plan and streams
// the same SSE events as /api/generate. Refine results are intentionally NOT
// persisted to history: they are interactive drafts that would otherwise
// pollute the saved-plan list. History only ever holds full /api/generate runs.
func handleRefine(w http.ResponseWriter, r *http.Request) {
	var req refineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !agent.IsStage(req.FromStage) {
		http.Error(w, "invalid fromStage", http.StatusBadRequest)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	provider, _ := llm.FromEnv()
	emit := func(e model.Event) {
		data, _ := json.Marshal(e)
		w.Write([]byte("data: "))
		w.Write(data)
		w.Write([]byte("\n\n"))
		flusher.Flush()
	}
	o := agent.New(provider, emit)
	if _, err := o.RunFrom(r.Context(), &req.Plan, req.FromStage, req.Only, req.Note); err != nil {
		log.Printf("refine error: %v", err)
		return
	}
	// Deliberately no db.Save here — see doc comment above.
}

func handleHistoryList(w http.ResponseWriter, _ *http.Request) {
	if db == nil {
		writeJSON(w, http.StatusOK, []store.Summary{})
		return
	}
	list, err := db.List()
	if err != nil {
		log.Printf("history: list failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func handleHistoryGet(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	id := chi.URLParam(r, "id")
	rec, err := db.Get(id)
	if err != nil {
		log.Printf("history: get failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rec == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func handleHistoryDelete(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := db.Delete(chi.URLParam(r, "id")); err != nil {
		log.Printf("history: delete failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
