package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

const queueKey = "pdf-queue"

var (
	rdb *redis.Client
	ctx = context.Background()
)

func mustLoadEnv() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		log.Fatal("REDIS_ADDR is not set — refusing to start")
	}
	rdb = redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("cannot reach redis at %s: %v", addr, err)
	}
}

func newJobID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generatePDF returns immediately — it never talks to Gotenberg itself.
// It just records the job and hands it off to a worker via the queue.
func generatePDF(w http.ResponseWriter, r *http.Request) {
	id := newJobID()

	if err := rdb.HSet(ctx, "job:"+id, "status", "pending").Err(); err != nil {
		http.Error(w, "failed to create job", http.StatusInternalServerError)
		return
	}
	if err := rdb.LPush(ctx, queueKey, id).Err(); err != nil {
		http.Error(w, "failed to queue job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": id,
		"status": "pending",
	})
}

func jobStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/status/")
	if id == "" {
		http.Error(w, "missing job id", http.StatusBadRequest)
		return
	}

	status, err := rdb.HGet(ctx, "job:"+id, "status").Result()
	if err == redis.Nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "failed to read job status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": id,
		"status": status,
	})
}

// downloadResult serves the finished PDF once status is "done".
func downloadResult(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/result/")
	if id == "" {
		http.Error(w, "missing job id", http.StatusBadRequest)
		return
	}

	status, err := rdb.HGet(ctx, "job:"+id, "status").Result()
	if err == redis.Nil || status != "done" {
		http.Error(w, "job not ready", http.StatusNotFound)
		return
	}

	pdf, err := rdb.HGet(ctx, "job:"+id, "pdf").Bytes()
	if err != nil {
		http.Error(w, "failed to read result", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Write(pdf)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {
	mustLoadEnv()
	http.HandleFunc("/generate-pdf", generatePDF)
	http.HandleFunc("/status/", jobStatus)
	http.HandleFunc("/result/", downloadResult)
	http.HandleFunc("/health", health)
	log.Println("async-api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}