package replication

import (
    "encoding/json"
    "log"

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
        err = json.Unmarshal(message, &msg)
        if err != nil {
            log.Printf("Error unmarshaling message from peer %s: %v", p.url, err)
            continue
        }

        switch msg.Type {
        case pb.Message_GOSSIP:
            var gossipMsg pb.GossipMessage
            err = json.Unmarshal(msg.Payload, &gossipMsg)
            if err != nil {
                log.Printf("Error unmarshaling gossip message from peer %s: %v", p.url, err)
                continue
            }
            eventChan <- GossipMessageEvent{Peer: p, Message: &gossipMsg}
        case pb.Message_UPDATE_REQUEST:
            var updateReq pb.UpdateRequest
            err = json.Unmarshal(msg.Payload, &updateReq)
            if err != nil {
                log.Printf("Error unmarshaling update request from peer %s: %v", p.url, err)
                continue
            }
            eventChan <- UpdateRequestEvent{
                Peer:       p,
                Since:      fromProtoVectorClock(updateReq.Since),
                MaxResults: int(updateReq.MaxResults),
            }
        case pb.Message_BATCH_UPDATE:
            var batchUpdate pb.BatchUpdate
            err = json.Unmarshal(msg.Payload, &batchUpdate)
            if err != nil {
                log.Printf("Error unmarshaling batch update from peer %s: %v", p.url, err)
                continue
            }
            eventChan <- BatchUpdateEvent{Peer: p, BatchUpdate: &batchUpdate}
        default:
            log.Printf("Unknown message type from peer %s: %v", p.url, msg.Type)
        }
    }
}
