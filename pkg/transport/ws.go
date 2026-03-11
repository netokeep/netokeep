package transport

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

/*
Upgrade2Ws upgrade the HTTP server connection to WebSocket
*/
func Upgrade2Ws(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	// TODO: configure the size and the domain verification
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024 * 32,
		WriteBufferSize: 1024 * 32,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return upgrader.Upgrade(w, r, nil)
}

/*
IsWsRequest checks whether the request is websocket or just http request.
*/
func IsWsRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	// Get client ip address
	client := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		client = fwd
	}

	// Validate the request
	if r.Header.Get("Upgrade") != "websocket" {
		log.Printf("[WS] Refused request from: %s (not ws)", client)
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return "", false
	}

	// Find the session ID
	sid := r.Header.Get("X-Session-ID")
	if sid == "" {
		return "", false
	}

	log.Printf("New WebSocket connection form: %s", client)
	return sid, true
}
