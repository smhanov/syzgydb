package replication

import (
	"log"
	"time"

	pb "github.com/smhanov/syzgydb/replication/proto"
	"google.golang.org/protobuf/proto"
)

// Peer represents a connected peer in the replication system.
type Peer struct {
	url                  string
	connection           Connection
	lastActive           time.Time
	lastKnownVectorClock *VectorClock
	stateMachine         *StateMachine
}

type PeerConnection struct {
	URL        string
	Connection Connection
}

// updateRequest represents a pending update request to a peer.
type updateRequest struct {
	peerURL      string
	since        *VectorClock
	inProgress   bool
	responseChan chan bool
}

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
