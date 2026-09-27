package main

import (
	"context"
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

	ctx := context.Background()

	if err := initDB(ctx); err != nil {
		log.Fatal("db init failed: %v", err)
	}
	defer db.Close()

	if err := runMigrations(ctx); err != nil {
		log.Fatal("migrations failed: %v", err)
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

	written, err := io.Copy(dst, file)
	if err != nil {
		http.Error(w, "cannot save file", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(r.Context(),
		`INSERT INTO files (id, filename, size, content_type) VALUES ($1, $2, $3, $4)`,
		id, header.Filename, written, header.Header.Get("Content-Tyoe"),
	)

	if err != nil {
		log.Printf("db insert failed: %v", err)
		http.Error(w, "cannot save metadata", http.StatusInternalServerError)
		return
	}

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

	var filename string
	err := db.QueryRow(r.Context(),
		`SELECT filename FROM files WHERE id = $1`, id,
	).Scan(&filename)

	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "file missing on disk", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%s`, filename))

	http.ServeFile(w, r, path)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok"}`)
}
