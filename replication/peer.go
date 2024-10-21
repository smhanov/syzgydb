package replication

import (
	"log"
	"time"

	pb "github.com/smhanov/syzgydb/replication/proto"
	"google.golang.org/protobuf/proto"
)

// Peer represents a connected peer in the replication system.
type Peer struct {
	name          string
	url           string
	connection    Connection
	lastActive    time.Time
	nodeSequences *NodeSequences
	stateMachine  *StateMachine
}

// NewPeer creates a new Peer instance.
func NewPeer(name, url string, sm *StateMachine) *Peer {
	return &Peer{
		name:          name,
		url:           url,
		lastActive:    time.Now(),
		nodeSequences: NewNodeSequences(),
		stateMachine:  sm,
	}
}

type PeerConnection struct {
	URL        string
	Connection Connection
}

func (p *Peer) ReadLoop(eventChan chan<- Event) {
	for {
		_, message, err := p.connection.ReadMessage()
		if err != nil {
			log.Printf("[%d]<-[%s] Peer disconnected", p.stateMachine.config.NodeID, p.name)
			break
		}

		var msg pb.Message
		err = proto.Unmarshal(message, &msg)
		if err != nil {
			log.Printf("Error unmarshaling message from peer %s: %v", p.url, err)
			continue
		}

		//log.Printf("msg is %+v", msg)

		// Update the state machine's timestamp
		p.stateMachine.updateTimestamp(Timestamp{
			UnixTime:     msg.TimeStamp.UnixTime,
			LamportClock: msg.TimeStamp.LamportClock,
		})

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
					Since:      fromProtoNodeSequences(updateReq.Since),
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
	p.connection.Close()
	p.connection = nil
}
