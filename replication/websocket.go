package replication

import (
    "net/http"

    "github.com/gorilla/websocket"
)

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
