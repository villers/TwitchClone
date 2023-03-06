package main

import (
	"TwitchClone/internal/messaging"
	"fmt"
	"log"
	"strings"

	"github.com/haivision/srtgo"
)

func main() {
	streams := messaging.New()

	options := make(map[string]string)
	options["blocking"] = "0"
	options["transtype"] = "live"
	sck := srtgo.NewSrtSocket("127.0.0.1", 6000, options)
	if err := sck.Listen(10); err != nil {
		log.Fatal("Unable to listen for SRT clients:", err)
	}

	for {
		// Wait for new connection
		s, _, err := sck.Accept()
		if err != nil {
			// Something wrong happened
			log.Println(err)
			continue
		}

		// streamid can be "name:password" for streamer or "name" for viewer
		streamID, err := s.GetSockOptString(srtgo.SRTO_STREAMID)
		if err != nil {
			log.Print("Failed to get socket streamid")
			continue
		}
		split := strings.Split(streamID, ":")

		if len(split) > 1 {
			// password was provided so it is a streamer
			name, password := split[0], split[1]
			if name != "demo" && password != "demo" {
				// check password
				log.Printf("Failed to authenticate for stream %s", name)
				s.Close()
				continue
			}

			go handleStreamer(s, streams, name)
		} else {
			// password was not provided so it is a viewer
			name := split[0]

			// Send stream
			go handleViewer(s, streams, name)
		}
	}

}

func handleStreamer(socket *srtgo.SrtSocket, streams *messaging.Streams, name string) {
	// Create stream
	stream, err := streams.Create(name)
	if err != nil {
		log.Printf("Error on stream creating: %s", err)
		socket.Close()
		return
	}

	// Create source quality
	q, err := stream.CreateQuality("source")
	if err != nil {
		log.Printf("Error on quality creating: %s", err)
		socket.Close()
		return
	}
	log.Printf("New SRT streamer for stream '%s' quality 'source'", name)

	// Read RTP packets forever and send them to the WebRTC Client
	for {
		// Create a new buffer
		// UDP packet cannot be larger than MTU (1500)
		buff := make([]byte, 1500)

		// 5s timeout
		n, err := socket.Read(buff)
		if err != nil {
			log.Println("Error occurred while reading SRT socket:", err)
			break
		}

		if n == 0 {
			// End of stream
			log.Printf("Received no bytes, stopping stream.")
			break
		}

		// Send raw data to other streams
		buff = buff[:n]
		q.Broadcast <- buff
	}

	// Close stream
	streams.Delete(name)
	socket.Close()
}

func handleViewer(socket *srtgo.SrtSocket, streams *messaging.Streams, name string) {
	// Get requested stream
	fmt.Println(streams)
	stream, err := streams.Get(name)
	if err != nil {
		log.Printf("Failed to get stream: %s", err)
		socket.Close()
		return
	}

	// Get requested quality
	// FIXME: make qualities available
	qualityName := "source"
	q, err := stream.GetQuality(qualityName)
	if err != nil {
		log.Printf("Failed to get quality: %s", err)
		socket.Close()
		return
	}
	log.Printf("New SRT viewer for stream %s quality %s", name, qualityName)

	// Register new output
	c := make(chan []byte, 1024)
	q.Register(c)
	stream.IncrementClientCount()

	// Receive data and send them
	for data := range c {
		if len(data) < 1 {
			log.Print("Remove SRT viewer because of end of stream")
			break
		}

		// Send data
		_, err := socket.Write(data)
		if err != nil {
			log.Printf("Remove SRT viewer because of sending error, %s", err)
			break
		}
	}

	// Close output
	q.Unregister(c)
	stream.DecrementClientCount()
	socket.Close()
}
