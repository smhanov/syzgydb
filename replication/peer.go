package replication

import (
    "log"

    "google.golang.org/protobuf/proto"
    pb "github.com/smhanov/syzgydb/replication/proto"
)

func (p *Peer) ReadLoop(eventChan chan<- Event) {
    for {
        _, message, err := p.connection.ReadMessage()
        if err != nil {
            log.Printf("Error reading message from peer %s: %v", p.url, err)
            break
        }

        var msg pb.Message
        err = proto.Unmarshal(message, &msg)
        if err != nil {
            log.Printf("Error unmarshaling message from peer %s: %v", p.url, err)
            continue
        }

        switch msg.Type {
        case pb.Message_GOSSIP:
            if gossipMsg := msg.GetGossipMessage(); gossipMsg != nil {
                eventChan <- GossipMessageEvent{Peer: p, Message: gossipMsg}
            } else {
                log.Printf("Received GOSSIP message with nil content from peer %s", p.url)
            }
        case pb.Message_UPDATE_REQUEST:
            if updateReq := msg.GetUpdateRequest(); updateReq != nil {
                eventChan <- UpdateRequestEvent{
                    Peer:       p,
                    Since:      fromProtoVectorClock(updateReq.Since),
                    MaxResults: int(updateReq.MaxResults),
                }
            } else {
                log.Printf("Received UPDATE_REQUEST message with nil content from peer %s", p.url)
            }
        case pb.Message_BATCH_UPDATE:
            if batchUpdate := msg.GetBatchUpdate(); batchUpdate != nil {
                eventChan <- BatchUpdateEvent{Peer: p, BatchUpdate: batchUpdate}
            } else {
                log.Printf("Received BATCH_UPDATE message with nil content from peer %s", p.url)
            }
        default:
            log.Printf("Unknown message type from peer %s: %v", p.url, msg.Type)
        }
    }
}
