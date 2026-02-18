package game

import gametrack "polyserver/game/track"

type GameSession struct {
	SessionID        uint32
	GameMode         GameMode
	SwitchingSession bool
	CurrentTrack     *gametrack.Track
	MaxPlayers       int
}

// TODO: Proper session switching
