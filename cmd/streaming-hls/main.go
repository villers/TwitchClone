package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	srtUrl      = "srt://127.0.0.1:6000?streamid=demo"
	httpPort    = 8080
	playlist    = "stream.m3u8"
	segmentName = "stream-%d.ts"
)

func main() {
	if err := startHlsServer(); err != nil {
		log.Fatal("Error starting HLS server: ", err.Error())
	}
}

func startHlsServer() error {
	go func() {
		if err := startFfmpeg(); err != nil {
			log.Fatal("Error starting FFmpeg: ", err.Error())
		}
		log.Println("FFmpeg started")
	}()

	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/"+playlist)
	})

	router.GET("/"+playlist, func(c *gin.Context) {
		c.File("hls/" + playlist)
	})

	router.GET("/:segmentName", func(c *gin.Context) {
		if strings.HasPrefix(c.Param("segmentName"), "stream-") && strings.HasSuffix(c.Param("segmentName"), ".ts") {
			c.File("hls/" + c.Param("segmentName"))
		} else {
			c.Status(http.StatusNotFound)
		}
	})

	return router.Run("127.0.0.1:" + strconv.Itoa(httpPort))
}

func startFfmpeg() error {
	var cmd *exec.Cmd
	var err error

	for i := 0; i < 5; i++ {
		cmd = exec.Command("ffmpeg",
			"-re",        // Specifies that the input should be read as fast as possible
			"-i", srtUrl, // Specifies the URL of the video source to be transcoded
			"-c:v", "libx264", // Specifies the video codec to be used (here libx264, an open-source video codec)
			"-preset", "veryfast", // Specifies the speed setting of the video encoder (here veryfast, a fast setting for low-latency video encoding)
			"-crf", "23", // Specifies the quality factor of the video encoding (here 23, a good compromise between quality and file size)
			"-c:a", "aac", // Specifies the audio codec to be used (here AAC, a standard audio codec)
			"-b:a", "128k", // Specifies the audio bit rate (here 128k, or 128 kilobits per second)
			"-f", "hls", // Specifies the output format (here HLS, or HTTP Live Streaming)
			"-hls_time", "2", // Specifies the duration of each HLS segment in seconds (here 2 seconds)
			"-hls_list_size", "6", // Specifies the maximum number of HLS segments in the playlist (here 6 segments)
			"-hls_segment_filename", "hls/"+segmentName, // Specifies the filename pattern for the HLS segments (here "stream-%d.ts")
			"hls/"+playlist, // Specifies the output playlist filename (here "stream.m3u8")
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err = cmd.Start(); err == nil {
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			err = <-done

			if err == nil {
				break
			}
		}

		log.Println("Error starting FFmpeg: ", err.Error())
		log.Println("Retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
	}

	return err
}
