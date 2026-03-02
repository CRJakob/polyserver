package webrtc_session

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v4"
)

type PeerSession struct {
	SessionID    string
	Peer         *webrtc.PeerConnection
	OnIceCanFunc func(iceCandidate []byte, session string) error
	ReliableDC   *webrtc.DataChannel
	UnreliableDC *webrtc.DataChannel
}

func NewPeerSession(
	sessionID string,
	offerSDP string,
	onIceCanFunc func(candidate []byte, session string) error,
	iceServers []webrtc.ICEServer,
	onClose func(string),
) (*PeerSession, string, error) {
	config := webrtc.Configuration{
		ICEServers: iceServers,
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
			onClose(ps.SessionID)
		}
	})

	ps.setupICE()

	answer, err := ps.handleOffer(offerSDP)
	if err != nil {
		return nil, "", err
	}
	log.Println("Returning answer...")
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
	log.Println("Data channels created!")

	answer, err := ps.Peer.CreateAnswer(nil)
	if err != nil {
		return "", err
	}
	log.Println("Answer created!")

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

	ps.ReliableDC = reliable
	ps.UnreliableDC = unreliable

	return nil
}
