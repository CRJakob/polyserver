package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"polyserver/game"
	gamepackets "polyserver/game/packets"
	"polyserver/metrics"
	"polyserver/signaling"
	"polyserver/tracks"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

func setupLogging() {
	file, err := os.OpenFile(
		"polyserver.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	multi := io.MultiWriter(os.Stdout, file)

	log.SetOutput(multi)

	// Optional: include date + time + file:line
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func runServer() {

	tracksDir := flag.String("tracks", "tracks/official", "track directory")
	controlPort := flag.Int("control-port", 9090, "internal control port")

	flag.Parse()

	log.Println("Game server starting...")

	tracksMap, trackNames := tracks.LoadAllTracks(*tracksDir)
	if len(trackNames) == 0 {
		log.Fatal("No tracks found")
	}

	defaultTrack := tracksMap[trackNames[0]]

	server := signaling.NewServer()

	if err := server.Connect(); err != nil {
		log.Fatal(err)
	}
	go server.Start()

	gameServer := game.NewServer(server)

	gameServer.UpdateGameSession(game.GameSession{
		SessionID:        0,
		GameMode:         game.Competitive,
		SwitchingSession: false,
		CurrentTrack:     defaultTrack,
		MaxPlayers:       200,
	})

	if err := server.CreateInvite(); err != nil {
		log.Fatalf("Failed to create invite: %v", err)
	}

	log.Println("Initial invite:", server.CurrentInvite)

	// ---- CONTROL API ----

	app := fiber.New()

	app.Get("/status", func(c *fiber.Ctx) error {

		currentName := ""
		currentSession, err := json.Marshal(game.GameSession{
			SessionID:        gameServer.GameSession.SessionID,
			GameMode:         gameServer.GameSession.GameMode,
			SwitchingSession: gameServer.GameSession.SwitchingSession,
			MaxPlayers:       gameServer.GameSession.MaxPlayers,
		})
		if err != nil {
			log.Println("Error marshalling session: " + err.Error())
		}
		for name, t := range tracksMap {
			if t == gameServer.GameSession.CurrentTrack {
				currentName = name
				break
			}
		}

		return c.JSON(fiber.Map{
			"invite":  server.CurrentInvite,
			"tracks":  trackNames,
			"current": currentName,
			"session": string(currentSession),
		})
	})

	app.Post("/invite", func(c *fiber.Ctx) error {

		if err := server.CreateInvite(); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(fiber.Map{
			"invite": server.CurrentInvite,
		})
	})

	app.Post("/track", func(c *fiber.Ctx) error {

		type Req struct {
			Name string `json:"name"`
		}

		var req Req
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).SendString("Invalid body")
		}

		t, ok := tracksMap[req.Name]
		if !ok {
			return c.Status(404).SendString("Track not found")
		}

		cur := gameServer.GameSession
		cur.CurrentTrack = t

		log.Println("Track switched to", req.Name)

		return c.SendStatus(204)
	})

	app.Post("/kick", func(c *fiber.Ctx) error {

		type Req struct {
			ID uint32 `json:"id"`
		}

		var req Req
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).SendString("Invalid body")
		}

		for _, player := range gameServer.Players {
			if player.ID == req.ID {
				log.Println("Kicked player: ", player.Nickname)
				player.Send(gamepackets.KickPlayerPacket{})
				for _, p := range gameServer.Players {
					p.Send(gamepackets.RemovePlayerPacket{
						ID:       player.ID,
						IsKicked: true,
					})
				}
				time.AfterFunc(1*time.Second, func() {
					player.Session.Peer.Close()
				})
				break
			}
		}

		return c.SendStatus(204)
	})

	app.Post("/session/end", func(c *fiber.Ctx) error {
		if gameServer.GameSession.SwitchingSession {
			log.Println("Can't end session, already ended.")
			return c.SendStatus(400)
		}
		log.Println("Ending session...")
		gameServer.GameSession.SwitchingSession = true
		for _, player := range gameServer.Players {
			player.Send(gamepackets.EndSessionPacket{})
		}
		return c.SendStatus(204)
	})

	app.Post("/session/start", func(c *fiber.Ctx) error {
		if !gameServer.GameSession.SwitchingSession {
			log.Println("Can't start session, already started.")
			return c.SendStatus(400)
		}
		log.Println("Starting session...")
		gameServer.GameSession.SwitchingSession = false
		for _, player := range gameServer.Players {
			player.StartNewSession()
		}
		return c.SendStatus(204)
	})

	app.Post("/session/set", func(c *fiber.Ctx) error {

		type Req struct {
			GameMode   game.GameMode `json:"gamemode"`
			Track      string        `json:"track"`
			MaxPlayers int           `json:"maxPlayers"`
		}

		var req Req
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).SendString("Invalid body")
		}
		t, ok := tracksMap[req.Track]

		if !ok {
			log.Println("Track " + req.Track + " not found.")
			return c.SendStatus(400)
		}

		gameServer.UpdateGameSession(game.GameSession{
			GameMode:         req.GameMode,
			SwitchingSession: true,
			CurrentTrack:     t,
			MaxPlayers:       req.MaxPlayers,
		})
		log.Println("Got new session data...")

		return c.SendStatus(204)
	})

	app.Get("/players", func(c *fiber.Ctx) error {

		list := []fiber.Map{}
		for _, p := range gameServer.Players {

			timeStr := "-"
			if p.NumberOfFrames != nil {
				seconds := float64(*p.NumberOfFrames) / 1000.0
				timeStr = fmt.Sprintf("%.3fs", seconds)
			}

			list = append(list, fiber.Map{
				"id":   p.ID,
				"name": p.Nickname,
				"time": timeStr,
				"ping": p.Ping,
			})
		}

		return c.JSON(fiber.Map{
			"players": list,
		})
	})

	// Metrics endpoint - returns bandwidth and connection stats
	app.Get("/metrics", func(c *fiber.Ctx) error {
		snap := metrics.GetMetrics().Snapshot()
		return c.JSON(fiber.Map{
			"uptime_seconds":        snap.Uptime,
			"active_connections":    snap.ActiveConnections,
			"total_connections":     snap.TotalConnections,
			"bytes_sent":            snap.BytesSent,
			"bytes_received":        snap.BytesReceived,
			"mbps_sent":             snap.MbpsSent,
			"mbps_received":         snap.MbpsReceived,
			"packets_sent":          snap.PacketsSent,
			"packets_received":      snap.PacketsReceived,
			"avg_packet_size_out":   snap.AveragePacketSizeOut,
			"avg_packet_size_in":    snap.AveragePacketSizeIn,
			"players_joined":        snap.PlayersJoined,
			"players_left":          snap.PlayersLeft,
			"car_updates":           snap.CarUpdatesProcessed,
			"car_update_failures":   snap.CarUpdatesFailed,
			"car_failure_rate":      snap.CarUpdateFailureRate,
			"pings_processed":       snap.PingsProcessed,
			"avg_latency_ms":        snap.AverageLatencyMs,
			"peak_latency_ms":       snap.PeakLatencyMs,
		})
	})

	// Metrics reset endpoint (useful for benchmarking between tests)
	app.Post("/metrics/reset", func(c *fiber.Ctx) error {
		metrics.GetMetrics().Reset()
		return c.SendStatus(204)
	})

	addr := "127.0.0.1:" + strconv.Itoa(*controlPort)

	go func() {
		log.Println("Control API running on", addr)
		if err := app.Listen(addr); err != nil {
			log.Println(err)
		}
	}()

	// Start pprof on a different port for profiling
	go func() {
		pprofAddr := "127.0.0.1:6060"
		log.Println("pprof profiling available at http://" + pprofAddr + "/debug/pprof/")
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Println("pprof error:", err)
		}
	}()

	select {} // keep server alive
}
