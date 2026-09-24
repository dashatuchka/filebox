package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const storageDir = "./data"

func main() {
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /files", uploadHandler)
	mux.HandleFunc("GET /files/{id}", downloadHandler)
	mux.HandleFunc("GET /healthz", healthHandler)

	addr := ":8000"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file field required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	id := uuid.NewString()
	dstPath := filepath.Join(storageDir, id)

	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "cannot save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "canot save file", http.StatusInternalServerError)
		return
	}

	log.Printf("uploaded %s as %s", header.Filename, id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id":"%s","url":"/files/%s"}`, id, id)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue(("id"))
	path := filepath.Join(storageDir, id)

	if filepath.Base(path) != id {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, path)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}
