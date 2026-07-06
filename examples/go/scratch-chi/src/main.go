// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type task struct {
	Title string `json:"title"`
}

type taskRequest struct {
	Task task `json:"task"`
}

type taskResponse struct {
	Tasks []task `json:"tasks"`
}

type result struct {
	Error   bool   `json:"error"`
	Message string `json:"message,omitempty"`
}

type createTaskResponse struct {
	Result result `json:"result"`
	Tasks  []task `json:"tasks,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	tasks := []task{}
	var tasksMu sync.Mutex
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tasksMu.Lock()
		defer tasksMu.Unlock()

		writeJSON(w, http.StatusOK, taskResponse{Tasks: tasks})
	})
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var req taskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, createTaskResponse{Result: result{Error: true, Message: "Task title is required."}})
			return
		}

		req.Task.Title = strings.TrimSpace(req.Task.Title)
		if req.Task.Title == "" {
			writeJSON(w, http.StatusBadRequest, createTaskResponse{Result: result{Error: true, Message: "Task title is required."}})
			return
		}

		tasksMu.Lock()
		defer tasksMu.Unlock()

		tasks = append(tasks, req.Task)
		writeJSON(w, http.StatusCreated, createTaskResponse{Result: result{Error: false}, Tasks: tasks})
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		log.Printf("listening on http://localhost%s", srv.Addr)
		errc <- srv.ListenAndServe()
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigc)

	select {
	case sig := <-sigc:
		log.Printf("received %s, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("shutdown: %v", err)
		}
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
