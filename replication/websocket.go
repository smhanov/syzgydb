package replication

import (
    "fmt"
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

func dialWebSocket(url string, jwtSecret []byte) (Connection, error) {
    token, err := GenerateToken("", url, jwtSecret)
    if err != nil {
        return nil, err
    }

    header := http.Header{}
    header.Add("Authorization", "Bearer "+token)

    conn, _, err := websocket.DefaultDialer.Dial(url, header)
    if err != nil {
        return nil, err
    }

    return conn, nil
}

func upgradeToWebSocket(w http.ResponseWriter, r *http.Request) (Connection, error) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to upgrade connection: %v", err)
    }
    return conn, nil
}
