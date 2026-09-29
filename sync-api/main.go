package main

import (
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

const quotationHTMLPath = "/quotation.html"

var (
	gotenbergURL string
)

func mustLoadEnv() {
	gotenbergURL = os.Getenv("GOTENBERG_URL")
	if gotenbergURL == "" {
		log.Fatal("GOTENBERG_URL is not set — refusing to start")
	}
}

// generatePDF blocks here until Gotenberg finishes rendering — this is the
// synchronous call path we're deliberately reproducing from the old design.
func generatePDF(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(quotationHTMLPath)
	if err != nil {
		log.Printf("failed to read quotation HTML at %s: %v", quotationHTMLPath, err)
		http.Error(w, "failed to load quotation content", http.StatusInternalServerError)
		return
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", "index.html")
	if err != nil {
		http.Error(w, "failed to build request", http.StatusInternalServerError)
		return
	}
	part.Write(data)
	writer.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, gotenbergURL, &body)
	if err != nil {
		http.Error(w, "failed to build request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("gotenberg call failed: %v", err)
		http.Error(w, "pdf generation failed: "+err.Error(), http.StatusGatewayTimeout)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("gotenberg returned status %d", resp.StatusCode)
		http.Error(w, "pdf generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	io.Copy(w, resp.Body)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {
	mustLoadEnv()
	http.HandleFunc("/generate-pdf", generatePDF)
	http.HandleFunc("/health", health)
	log.Println("sync-api listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}