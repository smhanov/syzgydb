package replication

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking
		return true
	},
}

func dialWebSocket(name string, myUrl string, url string, jwtSecret []byte, eventChan chan<- Event) {
	go func() {
		log.Printf("[%s] Dialing websocket to %s from %s,%s", name, url, name, myUrl)
		token, err := GenerateToken(name, myUrl, jwtSecret)
		if err != nil {
			eventChan <- WebSocketDialFailedEvent{URL: url, Error: err}
			return
		}

		header := http.Header{}
		header.Add("Authorization", "Bearer "+token)

		conn, _, err := websocket.DefaultDialer.Dial(url, header)
		if err != nil {
			eventChan <- WebSocketDialFailedEvent{URL: url, Error: err}
			return
		}

		eventChan <- WebSocketDialSucceededEvent{URL: url, Connection: conn}
	}()
}

func upgradeToWebSocket(w http.ResponseWriter, r *http.Request) (Connection, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade connection: %v", err)
	}
	return conn, nil
}
