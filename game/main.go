package game

import (
	"fmt"
	"log"
	gamepackets "polyserver/game/packets"
	gametrack "polyserver/game/track"
	"polyserver/signaling"
	webrtc_session "polyserver/webrtc"
	"sync"
)

type GameServer struct {
	SignalingServer *signaling.WebRTCServer
	Players         []*Player
	playersLock     sync.Mutex
	GameSession     *GameSession
}

type GameMode uint8

const (
	Casual GameMode = iota
	Competitive
)

func (gm GameMode) String() string {
	switch gm {
	case Casual:
		return "Casual"
	case Competitive:
		return "Competitive"
	default:
		return fmt.Sprintf("Unknown(%d)", gm)
	}
}

type GameSession struct {
	SessionID        uint32
	GameMode         GameMode
	SwitchingSession bool
	CurrentTrack     *gametrack.Track
	NewTrack         *gametrack.Track
	MaxPlayers       int
}

func NewServer(signalingServer *signaling.WebRTCServer) *GameServer {
	server := &GameServer{
		SignalingServer: signalingServer,
		Players:         make([]*Player, 0),
		GameSession:     nil,
	}
	signalingServer.OnOpen = server.onPlayerJoin
	signalingServer.OnClose = server.onPlayerDisconnect
	return server
}

func (server *GameServer) onPlayerJoin(p signaling.JoinInvite, session *webrtc_session.PeerSession) {
	log.Println("Creating player " + p.Nickname)
	newPlayer := Player{
		Session:                 session,
		IsKicked:                false,
		ID:                      server.SignalingServer.ClientCount,
		Mods:                    p.Mods,
		IsModsVanillaCompatible: p.IsModsVanillaCompatible,
		Nickname:                p.Nickname,
		CountryCode:             p.CountryCode,
		ResetCounter:            0,
		CarStyle:                p.CarStyle,
		Record:                  nil,
		Ping:                    0,
		PingIdCounter:           0,
		PingPackages:            make([]PingPackage, 0),
		UnsentCarStates:         make([]any, 0),
	}
	server.Players = append(server.Players, &newPlayer)

	newPlayer.Send(gamepackets.EndSessionPacket{})
	newPlayer.SendTrack(*server.GameSession.CurrentTrack)
	newPlayer.StartNewSession(*server.GameSession)
}

func (server *GameServer) onPlayerDisconnect(sessionId string) {

	for index, player := range server.Players {
		if player.Session.SessionID == sessionId {
			log.Println("Removing player " + player.Nickname)
			server.playersLock.Lock()
			server.Players = append(server.Players[:index], server.Players[index+1:]...)
			server.playersLock.Unlock()
		}
	}
}
