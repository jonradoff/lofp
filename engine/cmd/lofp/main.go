package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"crypto/rand"
	"encoding/hex"

	"github.com/jonradoff/lofp/internal/api"
	"github.com/jonradoff/lofp/internal/auth"
	"github.com/jonradoff/lofp/internal/capture"
	"github.com/jonradoff/lofp/internal/config"
	"github.com/jonradoff/lofp/internal/email"
	"github.com/jonradoff/lofp/internal/engine"
	"github.com/jonradoff/lofp/internal/feedback"
	"github.com/jonradoff/lofp/internal/gamelog"
	"github.com/jonradoff/lofp/internal/gameworld"
	"github.com/jonradoff/lofp/internal/hub"
	"github.com/jonradoff/lofp/internal/scriptparser"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	configPath := "config/dev.yaml"
	if p := os.Getenv("LOFP_CONFIG"); p != "" {
		configPath = p
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Parse game scripts
	log.Println("Parsing game scripts...")
	start := time.Now()
	result, err := scriptparser.ParseConfig(cfg.Game.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to parse scripts: %v", err)
	}
	log.Printf("Parsed %d rooms, %d items, %d monsters, %d nouns, %d adjectives in %v",
		len(result.Rooms), len(result.Items), len(result.Monsters),
		len(result.Nouns), len(result.Adjectives), time.Since(start))

	// Convert to ParsedData
	parsed := &gameworld.ParsedData{
		Rooms:        result.Rooms,
		Items:        result.Items,
		Monsters:     result.Monsters,
		Nouns:        result.Nouns,
		Adjectives:   result.Adjectives,
		MonsterAdjs:  result.MonsterAdjs,
		Variables:    result.Variables,
		Regions:      result.Regions,
		MonsterLists:         result.MonsterLists,
		SeasonalMonsterLists: result.SeasonalMonsterLists,
		SeasonalRooms:        result.SeasonalRooms,
		CEvents:      result.CEvents,
		MoneyDefs:    result.MoneyDefs,
		ForageDefs:   result.ForageDefs,
		MineDefs:     result.MineDefs,
		StartRoom:    result.StartRoom,
		BumpRoom:     result.BumpRoom,
	}

	// Connect to MongoDB (optional — game works without it)
	var db *mongo.Database
	if cfg.MongoDB.URI != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoDB.URI))
		if err != nil {
			log.Printf("Warning: MongoDB connection failed: %v (continuing without persistence)", err)
		} else {
			if err := client.Ping(ctx, nil); err != nil {
				log.Printf("Warning: MongoDB ping failed: %v (continuing without persistence)", err)
			} else {
				db = client.Database(cfg.MongoDB.Database)
				log.Printf("Connected to MongoDB: %s", cfg.MongoDB.Database)
			}
		}
	}

	// Generate unique machine ID
	machineBytes := make([]byte, 8)
	rand.Read(machineBytes)
	machineID := hex.EncodeToString(machineBytes)

	// Wire up script parser callback for hot-reload (avoids circular import)
	engine.ScriptParser = func(content, filename string) (*gameworld.ParsedData, error) {
		r, err := scriptparser.ParseScriptContent(content, filename)
		if err != nil {
			return nil, err
		}
		return &gameworld.ParsedData{
			Rooms:       r.Rooms,
			Items:       r.Items,
			Monsters:    r.Monsters,
			Nouns:       r.Nouns,
			Adjectives:  r.Adjectives,
			MonsterAdjs: r.MonsterAdjs,
			Variables:   r.Variables,
			CEvents:     r.CEvents,
			ForageDefs:  r.ForageDefs,
		}, nil
	}

	// Create game engine
	ge := engine.NewGameEngine(db, parsed)

	// Create game logger
	gl := gamelog.New(db)

	// Create cross-machine hub
	h := hub.New(db, machineID)

	// Create auth service (needed for email/password login, Google OAuth, JWT validation)
	var authSvc *auth.Service
	if cfg.Auth.JWTSecret != "" {
		authSvc = auth.NewService(db, cfg.Auth.GoogleClientID, cfg.Auth.JWTSecret)
		if cfg.Auth.GoogleClientID != "" {
			log.Printf("Google OAuth enabled (client ID: %s...)", cfg.Auth.GoogleClientID[:min(20, len(cfg.Auth.GoogleClientID))])
		}
		log.Println("Auth service enabled (email/password + JWT)")
	} else {
		log.Println("Auth service not configured (set JWT_SECRET to enable)")
	}

	// Create email service
	emailSvc := email.New(cfg.Email.ResendAPIKey, cfg.Email.FromAddress, cfg.Server.FrontendURL)
	if emailSvc.Enabled() {
		log.Println("Email service enabled (Resend)")
	} else {
		log.Println("Email service not configured (set RESEND_API_KEY to enable)")
	}

	// Create capture store
	cs := capture.NewStore(db)

	// Create feedback client (VibeCtl integration)
	fb := feedback.New(cfg.Feedback.VibectlURL, cfg.Feedback.VibectlAPIKey)
	if fb.Enabled() {
		log.Println("Feedback pipeline enabled (VibeCtl)")
	}

	// Create API server
	srv := api.NewServer(ge, parsed, authSvc, emailSvc, gl, h, cs, fb, cfg.Server.FrontendURL)
	ge.LoadBanner()
	ge.LoadGMScripts()
	h.Start()
	ge.StartTimeCycle()
	ge.StartWeatherCycle()
	ge.StartRegenCycle()
	ge.StartCEventLoop()
	ge.StartMonsterLoop()

	// Start telnet server (for MUD clients)
	if cfg.Server.TelnetPort > 0 {
		telnetAddr := fmt.Sprintf(":%d", cfg.Server.TelnetPort)
		go srv.ListenTelnet(telnetAddr)
		log.Printf("Telnet server starting on %s", telnetAddr)
	}

	// Start TLS telnet server
	if cfg.Server.TelnetTLSPort > 0 && cfg.Server.TelnetTLSCert != "" && cfg.Server.TelnetTLSKey != "" {
		tlsAddr := fmt.Sprintf(":%d", cfg.Server.TelnetTLSPort)
		go srv.ListenTelnetTLS(tlsAddr, cfg.Server.TelnetTLSCert, cfg.Server.TelnetTLSKey)
		log.Printf("Telnet TLS server starting on %s", tlsAddr)
	}

	// Start SSH server (for MUD clients with SSH)
	if cfg.Server.SSHPort > 0 {
		sshAddr := fmt.Sprintf(":%d", cfg.Server.SSHPort)
		go srv.ListenSSH(sshAddr)
		log.Printf("SSH server starting on %s", sshAddr)
	}

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Legends of Future Past server starting on %s", addr)
	log.Printf("Frontend URL: %s", cfg.Server.FrontendURL)
	if err := http.ListenAndServe(addr, srv.Router()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
