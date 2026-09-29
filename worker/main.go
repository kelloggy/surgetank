package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	quotationHTMLPath = "/quotation.html"
	queueKey           = "pdf-queue"
)

var (
	gotenbergURL string
	receiptHTML  []byte
	rdb          *redis.Client
	ctx          = context.Background()
)

func mustLoadEnv() {
	gotenbergURL = os.Getenv("GOTENBERG_URL")
	if gotenbergURL == "" {
		log.Fatal("GOTENBERG_URL is not set — refusing to start")
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		log.Fatal("REDIS_ADDR is not set — refusing to start")
	}
	rdb = redis.NewClient(&redis.Options{Addr: addr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("cannot reach redis at %s: %v", addr, err)
	}
}

func mustLoadQuotationHTML() {
	data, err := os.ReadFile(quotationHTMLPath)
	if err != nil {
		log.Fatalf("failed to read quotation HTML at %s: %v", quotationHTMLPath, err)
	}
	receiptHTML = data
}

// renderPDF is the exact same logic sync-api used to run directly in the
// request path. Here it runs in the background, with nobody waiting on it.
func renderPDF() ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, err
	}
	part.Write(receiptHTML)
	writer.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, gotenbergURL, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func processJob(id string) {
	log.Printf("processing job %s", id)
	rdb.HSet(ctx, "job:"+id, "status", "processing")

	pdf, err := renderPDF()
	if err != nil {
		log.Printf("job %s failed: %v", id, err)
		rdb.HSet(ctx, "job:"+id, "status", "failed")
		return
	}

	rdb.HSet(ctx, "job:"+id, "status", "done", "pdf", pdf)
	log.Printf("job %s done", id)
}

func main() {
	mustLoadEnv()
	mustLoadQuotationHTML()
	log.Println("worker started, waiting for jobs...")

	for {
		// BRPop blocks until a job appears, or times out after 5s so the
		// loop can still tick over periodically.
		result, err := rdb.BRPop(ctx, 5*time.Second, queueKey).Result()
		if err == redis.Nil {
			continue // timed out, no job — loop again
		}
		if err != nil {
			log.Printf("error reading from queue: %v", err)
			time.Sleep(time.Second)
			continue
		}
		jobID := result[1] // result[0] = queue name, result[1] = the value popped
		processJob(jobID)
	}
}