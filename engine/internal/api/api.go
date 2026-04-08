package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jonradoff/lofp/internal/auth"
	"github.com/jonradoff/lofp/internal/capture"
	"github.com/jonradoff/lofp/internal/email"
	"github.com/jonradoff/lofp/internal/engine"
	"github.com/jonradoff/lofp/internal/gamelog"
	"github.com/jonradoff/lofp/internal/gameworld"
	"github.com/jonradoff/lofp/internal/hub"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Server struct {
	engine      *engine.GameEngine
	parsed      *gameworld.ParsedData
	auth        *auth.Service
	email       *email.Service
	gamelog     *gamelog.Logger
	hub         *hub.Hub
	captures    *capture.Store
	router      *mux.Router
	upgrader    websocket.Upgrader
	sessions    map[string]*Session
	mu          sync.RWMutex
	frontendURL string
	connsByIP   map[string]int // per-IP WebSocket connection count
	connMu      sync.Mutex
	rateLimits  map[string][]time.Time // per-IP rate limiting for auth endpoints
	rateMu      sync.Mutex
}

// ClientConn abstracts the transport layer (WebSocket or Telnet).
type ClientConn interface {
	SendResult(result *engine.CommandResult) error
	SendBroadcast(messages []string) error
	SendTypedMessage(msgType string, payload interface{}) error // WS-specific; telnet may no-op
	Close() error
	RemoteAddr() string
}

type Session struct {
	Player       *engine.Player
	Conn         ClientConn
	CaptureID    string    // active capture session ID, empty if not recording
	lastCmdTime  time.Time // rate limiting: last command timestamp
	cmdCount     int       // rate limiting: commands in current window
	chatTimes    []time.Time // chat flood: timestamps of recent broadcasts
	cmdTimes     []time.Time // command rate: sliding window for 10/10s limit
	authFailures int        // auth attempt failures (disconnect after 3)
	lastActivity time.Time  // idle timeout tracking
	quitSent     bool       // QUIT already broadcast departure
}

// wsConn wraps a gorilla WebSocket connection to implement ClientConn.
type wsConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *wsConn) SendResult(result *engine.CommandResult) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	data, _ := json.Marshal(result)
	return w.conn.WriteJSON(WSMessage{Type: "result", Data: data})
}

func (w *wsConn) SendBroadcast(messages []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	data, _ := json.Marshal(map[string]interface{}{"messages": messages})
	return w.conn.WriteJSON(WSMessage{Type: "broadcast", Data: data})
}

func (w *wsConn) SendTypedMessage(msgType string, payload interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	data, _ := json.Marshal(payload)
	return w.conn.WriteJSON(WSMessage{Type: msgType, Data: data})
}

func (w *wsConn) Close() error {
	return w.conn.Close()
}

func (w *wsConn) RemoteAddr() string {
	return w.conn.RemoteAddr().String()
}


// getClientIP extracts the real client IP from the request, preferring
// Fly-Client-IP (set by Fly.io proxy), then X-Forwarded-For, then RemoteAddr.
func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("Fly-Client-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// checkRateLimit enforces per-IP rate limiting. Returns true if the request is allowed.
// maxAttempts is the number of allowed requests within the given window duration.
func (s *Server) checkRateLimit(ip, endpoint string, maxAttempts int, window time.Duration) bool {
	key := endpoint + ":" + ip
	now := time.Now()
	cutoff := now.Add(-window)

	s.rateMu.Lock()
	defer s.rateMu.Unlock()

	// Prune old entries
	timestamps := s.rateLimits[key]
	valid := timestamps[:0]
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= maxAttempts {
		s.rateLimits[key] = valid
		return false
	}
	s.rateLimits[key] = append(valid, now)
	return true
}

func NewServer(ge *engine.GameEngine, parsed *gameworld.ParsedData, authSvc *auth.Service, emailSvc *email.Service, gl *gamelog.Logger, h *hub.Hub, cs *capture.Store, frontendURL string) *Server {
	s := &Server{
		engine:      ge,
		parsed:      parsed,
		auth:        authSvc,
		email:       emailSvc,
		gamelog:     gl,
		hub:         h,
		captures:    cs,
		sessions:    make(map[string]*Session),
		frontendURL: frontendURL,
		connsByIP:   make(map[string]int),
		rateLimits:  make(map[string][]time.Time),
	}
	s.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // Allow non-browser clients
			}
			// Parse both URLs and compare scheme+host exactly
			originURL, err := url.Parse(origin)
			if err != nil {
				return false
			}
			allowed := strings.TrimRight(s.frontendURL, "/")
			allowedURL, err := url.Parse(allowed)
			if err != nil {
				return false
			}
			return originURL.Scheme == allowedURL.Scheme && originURL.Host == allowedURL.Host
		},
	}
	s.router = mux.NewRouter()
	s.setupRoutes()
	ge.SetSessionProvider(s)

	// Set up cross-machine coordination
	h.SetDeliveryFunc(s.deliverRemoteEvent)
	ge.SetRoomChangeCallback(func(change engine.RoomChange) {
		s.hub.Publish(&hub.Event{
			Type: "room_state_change",
			Data: bson.M{
				"roomNumber": change.RoomNumber,
				"type":       change.Type,
				"itemRef":    change.ItemRef,
				"newState":   change.NewState,
				"item":       change.Item,
			},
		})
	})

	// Set up room broadcast for background tasks (monsters, CEVENTs)
	ge.SetRoomBroadcast(func(roomNumber int, messages []string) {
		s.broadcastToRoom(roomNumber, "", messages)
	})

	// Local-only broadcast for monster activity (no hub, this machine only)
	ge.SetLocalRoomBroadcast(func(roomNumber int, messages []string) {
		s.mu.RLock()
		for _, sess := range s.sessions {
			if sess.Player == nil || sess.Player.RoomNumber != roomNumber {
				continue
			}
			// BattleBrief suppresses monster ambient text
			if sess.Player.BattleBrief {
				continue
			}
			filtered := filterBroadcastForPlayer(sess.Player, messages)
			if len(filtered) > 0 {
				s.sendBroadcast(sess, filtered)
			}
		}
		s.mu.RUnlock()
	})

	// Set up player-targeted messages for background tasks (combat, etc.)
	ge.SetSendToPlayer(func(playerName string, messages []string) {
		s.sendToPlayer(playerName, messages)
	})

	return s
}

// OnlinePlayers returns all currently connected players (implements engine.SessionProvider).
func (s *Server) OnlinePlayers() []*engine.Player {
	// Start with local players
	s.mu.RLock()
	localNames := make(map[string]bool, len(s.sessions))
	players := make([]*engine.Player, 0, len(s.sessions))
	for _, sess := range s.sessions {
		players = append(players, sess.Player)
		localNames[sess.Player.FirstName] = true
	}
	s.mu.RUnlock()

	// Add remote players from hub
	for _, rp := range s.hub.AllOnlinePlayers() {
		if localNames[rp.FirstName] {
			continue // already have this player locally
		}
		// Create a lightweight Player for remote presence
		lastName := ""
		if parts := strings.SplitN(rp.FullName, " ", 2); len(parts) > 1 {
			lastName = parts[1]
		}
		players = append(players, &engine.Player{
			FirstName:  rp.FirstName,
			LastName:   lastName,
			RoomNumber: rp.RoomNumber,
			Race:       rp.Race,
			Position:   rp.Position,
			IsGM:       rp.IsGM,
			GMHat:      rp.GMHat,
			GMHidden:   rp.GMHidden,
			GMInvis:    rp.GMInvis,
			Hidden:     rp.Hidden,
		})
	}
	return players
}

func (s *Server) Router() *mux.Router {
	return s.router
}

func (s *Server) setupRoutes() {
	s.router.Use(s.corsMiddleware)

	// Health check (vibectl compatible)
	s.router.HandleFunc("/healthz", s.handleHealthz).Methods("GET", "HEAD")

	// Game WebSocket
	s.router.HandleFunc("/ws/game", s.handleGameWS)
	s.router.HandleFunc("/ws/events", s.handleEventsWS)

	// REST endpoints
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Auth (public)
	api.HandleFunc("/auth/google", s.handleGoogleAuth).Methods("POST")
	api.HandleFunc("/auth/register", s.handleRegister).Methods("POST")
	api.HandleFunc("/auth/login", s.handlePasswordLogin).Methods("POST")
	api.HandleFunc("/auth/verify-email", s.handleVerifyEmail).Methods("POST")
	api.HandleFunc("/auth/forgot-password", s.handleForgotPassword).Methods("POST")
	api.HandleFunc("/auth/reset-password", s.handleResetPassword).Methods("POST")
	api.HandleFunc("/auth/resend-verification", s.handleResendVerification).Methods("POST")
	api.HandleFunc("/auth/verify-code", s.handleVerifyCode).Methods("POST")
	api.HandleFunc("/auth/me", s.handleAuthMe).Methods("GET")
	api.HandleFunc("/auth/me/name", s.handleUpdateName).Methods("PUT")
	api.HandleFunc("/auth/me/password", s.handleUpdatePassword).Methods("PUT")

	// Characters (authenticated)
	api.HandleFunc("/characters", s.handleListCharacters).Methods("GET")
	api.HandleFunc("/characters", s.handleCreateCharacter).Methods("POST")
	api.HandleFunc("/characters/{firstName}", s.handleDeleteCharacter).Methods("DELETE")
	api.HandleFunc("/characters/{firstName}/gm", s.handleToggleGM).Methods("PUT")
	api.HandleFunc("/characters/{firstName}/apikey", s.handleGenerateAPIKey).Methods("POST")
	api.HandleFunc("/characters/{firstName}/apikey", s.handleRevokeAPIKey).Methods("DELETE")

	// Session captures (authenticated)
	api.HandleFunc("/captures", s.handleListCaptures).Methods("GET")
	api.HandleFunc("/captures/{id}", s.handleGetCapture).Methods("GET")
	api.HandleFunc("/captures/{id}/text", s.handleGetCaptureText).Methods("GET")
	api.HandleFunc("/captures/{id}", s.handleDeleteCapture).Methods("DELETE")

	// Game world data (admin only)
	api.HandleFunc("/stats", s.handleStats).Methods("GET")
	api.HandleFunc("/rooms", s.handleListRooms).Methods("GET")
	api.HandleFunc("/rooms/{number}", s.handleGetRoom).Methods("GET")
	api.HandleFunc("/items", s.handleListItems).Methods("GET")
	api.HandleFunc("/items/{number}", s.handleGetItem).Methods("GET")
	api.HandleFunc("/monsters", s.handleListMonsters).Methods("GET")
	api.HandleFunc("/monsters/{number}", s.handleGetMonster).Methods("GET")
	api.HandleFunc("/nouns", s.handleListNouns).Methods("GET")
	api.HandleFunc("/adjectives", s.handleListAdjectives).Methods("GET")

	// Admin bootstrap (only works if no admins exist yet)
	api.HandleFunc("/admin/bootstrap", s.handleBootstrapAdmin).Methods("POST")

	// Admin endpoints (require admin)
	api.HandleFunc("/admin/accounts", s.handleListAccounts).Methods("GET")
	api.HandleFunc("/admin/accounts/{id}", s.handleGetAccount).Methods("GET")
	api.HandleFunc("/admin/accounts/{id}/admin", s.handleToggleAdmin).Methods("PUT")
	api.HandleFunc("/admin/characters", s.handleAdminListCharacters).Methods("GET")
	api.HandleFunc("/admin/characters/{firstName}", s.handleAdminGetCharacter).Methods("GET")
	api.HandleFunc("/admin/characters/{firstName}/reassign", s.handleAdminReassignCharacter).Methods("PUT")
	api.HandleFunc("/admin/logs", s.handleAdminLogs).Methods("GET")
	api.HandleFunc("/admin/characters/deleted", s.handleAdminListDeletedCharacters).Methods("GET")
	api.HandleFunc("/admin/characters/{firstName}/recover", s.handleAdminRecoverCharacter).Methods("PUT")

	// Serve static frontend files in production (if /app/static exists)
	staticDir := os.Getenv("LOFP_STATIC_DIR")
	if staticDir == "" {
		staticDir = "/app/static"
	}
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		spa := spaHandler{staticDir: staticDir}
		s.router.PathPrefix("/").Handler(spa)
		log.Printf("Serving static frontend from %s", staticDir)
	}
}

// spaHandler serves static files, falling back to index.html for client-side routing.
type spaHandler struct {
	staticDir string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := h.staticDir + r.URL.Path
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		http.ServeFile(w, r, h.staticDir+"/index.html")
		return
	}
	http.ServeFile(w, r, path)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.frontendURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WebSocket message types
type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type CommandMsg struct {
	Input string `json:"input"`
}

type CreateCharMsg struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Race      int    `json:"race"`
	Gender    int    `json:"gender"`
}

type AuthMsg struct {
	Token string `json:"token"`
}

func (s *Server) handleEventsWS(w http.ResponseWriter, r *http.Request) {
	// Validate admin auth via query param token
	token := r.URL.Query().Get("token")
	if token == "" || s.auth == nil {
		http.Error(w, "unauthorized", 401)
		return
	}
	accountID, err := s.auth.ValidateJWT(token)
	if err != nil || accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	account, err := s.auth.GetAccount(r.Context(), accountID)
	if err != nil || account == nil || !account.IsAdmin {
		http.Error(w, "forbidden", 403)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Subscribe to engine events
	ch := s.engine.Events.Subscribe()
	defer s.engine.Events.Unsubscribe(ch)

	// Send initial status
	conn.WriteJSON(map[string]interface{}{
		"type": "event",
		"data": engine.EngineEvent{
			Time:     time.Now(),
			Category: "system",
			Message:  fmt.Sprintf("Event monitor connected. Game time: Hour %d, Day %d of %s, Year %d", engine.GameHour(), engine.GameDay(), engine.GameMonthName(), engine.GameYear()),
		},
	})

	// Read pump (just to detect disconnect)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Write pump
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteJSON(map[string]interface{}{
				"type": "event",
				"data": event,
			}); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

func (s *Server) handleGameWS(w http.ResponseWriter, r *http.Request) {
	// Per-IP connection limiting
	ip := getClientIP(r)
	s.connMu.Lock()
	if s.connsByIP[ip] >= 5 { // max 5 connections per IP
		s.connMu.Unlock()
		http.Error(w, "too many connections", 429)
		return
	}
	s.connsByIP[ip]++
	s.connMu.Unlock()
	defer func() {
		s.connMu.Lock()
		s.connsByIP[ip]--
		if s.connsByIP[ip] <= 0 { delete(s.connsByIP, ip) }
		s.connMu.Unlock()
	}()

	// Total connection cap
	s.mu.RLock()
	totalConns := len(s.sessions)
	s.mu.RUnlock()
	if totalConns >= 500 {
		http.Error(w, "server full", 503)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}
	defer conn.Close()
	conn.SetReadLimit(65536)

	session := &Session{Conn: &wsConn{conn: conn}}
	var accountID, authName, authEmail string

	// Send welcome
	s.sendResult(session, &engine.CommandResult{
		Messages: []string{
			"",
			"====================================",
			" LEGENDS OF FUTURE PAST",
			" The Shattered Realms Await...",
			"====================================",
			"",
		},
	})

	session.lastActivity = time.Now()
	for {
		// Idle timeout: 30 minutes with no messages → disconnect
		conn.SetReadDeadline(time.Now().Add(30 * time.Minute))
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WS read error: %v", err)
			}
			break
		}

		switch msg.Type {
		case "auth":
			var authMsg AuthMsg
			json.Unmarshal(msg.Data, &authMsg)
			if s.auth != nil && authMsg.Token != "" {
				claims, err := s.auth.ValidateJWTFull(authMsg.Token)
				if err != nil {
					session.Conn.SendTypedMessage("auth_result", map[string]interface{}{
						"success": false,
						"error":   "invalid token",
					})
					continue
				}
				accountID = claims.AccountID
				authName = claims.Name
				authEmail = claims.Email
				s.gamelog.Log(gamelog.EventLogin, claims.Name, claims.AccountID, claims.Email, 0, "")
				session.Conn.SendTypedMessage("auth_result", map[string]interface{}{
					"success": true,
				})
			} else {
				session.Conn.SendTypedMessage("auth_result", map[string]interface{}{
					"success": false,
					"error":   "auth not configured",
				})
			}

		case "auth_apikey":
			var keyMsg struct {
				Key string `json:"key"`
			}
			json.Unmarshal(msg.Data, &keyMsg)
			ctx := context.Background()
			player, err := s.engine.ValidateAPIKey(ctx, keyMsg.Key)
			if err != nil {
				session.authFailures++
				session.Conn.SendTypedMessage("auth_result", map[string]interface{}{
					"success": false,
					"error":   "invalid API key",
				})
				if session.authFailures >= 3 {
					session.Conn.SendTypedMessage("error", map[string]interface{}{"message": "Too many failed auth attempts."})
					return // disconnect
				}
				continue
			}
			session.Player = player
			s.mu.Lock()
			s.sessions[player.FirstName] = session
			s.mu.Unlock()
			s.hub.RegisterPlayer(player.FirstName, player.FullName(), player.RoomNumber,
				player.Race, player.RaceName(), player.Position,
				player.IsGM, player.GMHat, player.GMHidden, player.GMInvis, player.Hidden)
			s.gamelog.Log(gamelog.EventGameEnter, player.FullName(), player.AccountID,
				"bot login via API key", player.RoomNumber, "")
			s.broadcastGlobal(player.FirstName,
				[]string{fmt.Sprintf("** %s has just entered the Realms.", player.FirstName)})
			s.broadcastToRoom(player.RoomNumber, player.FirstName, []string{fmt.Sprintf("%s arrives.", player.FirstName)})
			result := s.engine.EnterRoom(ctx, player)
			s.sendResult(session, result)
			session.Conn.SendTypedMessage("auth_result", map[string]interface{}{
				"success":   true,
				"character": player.FirstName,
			})
			continue

		case "create_character":
			var create CreateCharMsg
			json.Unmarshal(msg.Data, &create)

			ctx := context.Background()

			// Block unverified accounts from creating/entering characters
			if accountID != "" && s.auth != nil {
				acct, err := s.auth.GetAccount(ctx, accountID)
				if err == nil && !acct.EmailVerified && acct.GoogleID == "" {
					session.Conn.SendTypedMessage("error", map[string]interface{}{
						"message":    "Please verify your email address before playing.",
						"needVerify": true,
					})
					continue
				}
			}

			// Try to load existing character first (no validation needed for existing chars)
			player, err := s.engine.LoadPlayer(ctx, create.FirstName, create.LastName)
			if err != nil || player == nil {
				// New character — validate name and limits
				if err := engine.ValidateCharacterInput(create.FirstName, create.LastName, create.Race, create.Gender); err != nil {
					session.Conn.SendTypedMessage("error", map[string]interface{}{"message": err.Error()})
					continue
				}
				// Max 8 characters per account
				existing, _ := s.engine.ListPlayersByAccount(ctx, accountID)
				if len(existing) >= 8 {
					session.Conn.SendTypedMessage("error", map[string]interface{}{"message": "You can have at most 8 characters per account."})
					continue
				}
				player = s.engine.CreateNewPlayer(ctx, create.FirstName, create.LastName, create.Race, create.Gender, accountID)
				s.gamelog.Log(gamelog.EventCharacterCreate, player.FullName(), accountID,
					fmt.Sprintf("Race: %s, Gender: %d", player.RaceName(), player.Gender),
					player.RoomNumber, "")
				s.sendResult(session, &engine.CommandResult{
					Messages: []string{
						fmt.Sprintf("Welcome to the Shattered Realms, %s the %s!", player.FullName(), player.RaceName()),
						"",
					},
				})
			} else if player.AccountID != accountID {
				session.Conn.SendTypedMessage("error", map[string]interface{}{
					"message": fmt.Sprintf("The name '%s %s' is already taken. Please choose a different name.", create.FirstName, create.LastName),
				})
				continue
			} else {
				s.sendResult(session, &engine.CommandResult{
					Messages: []string{
						fmt.Sprintf("Welcome back, %s the %s!", player.FullName(), player.RaceName()),
						"",
					},
				})
			}
			session.Player = player

			s.mu.Lock()
			s.sessions[player.FirstName] = session
			s.mu.Unlock()

			// Register in cross-machine presence
			s.hub.RegisterPlayer(player.FirstName, player.FullName(), player.RoomNumber,
				player.Race, player.RaceName(), player.Position,
				player.IsGM, player.GMHat, player.GMHidden, player.GMInvis, player.Hidden)

			// Log character entering the game world
			s.gamelog.Log(gamelog.EventGameEnter, player.FullName(), accountID,
				fmt.Sprintf("%s (%s)", authName, authEmail), player.RoomNumber, "")
			s.broadcastGlobal(player.FirstName,
				[]string{fmt.Sprintf("** %s has just entered the Realms.", player.FirstName)})
			s.broadcastToRoom(player.RoomNumber, player.FirstName, []string{fmt.Sprintf("%s arrives.", player.FirstName)})

			result := s.engine.EnterRoom(ctx, player)
			s.sendResult(session, result)
			if len(result.GMBroadcast) > 0 {
				s.broadcastToGMs(result.GMBroadcast)
			}

		case "start_capture":
			if session.Player == nil || accountID == "" {
				log.Printf("capture: cannot start — player=%v accountID=%q", session.Player != nil, accountID)
				continue
			}
			if session.CaptureID != "" {
				log.Printf("capture: already recording %s", session.CaptureID)
				continue
			}
			ctx := context.Background()
			id, err := s.captures.Start(ctx, accountID, session.Player.FullName())
			if err != nil {
				log.Printf("capture: start failed: %v", err)
			} else if id != "" {
				session.CaptureID = id
				log.Printf("capture: started %s for %s", id, session.Player.FullName())
				session.Conn.SendTypedMessage("capture_status", map[string]interface{}{"recording": true, "id": id})
			}
			continue

		case "stop_capture":
			if session.CaptureID != "" {
				ctx := context.Background()
				s.captures.Stop(ctx, session.CaptureID)
				session.Conn.SendTypedMessage("capture_status", map[string]interface{}{"recording": false})
				session.CaptureID = ""
			}
			continue

		case "command":
			if session.Player == nil {
				s.sendResult(session, &engine.CommandResult{
					Messages: []string{"You must create a character first."},
				})
				continue
			}
			// Track activity for idle timeout
			session.lastActivity = time.Now()
			now := time.Now()

			// Rate limit 1: max 4 commands per second (burst)
			if now.Sub(session.lastCmdTime) > time.Second {
				session.cmdCount = 0
				session.lastCmdTime = now
			}
			session.cmdCount++
			if session.cmdCount > 4 {
				s.sendResult(session, &engine.CommandResult{
					Messages: []string{"[Slow down! Too many commands.]"},
				})
				continue
			}
			// Rate limit 2: max 10 commands per 10 seconds (sustained)
			cutoff := now.Add(-10 * time.Second)
			var recentCmds []time.Time
			for _, t := range session.cmdTimes {
				if t.After(cutoff) { recentCmds = append(recentCmds, t) }
			}
			session.cmdTimes = append(recentCmds, now)
			if len(session.cmdTimes) > 10 {
				s.sendResult(session, &engine.CommandResult{
					Messages: []string{"[Slow down! Too many commands.]"},
				})
				continue
			}
			var cmd CommandMsg
			json.Unmarshal(msg.Data, &cmd)
			ctx := context.Background()
			playerRoom := session.Player.RoomNumber
			result := s.engine.ProcessCommand(ctx, session.Player, cmd.Input)
			result.PlayerState = session.Player
			result.PromptIndicators = session.Player.PromptIndicators()
			s.sendResult(session, result)

			// Capture input and output
			if session.CaptureID != "" {
				var lines []capture.Line
				lines = append(lines, capture.Line{Time: time.Now(), Type: "input", Text: cmd.Input})
				for _, m := range result.Messages {
					lines = append(lines, capture.Line{Time: time.Now(), Type: "output", Text: m})
				}
				log.Printf("capture: appending %d lines to %s", len(lines), session.CaptureID)
				go func() {
					if err := s.captures.AppendLines(context.Background(), session.CaptureID, lines); err != nil {
						log.Printf("capture: append failed: %v", err)
					}
				}()
			}

			// Chat flood protection: 5 broadcast messages per 10 seconds
			if len(result.RoomBroadcast) > 0 {
				now := time.Now()
				cutoff := now.Add(-10 * time.Second)
				var recent []time.Time
				for _, t := range session.chatTimes {
					if t.After(cutoff) { recent = append(recent, t) }
				}
				session.chatTimes = recent
				if len(session.chatTimes) >= 5 {
					s.sendResult(session, &engine.CommandResult{
						Messages: []string{"[You are sending messages too quickly. Please wait.]"},
					})
					continue
				}
				session.chatTimes = append(session.chatTimes, now)
			}

			s.dispatchCommandResult(session, result)
			_ = playerRoom
		}
	}

	// Cleanup — global departure broadcast (skip if QUIT already sent it)
	if session.Player != nil {
		s.gamelog.Log(gamelog.EventGameExit, session.Player.FullName(), session.Player.AccountID,
			fmt.Sprintf("%s (%s)", authName, authEmail), session.Player.RoomNumber, "")
		if !session.quitSent {
			s.broadcastGlobal(session.Player.FirstName,
				[]string{fmt.Sprintf("** %s has just left the Realms.", session.Player.FirstName)})
		}
		s.broadcastToRoom(session.Player.RoomNumber, session.Player.FirstName,
			[]string{fmt.Sprintf("%s fades from the Realms.", session.Player.FirstName)})
		if session.CaptureID != "" {
			s.captures.Stop(context.Background(), session.CaptureID)
			session.CaptureID = ""
		}
		s.hub.UnregisterPlayer(session.Player.FirstName)
		s.mu.Lock()
		delete(s.sessions, session.Player.FirstName)
		s.mu.Unlock()
	}
	if accountID != "" {
		s.gamelog.Log(gamelog.EventLogout, authName, accountID, authEmail, 0, "")
	}
}

// dispatchCommandResult handles all broadcast routing after a command is processed.
// Shared by both WebSocket and Telnet command loops.
func (s *Server) dispatchCommandResult(session *Session, result *engine.CommandResult) {
	// Sanitize all outgoing messages (strip HTML tags for defense in depth)
	sanitizeMessages(result.Messages)
	sanitizeMessages(result.RoomBroadcast)
	sanitizeMessages(result.OldRoomMsg)
	sanitizeMessages(result.TargetMsg)
	sanitizeMessages(result.GlobalBroadcast)
	sanitizeMessages(result.GMBroadcast)

	// Broadcast to others in the room (excluding emote target if they get a special message)
	if len(result.RoomBroadcast) > 0 {
		if result.TargetName != "" {
			s.broadcastToRoom(session.Player.RoomNumber, session.Player.FirstName, result.RoomBroadcast, result.TargetName)
		} else {
			s.broadcastToRoom(session.Player.RoomNumber, session.Player.FirstName, result.RoomBroadcast)
		}
	}
	// Send second-person message to emote target
	if result.TargetName != "" && len(result.TargetMsg) > 0 {
		s.sendToPlayer(result.TargetName, result.TargetMsg)
	}
	// Broadcast departure to old room (movement)
	if result.OldRoom > 0 && len(result.OldRoomMsg) > 0 {
		s.broadcastToRoom(result.OldRoom, session.Player.FirstName, result.OldRoomMsg)
		s.hub.UpdatePlayerRoom(session.Player.FirstName, session.Player.RoomNumber)
	}
	// Whisper to specific target
	if result.WhisperTarget != "" && result.WhisperMsg != "" {
		s.sendToPlayer(result.WhisperTarget, []string{result.WhisperMsg})
	}
	// Global broadcast (e.g. quit)
	if len(result.GlobalBroadcast) > 0 {
		s.broadcastGlobal(session.Player.FirstName, result.GlobalBroadcast)
		if result.Quit {
			session.quitSent = true
		}
	}
	// GM broadcast
	if len(result.GMBroadcast) > 0 {
		s.broadcastToGMs(result.GMBroadcast)
	}
	// Log event (e.g., REPORT)
	if result.LogEventType != "" {
		s.gamelog.Log(gamelog.EventType(result.LogEventType), session.Player.FullName(), "",
			result.LogEventDetail, session.Player.RoomNumber, "")
	}
	// Telepathy broadcast
	if result.TelepathyMsg != "" {
		telepathyLines := []string{
			fmt.Sprintf("You feel the touch of %s's mind:", result.TelepathySender),
			fmt.Sprintf("\"%s\"", result.TelepathyMsg),
		}
		s.mu.RLock()
		for _, sess := range s.sessions {
			if sess.Player == nil || sess.Player.FirstName == result.TelepathySender {
				continue
			}
			if sess.Player.TelepathyActive {
				s.sendBroadcast(sess, telepathyLines)
			}
		}
		s.mu.RUnlock()
	}
}

func (s *Server) sendResult(session *Session, result *engine.CommandResult) {
	session.Conn.SendResult(result)
}

// filterBroadcastForPlayer filters broadcast messages based on the player's SET preferences.
// Returns the filtered list (may be empty).
func filterBroadcastForPlayer(player *engine.Player, messages []string) []string {
	if player == nil {
		return messages
	}
	filtered := make([]string, 0, len(messages))
	for _, msg := range messages {
		// Global message filters
		if player.SuppressLogon && strings.Contains(msg, "has just entered the Realms.") {
			continue
		}
		if player.SuppressLogoff && strings.Contains(msg, "has just left the Realms.") {
			continue
		}
		if player.SuppressDisconnect && strings.Contains(msg, "has just disconnected.") {
			continue
		}
		// ActionBrief: filter healing/spell/eat/drink messages from others
		if player.ActionBrief {
			if strings.Contains(msg, "looks a little better") ||
				strings.Contains(msg, "looks much better") ||
				strings.Contains(msg, "incants a spell") ||
				strings.Contains(msg, "gestures") ||
				strings.Contains(msg, " drinks ") ||
				strings.Contains(msg, " eats ") {
				continue
			}
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func (s *Server) sendBroadcast(session *Session, messages []string) {
	session.Conn.SendBroadcast(messages)

	// Capture broadcast messages
	if session.CaptureID != "" {
		var lines []capture.Line
		for _, text := range messages {
			lines = append(lines, capture.Line{Time: time.Now(), Type: "broadcast", Text: text})
		}
		go s.captures.AppendLines(context.Background(), session.CaptureID, lines)
	}
}

// broadcastToRoom sends messages to all players in a room except the excluded player.
func (s *Server) broadcastToRoom(roomNumber int, excludeName string, messages []string, excludeNames ...string) {
	allExcludes := append([]string{excludeName}, excludeNames...)

	// Deliver to local connections
	s.mu.RLock()
	for _, sess := range s.sessions {
		if sess.Player == nil || sess.Player.RoomNumber != roomNumber {
			continue
		}
		if isExcluded(sess.Player.FirstName, allExcludes) {
			continue
		}
		filtered := filterBroadcastForPlayer(sess.Player, messages)
		if len(filtered) > 0 {
			s.sendBroadcast(sess, filtered)
		}
	}
	s.mu.RUnlock()

	// Publish to hub for other machines
	s.hub.Publish(&hub.Event{
		Type:           "room_broadcast",
		RoomNumber:     roomNumber,
		ExcludePlayers: allExcludes,
		Messages:       messages,
	})
}

func (s *Server) sendToPlayer(firstName string, messages []string) {
	// Try local first
	s.mu.RLock()
	if sess, ok := s.sessions[firstName]; ok {
		s.sendBroadcast(sess, messages)
	}
	s.mu.RUnlock()

	// Publish to hub for other machines
	s.hub.Publish(&hub.Event{
		Type:         "send_to_player",
		TargetPlayer: firstName,
		Messages:     messages,
	})
}

func (s *Server) broadcastGlobal(excludeName string, messages []string) {
	s.mu.RLock()
	for _, sess := range s.sessions {
		if sess.Player == nil || sess.Player.FirstName == excludeName {
			continue
		}
		filtered := filterBroadcastForPlayer(sess.Player, messages)
		if len(filtered) > 0 {
			s.sendBroadcast(sess, filtered)
		}
	}
	s.mu.RUnlock()

	s.hub.Publish(&hub.Event{
		Type:           "global_broadcast",
		ExcludePlayers: []string{excludeName},
		Messages:       messages,
	})
}

func (s *Server) broadcastToGMs(messages []string) {
	s.mu.RLock()
	for _, sess := range s.sessions {
		if sess.Player == nil || !sess.Player.IsGM {
			continue
		}
		s.sendBroadcast(sess, messages)
	}
	s.mu.RUnlock()

	s.hub.Publish(&hub.Event{
		Type:     "gm_broadcast",
		Messages: messages,
	})
}

// deliverRemoteEvent handles events from other machines, delivering to local connections.
func (s *Server) deliverRemoteEvent(event *hub.Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch event.Type {
	case "room_broadcast":
		for _, sess := range s.sessions {
			if sess.Player == nil || sess.Player.RoomNumber != event.RoomNumber {
				continue
			}
			if isExcluded(sess.Player.FirstName, event.ExcludePlayers) {
				continue
			}
			filtered := filterBroadcastForPlayer(sess.Player, event.Messages)
			if len(filtered) > 0 {
				s.sendBroadcast(sess, filtered)
			}
		}
	case "global_broadcast":
		for _, sess := range s.sessions {
			if sess.Player == nil || isExcluded(sess.Player.FirstName, event.ExcludePlayers) {
				continue
			}
			filtered := filterBroadcastForPlayer(sess.Player, event.Messages)
			if len(filtered) > 0 {
				s.sendBroadcast(sess, filtered)
			}
		}
	case "send_to_player":
		if sess, ok := s.sessions[event.TargetPlayer]; ok {
			s.sendBroadcast(sess, event.Messages)
		}
	case "gm_broadcast":
		for _, sess := range s.sessions {
			if sess.Player == nil || !sess.Player.IsGM {
				continue
			}
			s.sendBroadcast(sess, event.Messages)
		}
	case "room_state_change":
		// Apply remote room state change to local game engine
		if event.Data != nil {
			change := engine.RoomChange{}
			if v, ok := event.Data["roomNumber"].(int32); ok {
				change.RoomNumber = int(v)
			} else if v, ok := event.Data["roomNumber"].(int64); ok {
				change.RoomNumber = int(v)
			}
			if v, ok := event.Data["type"].(string); ok {
				change.Type = v
			}
			if v, ok := event.Data["itemRef"].(int32); ok {
				change.ItemRef = int(v)
			} else if v, ok := event.Data["itemRef"].(int64); ok {
				change.ItemRef = int(v)
			}
			if v, ok := event.Data["newState"].(string); ok {
				change.NewState = v
			}
			if v, ok := event.Data["item"]; ok && v != nil {
				// Decode RoomItem from BSON map
				if itemMap, ok := v.(bson.M); ok {
					data, _ := bson.Marshal(itemMap)
					var ri gameworld.RoomItem
					bson.Unmarshal(data, &ri)
					change.Item = &ri
				}
			}
			s.engine.ApplyRoomChange(change)
		}
	}
}

// sanitizeMessages strips HTML tags from messages (defense in depth against XSS).
func sanitizeMessages(msgs []string) {
	for i, msg := range msgs {
		msgs[i] = strings.ReplaceAll(msg, "<", "&lt;")
		msgs[i] = strings.ReplaceAll(msgs[i], ">", "&gt;")
	}
}

func isExcluded(name string, excludes []string) bool {
	for _, e := range excludes {
		if name == e {
			return true
		}
	}
	return false
}

// REST handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s.mu.RLock()
	sessions := len(s.sessions)
	s.mu.RUnlock()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"sessions": sessions,
		"rooms":    len(s.parsed.Rooms),
		"items":    len(s.parsed.Items),
		"monsters": len(s.parsed.Monsters),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rooms":    len(s.parsed.Rooms),
		"items":    len(s.parsed.Items),
		"monsters": len(s.parsed.Monsters),
		"nouns":    len(s.parsed.Nouns),
		"adjs":     len(s.parsed.Adjectives),
		"sessions": len(s.sessions),
	})
}

func (s *Server) handleListRooms(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	// Return summary list
	type roomSummary struct {
		Number  int    `json:"number"`
		Name    string `json:"name"`
		Terrain string `json:"terrain"`
		Exits   int    `json:"exits"`
		File    string `json:"file"`
	}
	var rooms []roomSummary
	q := strings.ToLower(r.URL.Query().Get("q"))
	for _, room := range s.parsed.Rooms {
		if q != "" && !strings.Contains(strings.ToLower(room.Name), q) {
			continue
		}
		rooms = append(rooms, roomSummary{
			Number: room.Number, Name: room.Name,
			Terrain: room.Terrain, Exits: len(room.Exits), File: room.SourceFile,
		})
	}
	json.NewEncoder(w).Encode(rooms)
}

func (s *Server) resolveItemDef(archetype int) map[string]interface{} {
	for _, item := range s.parsed.Items {
		if item.Number == archetype {
			return map[string]interface{}{
				"name":       s.resolveNoun(item.NameID),
				"type":       item.Type,
				"weight":     item.Weight,
				"volume":     item.Volume,
				"substance":  item.Substance,
				"article":    item.Article,
				"wornSlot":   item.WornSlot,
				"container":  item.Container,
				"flags":      item.Flags,
				"sourceFile": item.SourceFile,
			}
		}
	}
	return nil
}

func (s *Server) resolveRoomName(num int) string {
	for _, room := range s.parsed.Rooms {
		if room.Number == num {
			return room.Name
		}
	}
	return ""
}

func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	vars := mux.Vars(r)
	var num int
	fmt.Sscanf(vars["number"], "%d", &num)
	for _, room := range s.parsed.Rooms {
		if room.Number == num {
			// Enrich items with resolved names and full archetype data
			var items []map[string]interface{}
			for _, ri := range room.Items {
				ei := map[string]interface{}{
					"ref":       ri.Ref,
					"archetype": ri.Archetype,
					"itemDef":   s.resolveItemDef(ri.Archetype),
				}
				if ri.Adj1 != 0 {
					ei["adj1"] = ri.Adj1
					ei["adj1Name"] = s.resolveAdj(ri.Adj1)
				}
				if ri.Adj2 != 0 {
					ei["adj2"] = ri.Adj2
					ei["adj2Name"] = s.resolveAdj(ri.Adj2)
				}
				if ri.Adj3 != 0 {
					ei["adj3"] = ri.Adj3
					ei["adj3Name"] = s.resolveAdj(ri.Adj3)
				}
				if ri.Val1 != 0 {
					ei["val1"] = ri.Val1
				}
				if ri.Val2 != 0 {
					ei["val2"] = ri.Val2
				}
				if ri.Val3 != 0 {
					ei["val3"] = ri.Val3
				}
				if ri.Val4 != 0 {
					ei["val4"] = ri.Val4
				}
				if ri.Val5 != 0 {
					ei["val5"] = ri.Val5
				}
				if ri.State != "" {
					ei["state"] = ri.State
				}
				if ri.Extend != "" {
					ei["extend"] = ri.Extend
				}
				if ri.IsPut {
					ei["isPut"] = true
					ei["putIn"] = ri.PutIn
				}
				items = append(items, ei)
			}

			// Enrich exits with room names
			type enrichedExit struct {
				Room     int    `json:"room"`
				RoomName string `json:"roomName"`
			}
			exits := make(map[string]enrichedExit)
			for dir, destNum := range room.Exits {
				exits[dir] = enrichedExit{
					Room:     destNum,
					RoomName: s.resolveRoomName(destNum),
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"number":       room.Number,
				"name":         room.Name,
				"description":  room.Description,
				"exits":        exits,
				"items":        items,
				"terrain":      room.Terrain,
				"lighting":     room.Lighting,
				"monsterGroup": room.MonsterGroup,
				"modifiers":    room.Modifiers,
				"scripts":      room.Scripts,
				"sourceFile":   room.SourceFile,
			})
			return
		}
	}
	http.Error(w, "room not found", 404)
}

func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	type itemSummary struct {
		Number     int    `json:"number"`
		Name       string `json:"name"`
		Type       string `json:"type"`
		Weight     int    `json:"weight"`
		Substance  string `json:"substance"`
		SourceFile string `json:"sourceFile"`
	}
	var items []itemSummary
	for _, item := range s.parsed.Items {
		name := fmt.Sprintf("noun#%d", item.NameID)
		for _, n := range s.parsed.Nouns {
			if n.ID == item.NameID {
				name = n.Name
				break
			}
		}
		items = append(items, itemSummary{
			Number: item.Number, Name: name, Type: item.Type,
			Weight: item.Weight, Substance: item.Substance, SourceFile: item.SourceFile,
		})
	}
	json.NewEncoder(w).Encode(items)
}

func (s *Server) resolveNoun(id int) string {
	for _, n := range s.parsed.Nouns {
		if n.ID == id {
			return n.Name
		}
	}
	return fmt.Sprintf("noun#%d", id)
}

func (s *Server) resolveAdj(id int) string {
	for _, a := range s.parsed.Adjectives {
		if a.ID == id {
			return a.Name
		}
	}
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("adj#%d", id)
}

func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	vars := mux.Vars(r)
	var num int
	fmt.Sscanf(vars["number"], "%d", &num)
	for _, item := range s.parsed.Items {
		if item.Number == num {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"number":       item.Number,
				"nameId":       item.NameID,
				"resolvedName": s.resolveNoun(item.NameID),
				"type":         item.Type,
				"weight":       item.Weight,
				"volume":       item.Volume,
				"substance":    item.Substance,
				"article":      item.Article,
				"parameter1":   item.Parameter1,
				"parameter2":   item.Parameter2,
				"parameter3":   item.Parameter3,
				"container":    item.Container,
				"interior":     item.Interior,
				"wornSlot":     item.WornSlot,
				"flags":        item.Flags,
				"scripts":      item.Scripts,
				"sourceFile":   item.SourceFile,
			})
			return
		}
	}
	http.Error(w, "item not found", 404)
}

func (s *Server) handleListMonsters(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	json.NewEncoder(w).Encode(s.parsed.Monsters)
}

func (s *Server) handleGetMonster(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	vars := mux.Vars(r)
	var num int
	fmt.Sscanf(vars["number"], "%d", &num)
	for _, mon := range s.parsed.Monsters {
		if mon.Number == num {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"number":      mon.Number,
				"name":        mon.Name,
				"adjective":   mon.Adjective,
				"adjName":     s.resolveAdj(mon.Adjective),
				"description": mon.Description,
				"bodyType":    mon.BodyType,
				"body":        mon.Body,
				"attack1":     mon.Attack1,
				"attack2":     mon.Attack2,
				"defense":     mon.Defense,
				"strategy":    mon.Strategy,
				"treasure":    mon.Treasure,
				"speed":       mon.Speed,
				"armor":       mon.Armor,
				"race":        mon.Race,
				"gender":      mon.Gender,
				"unique":      mon.Unique,
				"scripts":     mon.Scripts,
				"sourceFile":  mon.SourceFile,
			})
			return
		}
	}
	http.Error(w, "monster not found", 404)
}

func (s *Server) handleListNouns(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	json.NewEncoder(w).Encode(s.parsed.Nouns)
}

func (s *Server) handleListAdjectives(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	json.NewEncoder(w).Encode(s.parsed.Adjectives)
}

func (s *Server) handleListCharacters(w http.ResponseWriter, r *http.Request) {
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	players, err := s.engine.ListPlayersByAccount(r.Context(), accountID)
	if err != nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

func (s *Server) handleCreateCharacter(w http.ResponseWriter, r *http.Request) {
	var req CreateCharMsg
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	// Block unverified accounts
	if s.auth != nil {
		acct, err := s.auth.GetAccount(r.Context(), accountID)
		if err == nil && !acct.EmailVerified && acct.GoogleID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(map[string]string{"error": "Please verify your email address before creating a character."})
			return
		}
	}
	// Trim whitespace from names
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	// Validate
	if err := engine.ValidateCharacterInput(req.FirstName, req.LastName, req.Race, req.Gender); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Max 8 characters per account
	existing, _ := s.engine.ListPlayersByAccount(r.Context(), accountID)
	if len(existing) >= 8 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "You can have at most 8 characters per account."})
		return
	}
	// Check unique first name
	taken, _ := s.engine.IsFirstNameTaken(r.Context(), req.FirstName)
	if taken {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "That first name is already taken. Please choose another."})
		return
	}
	player := s.engine.CreateNewPlayer(r.Context(), req.FirstName, req.LastName, req.Race, req.Gender, accountID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func (s *Server) handleGenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	var req struct {
		AllowGM bool `json:"allowGM"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	key, err := s.engine.GenerateAPIKey(r.Context(), vars["firstName"], accountID, req.AllowGM)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"key": key})
}

func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	err := s.engine.RevokeAPIKey(r.Context(), vars["firstName"], accountID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
}

func (s *Server) handleDeleteCharacter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	err := s.engine.SoftDeletePlayer(r.Context(), vars["firstName"], accountID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *Server) handleAdminListDeletedCharacters(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	players, err := s.engine.ListDeletedPlayers(r.Context())
	if err != nil {
		json.NewEncoder(w).Encode([]interface{}{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

func (s *Server) handleAdminRecoverCharacter(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	vars := mux.Vars(r)
	var req struct {
		NewFirstName string `json:"newFirstName"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	player, err := s.engine.RecoverPlayer(r.Context(), vars["firstName"], req.NewFirstName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func (s *Server) handleToggleGM(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	vars := mux.Vars(r)
	var req struct {
		IsGM bool `json:"isGM"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	player, err := s.engine.SetPlayerGM(r.Context(), vars["firstName"], req.IsGM)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	if req.IsGM {
		s.gamelog.Log(gamelog.EventGMGrant, player.FullName(), player.AccountID, "", 0, "")
	} else {
		s.gamelog.Log(gamelog.EventGMRevoke, player.FullName(), player.AccountID, "", 0, "")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

// getAccountID extracts the account ID from the Authorization header JWT.
func (s *Server) getAccountID(r *http.Request) string {
	if s.auth == nil {
		return ""
	}
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	accountID, err := s.auth.ValidateJWT(token)
	if err != nil {
		return ""
	}
	return accountID
}

func (s *Server) handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Error(w, "auth not configured", 500)
		return
	}
	var req struct {
		Credential string `json:"credential"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}

	claims, err := s.auth.VerifyGoogleToken(req.Credential)
	if err != nil {
		log.Printf("Google token verification failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	account, err := s.auth.FindOrCreateAccount(r.Context(), claims)
	if err != nil {
		log.Printf("Account creation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	jwt, err := s.auth.IssueJWT(account)
	if err != nil {
		http.Error(w, "token error", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   jwt,
		"account": account,
	})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Error(w, "auth not configured", 500)
		return
	}
	if !s.checkRateLimit(getClientIP(r), "register", 5, time.Minute) {
		http.Error(w, "too many registration attempts, try again later", 429)
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	account, verifyToken, verifyCode, err := s.auth.RegisterWithPassword(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// Send verification email
	if s.email != nil && s.email.Enabled() {
		if err := s.email.SendVerification(account.Email, verifyToken, verifyCode); err != nil {
			log.Printf("Failed to send verification email to %s: %v", account.Email, err)
		}
	}
	// Issue JWT (account is usable immediately, but email unverified)
	token, err := s.auth.IssueJWT(account)
	if err != nil {
		http.Error(w, "token error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"account": account,
	})
}

func (s *Server) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Error(w, "auth not configured", 500)
		return
	}
	if !s.checkRateLimit(getClientIP(r), "login", 10, time.Minute) {
		http.Error(w, "too many login attempts, try again later", 429)
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	account, err := s.auth.LoginWithPassword(r.Context(), req.Email, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	token, err := s.auth.IssueJWT(account)
	if err != nil {
		http.Error(w, "token error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"account": account,
	})
}

func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Error(w, "auth not configured", 500)
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if err := s.auth.VerifyEmail(r.Context(), req.Token); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "verified"})
}

func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Error(w, "auth not configured", 500)
		return
	}
	if !s.checkRateLimit(getClientIP(r), "forgot-password", 3, time.Hour) {
		http.Error(w, "too many password reset attempts, try again later", 429)
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	account, token, err := s.auth.CreatePasswordResetToken(r.Context(), req.Email)
	if err != nil {
		// Don't reveal errors (prevent email enumeration)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}
	if account != nil && token != "" && s.email != nil && s.email.Enabled() {
		if err := s.email.SendPasswordReset(account.Email, token); err != nil {
			log.Printf("Failed to send password reset email to %s: %v", account.Email, err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Error(w, "auth not configured", 500)
		return
	}
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if err := s.auth.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "password_reset"})
}

func (s *Server) handleResendVerification(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Error(w, "auth not configured", 500)
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	account, token, code, err := s.auth.ResendVerification(r.Context(), req.Email)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if s.email != nil && s.email.Enabled() {
		if err := s.email.SendVerification(account.Email, token, code); err != nil {
			log.Printf("Failed to send verification email to %s: %v", account.Email, err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

func (s *Server) handleVerifyCode(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Error(w, "auth not configured", 500)
		return
	}
	if !s.checkRateLimit(getClientIP(r), "verify-code", 10, time.Minute) {
		http.Error(w, "too many verification attempts, try again later", 429)
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if err := s.auth.VerifyEmailByCode(r.Context(), req.Code); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "verified"})
}

func (s *Server) handleUpdateName(w http.ResponseWriter, r *http.Request) {
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if err := s.auth.UpdateAccountName(r.Context(), accountID, req.Name); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (s *Server) handleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if err := s.auth.SetPassword(r.Context(), accountID, req.Password); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	account, err := s.auth.GetAccount(r.Context(), accountID)
	if err != nil {
		http.Error(w, "account not found", 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

// requireAdmin checks if the request comes from an admin account. Returns the account or writes an error.
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) *auth.Account {
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return nil
	}
	account, err := s.auth.GetAccount(r.Context(), accountID)
	if err != nil || !account.IsAdmin {
		http.Error(w, "forbidden", 403)
		return nil
	}
	return account
}

func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	accounts, err := s.auth.ListAccounts(r.Context())
	if err != nil {
		http.Error(w, "failed to list accounts", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

func (s *Server) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	vars := mux.Vars(r)
	account, err := s.auth.GetAccount(r.Context(), vars["id"])
	if err != nil {
		http.Error(w, "account not found", 404)
		return
	}
	// Also fetch this account's characters
	players, _ := s.engine.ListPlayersByAccount(r.Context(), vars["id"])
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"account":    account,
		"characters": players,
	})
}

func (s *Server) handleToggleAdmin(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	vars := mux.Vars(r)
	var req struct {
		IsAdmin bool `json:"isAdmin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	account, err := s.auth.SetAdmin(r.Context(), vars["id"], req.IsAdmin)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

func (s *Server) handleAdminListCharacters(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	players, err := s.engine.ListPlayers(r.Context())
	if err != nil {
		http.Error(w, "failed to list players", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}

func (s *Server) handleAdminGetCharacter(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	vars := mux.Vars(r)
	player, err := s.engine.GetPlayer(r.Context(), vars["firstName"])
	if err != nil {
		http.Error(w, "player not found", 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func (s *Server) handleAdminReassignCharacter(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	vars := mux.Vars(r)
	var req struct {
		AccountID string `json:"accountId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if req.AccountID == "" {
		http.Error(w, "accountId is required", 400)
		return
	}
	player, err := s.engine.ReassignCharacter(r.Context(), vars["firstName"], req.AccountID)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func (s *Server) handleListCaptures(w http.ResponseWriter, r *http.Request) {
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	sessions, err := s.captures.List(r.Context(), accountID)
	if err != nil {
		http.Error(w, "failed to list captures", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleGetCapture(w http.ResponseWriter, r *http.Request) {
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	vars := mux.Vars(r)
	sess, err := s.captures.Get(r.Context(), vars["id"])
	if err != nil || sess == nil {
		http.Error(w, "capture not found", 404)
		return
	}
	if sess.AccountID != accountID {
		http.Error(w, "forbidden", 403)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

func (s *Server) handleGetCaptureText(w http.ResponseWriter, r *http.Request) {
	accountID := s.getAccountID(r)
	// Also accept token as query param for direct download links
	if accountID == "" {
		if token := r.URL.Query().Get("token"); token != "" && s.auth != nil {
			accountID, _ = s.auth.ValidateJWT(token)
		}
	}
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	vars := mux.Vars(r)
	sess, err := s.captures.Get(r.Context(), vars["id"])
	if err != nil || sess == nil {
		http.Error(w, "capture not found", 404)
		return
	}
	if sess.AccountID != accountID {
		http.Error(w, "forbidden", 403)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"capture-%s.txt\"", vars["id"]))
	w.Write([]byte(sess.ExportText()))
}

func (s *Server) handleDeleteCapture(w http.ResponseWriter, r *http.Request) {
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	vars := mux.Vars(r)
	sess, err := s.captures.Get(r.Context(), vars["id"])
	if err != nil || sess == nil {
		http.Error(w, "capture not found", 404)
		return
	}
	if sess.AccountID != accountID {
		http.Error(w, "forbidden", 403)
		return
	}
	if err := s.captures.Delete(r.Context(), vars["id"]); err != nil {
		http.Error(w, "failed to delete capture", 500)
		return
	}
	w.WriteHeader(204)
}

func (s *Server) handleAdminLogs(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	event := r.URL.Query().Get("event")
	player := r.URL.Query().Get("player")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	entries, err := s.gamelog.Query(r.Context(), event, player, limit)
	if err != nil {
		http.Error(w, "failed to query logs", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleBootstrapAdmin(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Error(w, "auth not configured", 500)
		return
	}
	// Only works if no admins exist
	hasAdmin, err := s.auth.HasAnyAdmin(r.Context())
	if err != nil {
		http.Error(w, "check failed", 500)
		return
	}
	if hasAdmin {
		http.Error(w, "admin already exists", 403)
		return
	}
	// Promote the requesting user
	accountID := s.getAccountID(r)
	if accountID == "" {
		http.Error(w, "unauthorized - log in first", 401)
		return
	}
	account, err := s.auth.SetAdmin(r.Context(), accountID, true)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	log.Printf("Bootstrap: %s (%s) promoted to admin", account.Name, account.Email)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}
