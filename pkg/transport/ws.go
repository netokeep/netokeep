package transport

import (
	"github.com/gorilla/websocket"
	"net/http"
)

/*
Upgrade2Ws upgrade the HTTP server connection to WebSocket
*/
func Upgrade2Ws(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	// TODO: configure the size and the domain verification
	upgrader := websocket.Upgrader{
		ReadBufferSize:  256 * 1024,
		WriteBufferSize: 256 * 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return upgrader.Upgrade(w, r, nil)
}

/*
IsWsRequest checks whether the request is websocket or just http request.
*/
func IsWsRequest(w http.ResponseWriter, r *http.Request) (string, string, bool, bool) {
	// Get client ip address
	client := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		client = fwd
	}

	// Validate the request
	if r.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
		return "", client, false, false
	}

	// Find the session ID
	sid := r.Header.Get("X-Session-ID")
	if sid == "" {
		return "", client, false, false
	}

	// Check if the traffic forwarding is enabled for this session
	forwardTraffic := false
	if r.Header.Get("X-Forward-Traffic") == "true" {
		forwardTraffic = true
	}

	return sid, client, forwardTraffic, true
}
