package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"polyserver/game"
	"polyserver/signaling"
	"polyserver/tracks"
	"strconv"

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

		cur := *gameServer.GameSession
		cur.CurrentTrack = t

		gameServer.UpdateGameSession(cur)

		log.Println("Track switched to", req.Name)

		return c.SendStatus(204)
	})

	app.Get("/players", func(c *fiber.Ctx) error {

		list := []fiber.Map{}
		for _, p := range gameServer.Players {

			timeStr := "-"
			if p.NumberOfFrames != nil {
				// 60 FPS → seconds
				seconds := float64(*p.NumberOfFrames) / 60.0
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

	addr := "127.0.0.1:" + strconv.Itoa(*controlPort)

	go func() {
		log.Println("Control API running on", addr)
		if err := app.Listen(addr); err != nil {
			log.Println(err)
		}
	}()

	select {} // keep server alive
}
