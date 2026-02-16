package game

import (
	"fmt"
	gamepackets "polyserver/game/packets"
	gametrack "polyserver/game/track"
	webrtc_session "polyserver/webrtc"
	"time"
)

type Player struct {
	Session                 *webrtc_session.PeerSession
	IsKicked                bool
	ID                      int
	Mods                    []string
	IsModsVanillaCompatible bool
	Nickname                string
	CountryCode             string
	ResetCounter            int
	CarStyle                string
	Record                  *Record
	Ping                    int
	PingIdCounter           int
	PingPackages            []PingPackage
	UnsentCarStates         []any // TODO: CAR STATE STUFF
}

type Record struct {
	numberOfFrames int
}

type PingPackage struct {
	pingId   int
	sentTime time.Time
}

func (player *Player) Send(packet gamepackets.PlayerPacket) error {
	data, err := packet.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal %s packet: %w", packet.Type(), err)
	}

	// Special handling for TrackChunk packets might be needed
	if packet.Type() == gamepackets.TrackChunk {
		return player.Session.ReliableDC.Send(data)
	}

	// For other packets, just send directly
	return player.Session.ReliableDC.Send(data)
}

func (player *Player) SendTrack(track gametrack.Track) error {
	// Send track ID
	trackId, err := track.GetTrackID()
	if err != nil {
		return fmt.Errorf("failed to get track ID: %w", err)
	}

	if err := player.Send(gamepackets.TrackIDPacket{TrackID: trackId}); err != nil {
		return fmt.Errorf("failed to send track ID: %w", err)
	}

	// Get the exported track string (base62 encoded)
	// This should be the same as n.toExportString(t) in JS
	trackString := track.ExportString

	// Send track data in chunks of 16383 bytes
	for offset := 0; offset < len(trackString); offset += 16383 {
		// Calculate chunk length (min of remaining data and chunk size)
		chunkEnd := offset + 16383
		if chunkEnd > len(trackString) {
			chunkEnd = len(trackString)
		}
		chunkLen := chunkEnd - offset

		// Create packet with 1 byte header + chunk data
		packet := make([]byte, 1+chunkLen)
		packet[0] = byte(gamepackets.TrackChunk)

		// Copy string characters directly (they're ASCII/base62)
		copy(packet[1:], trackString[offset:chunkEnd])

		// Send the raw packet
		if err := player.Session.ReliableDC.Send(packet); err != nil {
			return fmt.Errorf("failed to send chunk at offset %d: %w", offset, err)
		}

		// Optional: Add a small delay between chunks if needed
		// time.Sleep(10 * time.Millisecond)
	}

	return nil
}

func (player *Player) StartNewSession(session GameSession) {
	player.Send(gamepackets.NewSessionPacket{
		SessionID: session.SessionID,
		GameMode:  uint8(session.GameMode),
	})
}
