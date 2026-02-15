package signaling

import "github.com/pion/webrtc/v4"

type WebsocketResponse struct {
	Type string `json:"type"`
}

type CreateInviteResponse struct {
	Type                string `json:"type"`
	InviteCode          string `json:"inviteCode"`
	TimeoutMilliseconds int    `json:"timeoutMilliseconds"`
	CensoredNickname    string `json:"censoredNickname"`
}

type JoinInvite struct {
	Type                    string   `json:"type"`
	Session                 string   `json:"session"`
	Offer                   string   `json:"offer"`
	Nickname                string   `json:"nickname"`
	Mods                    []string `json:"mods"`
	IsModsVanillaCompatible bool     `json:"isModsVanillaCompatible"`
	CountryCode             string   `json:"countryCode"`
	CarStyle                string   `json:"carStyle"`
}

type AcceptJoinPacket struct {
	Type                    string   `json:"type"`
	Version                 string   `json:"version"`
	Session                 string   `json:"session"`
	IsModsVanillaCompatible bool     `json:"isModsVanillaCompatible"`
	Mods                    []string `json:"mods"`
	CliendId                int      `json:"clientId"`
	Answer                  string   `json:"answer"`
}

type IceServerResponse struct {
	Urls string
}

type IceCandidate struct {
	Candidate        string  `json:"candidate"`
	SDPMid           *string `json:"sdpMid"`
	SDPMLineIndex    *uint16 `json:"sdpMLineIndex"`
	UsernameFragment *string `json:"usernameFragment"`
}

type IceCandidatePacket struct {
	Type      string       `json:"type"`
	Candidate IceCandidate `json:"candidate"`
	Version   string       `json:"version"`
	Session   string       `json:"session"`
}

type IceCandidateResponse struct {
	Type      string                  `json:"type"`
	Session   string                  `json:"session"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}
