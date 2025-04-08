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

// downloadRequest now includes a Quality field.
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

    // (Optional) Serve the "downloads" folder as static files.
    fs := http.FileServer(http.Dir("downloads"))
    http.Handle("/downloads/", http.StripPrefix("/downloads", fs))

    log.Println("Go server running on http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
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

    // Choose a format string based on the quality requested.
    var formatString string
    switch req.Quality {
    case "1080p":
        formatString = "bv*[height<=1080][vcodec^=avc1]+ba[acodec^=mp4a]/best[ext=mp4]/best"
    case "720p":
        formatString = "bv*[height<=720][vcodec^=avc1]+ba[acodec^=mp4a]/best[ext=mp4]/best"
    case "480p":
        formatString = "bv*[height<=480][vcodec^=avc1]+ba[acodec^=mp4a]/best[ext=mp4]/best"
    default:
        // "best" or if unrecognized, simply use the overall best quality.
        formatString = "bestvideo+bestaudio/best"
    }

    // Use yt-dlp to download the video.
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

    // Optional: short sleep to ensure the file is completely written.
    time.Sleep(2 * time.Second)

    // Find the newest (most recently modified) file in the downloads folder.
    downloadedFile, err := findNewestFile("downloads")
    if err != nil {
        log.Println("Error finding downloaded file:", err)
        http.Error(w, "Download succeeded but file not found", http.StatusInternalServerError)
        return
    }

    // (Optional) Post-process via ffmpeg to make it WhatsApp/QuickTime compatible.
    // Uncomment the following block if you want to force conversion.
    /*
    convertedFile := filepath.Join("downloads", "whatsapp_compatible.mp4")
    ffmpegCmd := exec.Command("ffmpeg", "-i", downloadedFile, "-vcodec", "libx264", "-acodec", "aac", "-strict", "-2", convertedFile)
    ffmpegOutput, err := ffmpegCmd.CombinedOutput()
    if err != nil {
        log.Println("ffmpeg conversion failed:", err, string(ffmpegOutput))
        http.Error(w, "Video downloaded but failed to convert", http.StatusInternalServerError)
        return
    }
    // We use the converted file as the file to serve.
    downloadedFile = convertedFile
    */

    // Set headers to force the browser to download the file.
    filename := filepath.Base(downloadedFile)
    w.Header().Set("Content-Disposition", "attachment; filename="+filename)
    w.Header().Set("Content-Type", "application/octet-stream")

    // Stream the file back to the client.
    http.ServeFile(w, r, downloadedFile)
}

// findNewestFile returns the most recently modified file in the specified folder.
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
