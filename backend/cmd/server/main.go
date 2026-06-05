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
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(corsMiddleware)

	a := auth.New()
	log.Printf("auth enabled (user=%s)", a.User())

	r.Post("/api/login", a.LoginHandler)
	r.With(a.Middleware).Post("/api/generate", handleGenerate)
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	var brief model.Brief
	if err := json.NewDecoder(r.Body).Decode(&brief); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	if _, err := o.Run(r.Context(), brief); err != nil {
		log.Printf("pipeline error: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
