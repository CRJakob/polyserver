package webrtc

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v4"
)

type PeerSession struct {
	SessionID    string
	Peer         *webrtc.PeerConnection
	OnIceCanFunc func(iceCandidate []byte, session string) error
}

func NewPeerSession(
	sessionID string,
	offerSDP string,
	onIceCanFunc func(candidate []byte, session string) error,
	IceUrls []string,
) (*PeerSession, string, error) {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: IceUrls,
			},
		},
	}

	peer, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, "", err
	}

	ps := &PeerSession{
		SessionID:    sessionID,
		Peer:         peer,
		OnIceCanFunc: onIceCanFunc,
	}

	ps.Peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Println("Peer state:", state.String())
	})

	ps.Peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateDisconnected ||
			state == webrtc.PeerConnectionStateClosed {

			println("Cleaning up session:", ps.SessionID)
			ps.Peer.Close()
		}
	})

	ps.setupICE()

	answer, err := ps.handleOffer(offerSDP)
	if err != nil {
		return nil, "", err
	}
	return ps, answer, nil
}

func (ps *PeerSession) setupICE() {
	ps.Peer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		jsonCan, _ := json.Marshal(c.ToJSON())
		ps.OnIceCanFunc(jsonCan, ps.SessionID)
	})
}

func (ps *PeerSession) handleOffer(offerSDP string) (string, error) {

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offerSDP,
	}

	err := ps.Peer.SetRemoteDescription(offer)
	if err != nil {
		return "", err
	}

	log.Println("Creating data channels...")
	if err := ps.createDataChannels(); err != nil {
		return "", err
	}

	answer, err := ps.Peer.CreateAnswer(nil)
	if err != nil {
		return "", err
	}

	err = ps.Peer.SetLocalDescription(answer)
	if err != nil {
		return "", err
	}
	return answer.SDP, nil
}

func (ps *PeerSession) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	return ps.Peer.AddICECandidate(candidate)
}

func (ps *PeerSession) createDataChannels() error {

	// Reliable
	reliable, err := ps.Peer.CreateDataChannel(
		"reliable",
		&webrtc.DataChannelInit{
			Negotiated: func() *bool { b := true; return &b }(),
			ID:         func() *uint16 { id := uint16(0); return &id }(),
		},
	)
	if err != nil {
		return err
	}

	reliable.OnOpen(func() {
		log.Println("Reliable channel open")
	})

	reliable.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Println("Reliable message:", len(msg.Data))
	})

	// Unreliable
	unordered := false
	maxRetrans := uint16(0)
	id1 := uint16(1)
	neg := true

	unreliable, err := ps.Peer.CreateDataChannel(
		"unreliable",
		&webrtc.DataChannelInit{
			Negotiated:     &neg,
			ID:             &id1,
			Ordered:        &unordered,
			MaxRetransmits: &maxRetrans,
		},
	)
	if err != nil {
		return err
	}

	unreliable.OnOpen(func() {
		log.Println("Unreliable channel open")
	})

	unreliable.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Println("Unreliable message:", len(msg.Data))
	})

	return nil
}
