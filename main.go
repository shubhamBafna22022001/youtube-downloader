package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type downloadRequest struct {
	URL     string `json:"url"`
	Quality string `json:"quality"` // Accepts "best", "1080p", "720p", "480p"
}

func main() {
	// Create the downloads folder if it doesn't exist.
	err := os.MkdirAll("downloads", os.ModePerm)
	if err != nil {
		log.Fatal("Error creating downloads folder:", err)
	}

	// Set up the HTTP route.
	http.HandleFunc("/api/download", downloadHandler)

	// Serve the "downloads" folder as static files (optional).
	fs := http.FileServer(http.Dir("downloads"))
	http.Handle("/downloads/", http.StripPrefix("/downloads", fs))

	// Use Railway-provided port or fallback to 8080 locally.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Go server running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	// ✅ CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// ✅ Handle preflight requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req downloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "Invalid JSON or missing 'url' field", http.StatusBadRequest)
		return
	}

	log.Printf("Download requested for URL: %s with quality: %s\n", req.URL, req.Quality)

	// Format string based on quality
	var formatString string
	switch req.Quality {
	case "1080p":
		formatString = "bv*[height<=1080][vcodec^=avc1]+ba[acodec^=mp4a]/best[ext=mp4]/best"
	case "720p":
		formatString = "bv*[height<=720][vcodec^=avc1]+ba[acodec^=mp4a]/best[ext=mp4]/best"
	case "480p":
		formatString = "bv*[height<=480][vcodec^=avc1]+ba[acodec^=mp4a]/best[ext=mp4]/best"
	default:
		formatString = "bestvideo+bestaudio/best"
	}

	cmd := exec.Command("yt-dlp",
		"-f", formatString,
		"--merge-output-format", "mp4",
		"-o", "downloads/%(title)s.%(ext)s",
		req.URL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("yt-dlp error:", err, string(output))
		http.Error(w, "Failed to download video", http.StatusInternalServerError)
		return
	}

	time.Sleep(2 * time.Second)

	downloadedFile, err := findNewestFile("downloads")
	if err != nil {
		log.Println("Error finding downloaded file:", err)
		http.Error(w, "Download succeeded but file not found", http.StatusInternalServerError)
		return
	}

	filename := filepath.Base(downloadedFile)
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(w, r, downloadedFile)
}

func findNewestFile(folder string) (string, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return "", err
	}

	var newestFile string
	var newestModTime int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		modTime := info.ModTime().Unix()
		if modTime > newestModTime {
			newestModTime = modTime
			newestFile = filepath.Join(folder, entry.Name())
		}
	}

	if newestFile == "" {
		return "", fmt.Errorf("no file found in folder %s", folder)
	}

	return newestFile, nil
}
