package signaling

import (
	"encoding/json"
	"log"
)

func (s *Server) route(message []byte) {
	var env WebsocketResponse
	if err := json.Unmarshal(message, &env); err != nil {
		log.Println("invalid packet:", err)
		return
	}

	switch env.Type {

	case "createInvite":
		var packet CreateInviteResponse
		json.Unmarshal(message, &packet)
		s.handleCreateInvite(packet)

	case "joinInvite":
		var packet JoinInvite
		json.Unmarshal(message, &packet)
		s.handleJoinInvite(packet)

	case "iceCandidate":
		var packet IceCandidateResponse
		json.Unmarshal(message, &packet)
		s.handleICE(packet)
	}
}
