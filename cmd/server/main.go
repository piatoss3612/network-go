package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Event struct {
	data []byte
}

type Client struct {
	id   string
	conn chan []byte
}

type SSE struct {
	eventCh      chan Event
	newConnCh    chan Client
	connClosedCh chan string
	clients      map[string]Client
	mu           sync.RWMutex
}

func New(logging ...bool) *SSE {
	s := &SSE{
		eventCh:      make(chan Event, 10),
		newConnCh:    make(chan Client),
		connClosedCh: make(chan string),
		clients:      make(map[string]Client),
		mu:           sync.RWMutex{},
	}

	go s.listen(logging...)

	return s
}

func (s *SSE) listen(logging ...bool) {
	logEvents := false
	if len(logging) > 0 {
		logEvents = logging[0]
	}

	for {
		select {
		case event := <-s.eventCh:
			if logEvents {
				log.Printf("Event received: %s", event.data)
			}

			s.mu.RLock()
			for _, client := range s.clients {
				client.conn <- event.data
			}
			s.mu.RUnlock()
		case client := <-s.newConnCh:
			if logEvents {
				log.Printf("New client connected: %s", client.id)
			}

			s.mu.Lock()
			s.clients[client.id] = client
			s.mu.Unlock()
		case id := <-s.connClosedCh:
			if logEvents {
				log.Printf("Client disconnected: %s", id)
			}

			s.mu.Lock()
			delete(s.clients, id)
			s.mu.Unlock()
		}
	}
}

func (s *SSE) Notify(data []byte) {
	s.eventCh <- Event{data: data}
}

func (s *SSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	conn := make(chan []byte)
	client := Client{id: r.RemoteAddr, conn: conn}

	s.newConnCh <- client

	defer func() {
		s.connClosedCh <- client.id
	}()

	notify := r.Context().Done()

	go func() {
		<-notify
		s.connClosedCh <- client.id
	}()

	for {
		select {
		case <-notify:
			return
		case msg := <-conn:
			fmt.Fprintf(w, "data: %s\n", msg)
			flusher.Flush()
		}
	}
}

func main() {
	sse := New(true)

	go func() {
		for {
			time.Sleep(time.Second * 3)
			sse.Notify([]byte("Hello, world!"))
		}
	}()

	http.ListenAndServe("localhost:3000", sse)
}
