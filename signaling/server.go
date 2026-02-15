package signaling

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"polyserver/config"
	"polyserver/webrtc"
	"strconv"

	"github.com/gorilla/websocket"
)

type Server struct {
	Conn          *websocket.Conn
	ICEUrls       []string
	CurrentInvite string
	Sessions      map[string]*webrtc.PeerSession
	ClientCount   int
}

func NewServer() *Server {

	resp, err := http.Get(config.IceFetchUrl)
	if err != nil {
		log.Fatalln(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var IceServers []IceServerResponse
	err1 := json.Unmarshal(body, &IceServers)
	if err1 != nil {
		log.Fatalln("Failed: Invalid ICE server response.")
	}
	var IceUrls []string
	var numOfUrls = 0
	for _, urls := range IceServers {
		IceUrls = append(IceUrls, urls.Urls)
		numOfUrls++
	}
	log.Println("Got " + strconv.Itoa(numOfUrls) + " ICE URLs")
	return &Server{
		Sessions:    make(map[string]*webrtc.PeerSession),
		ClientCount: 1,
		ICEUrls:     IceUrls,
	}
}

func (s *Server) Connect() error {
	if s.Conn != nil {
		s.Conn.Close()
	}

	conn, _, err := websocket.DefaultDialer.Dial(config.WebsocketUrl, nil)
	if err != nil {
		return err
	}

	s.Conn = conn
	return nil
}

func (s *Server) RegenerateInvite() error {
	if err := s.Connect(); err != nil {
		return err
	}

	go s.Start()

	return s.CreateInvite()
}

func (s *Server) CreateInvite() error {
	if s.Conn == nil {
		return fmt.Errorf("not connected")
	}

	payload := map[string]interface{}{
		"version": config.PolyVersion,
		"type":    "createInvite",
	}

	data, _ := json.Marshal(payload)

	return s.Conn.WriteMessage(websocket.TextMessage, data)
}

func (s *Server) Start() {
	for {
		_, message, err := s.Conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			s.RegenerateInvite()
			return
		}

		s.route(message)
	}
}

func (s *Server) handleCreateInvite(p CreateInviteResponse) {
	s.CurrentInvite = p.InviteCode
	log.Println("Invite code:", p.InviteCode)
}

func (s *Server) handleJoinInvite(p JoinInvite) {
	log.Println("User is joining:", p.Nickname)

	session, answer, err := webrtc.NewPeerSession(
		p.Session,
		p.Offer,
		s.OnIceCandidateServer,
		s.ICEUrls,
	)
	if err != nil {
		log.Println("failed to create session:", err)
		return
	}

	s.Sessions[p.Session] = session
	log.Println("Created session:", p.Session)
	s.ClientCount++
	joinPacket, _ := json.Marshal(AcceptJoinPacket{
		Type:                    "acceptJoin",
		Version:                 config.PolyVersion,
		Session:                 p.Session,
		Mods:                    config.LoadedMods,
		IsModsVanillaCompatible: config.AcceptVanillaClients,
		CliendId:                s.ClientCount,
		Answer:                  answer,
	})
	log.Println("Answering...")

	s.send([]byte(joinPacket))
}

func (s *Server) handleICE(p IceCandidateResponse) {
	log.Println("Received ICE candidate.")
	session, ok := s.Sessions[p.Session]
	if !ok {
		log.Println("unknown session:", p.Session)
		return
	}

	err := session.AddICECandidate(p.Candidate)
	if err != nil {
		log.Println("failed to add ICE:", err)
	}
}

func (s *Server) OnIceCandidateServer(candidate []byte, session string) error {
	var iceCandidate IceCandidate
	err := json.Unmarshal(candidate, &iceCandidate)
	if err != nil {
		return err
	}
	icePacket, err := json.Marshal(IceCandidatePacket{
		Type:      "iceCandidate",
		Candidate: iceCandidate,
		Version:   config.PolyVersion,
		Session:   session,
	})
	if err != nil {
		return err
	}
	log.Println("Forwarding ICE candidate...")
	return s.send(icePacket)
}

func (s *Server) send(data []byte) error {
	return s.Conn.WriteMessage(1, data)
}
