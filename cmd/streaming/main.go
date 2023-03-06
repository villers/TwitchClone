package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	srtUrl := "srt://127.0.0.1:6000?streamid=demo" // URL du flux SRT à servir en HTTP
	httpPort := 8080                               // Port HTTP à utiliser pour servir le flux en HLS

	// Lancer FFmpeg pour lire le flux SRT et le convertir en HLS
	cmd := exec.Command("ffmpeg",
		"-i", srtUrl,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-f", "hls",
		"-hls_time", "10",
		"-hls_list_size", "6",
		"-hls_segment_filename", "hls/stream-%d.ts",
		"hls/stream.m3u8",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Démarrer le serveur HTTP pour servir HLS
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		fmt.Println(path)
		if strings.HasPrefix(path, "stream-") && strings.HasSuffix(path, ".ts") {
			http.ServeFile(w, r, "hls/"+path)
		} else if path == "stream.m3u8" {
			http.ServeFile(w, r, "hls/"+path)
		} else {
			http.NotFound(w, r)
		}
	})
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(httpPort), nil))
}
