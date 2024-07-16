package main

import (
	"bufio"
	"log"
	"net/http"
)

func main() {
	client := http.DefaultClient

	resp, err := client.Get("http://localhost:3000")
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("unexpected status code: %d", resp.StatusCode)
	}

	log.Println("Connected to server")

	// Read the stream of events
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		log.Println(scanner.Text())
	}
}
