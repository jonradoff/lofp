package api

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jonradoff/lofp/internal/auth"
	"github.com/jonradoff/lofp/internal/engine"
	"github.com/jonradoff/lofp/internal/gamelog"
	"golang.org/x/crypto/ssh"
)

// sshConn implements ClientConn for SSH connections.
type sshConn struct {
	channel ssh.Channel
	mu      sync.Mutex
	width   int
	addr    string
}

func (s *sshConn) SendResult(result *engine.CommandResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var buf strings.Builder
	if result.RoomName != "" {
		buf.WriteString(ansiBoldCyan + result.RoomName + ansiReset + "\r\n")
	}
	if result.RoomDesc != "" {
		buf.WriteString(wordWrap(result.RoomDesc, s.width) + "\r\n")
	}
	if len(result.Exits) > 0 {
		buf.WriteString(ansiGreen + "Obvious exits: " + strings.Join(result.Exits, ", ") + ansiReset + "\r\n")
	}
	if len(result.Items) > 0 {
		for _, item := range result.Items {
			buf.WriteString(ansiYellow + item + ansiReset + "\r\n")
		}
	}
	for _, msg := range result.Messages {
		buf.WriteString(msg + "\r\n")
	}
	if result.Error != "" {
		buf.WriteString(ansiRed + result.Error + ansiReset + "\r\n")
	}
	_, err := s.channel.Write([]byte(buf.String()))
	return err
}

func (s *sshConn) SendBroadcast(messages []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var buf strings.Builder
	for _, msg := range messages {
		if strings.HasPrefix(msg, "**") {
			buf.WriteString(ansiBoldWhite + msg + ansiReset + "\r\n")
		} else {
			buf.WriteString(msg + "\r\n")
		}
	}
	_, err := s.channel.Write([]byte(buf.String()))
	return err
}

func (s *sshConn) SendTypedMessage(msgType string, payload interface{}) error {
	if msgType == "error" {
		if m, ok := payload.(map[string]interface{}); ok {
			if msg, ok := m["message"].(string); ok {
				s.mu.Lock()
				defer s.mu.Unlock()
				_, err := s.channel.Write([]byte(ansiRed + msg + ansiReset + "\r\n"))
				return err
			}
		}
	}
	return nil
}

func (s *sshConn) Close() error {
	return s.channel.Close()
}

func (s *sshConn) RemoteAddr() string {
	return s.addr
}

func (s *sshConn) writeLine(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channel.Write([]byte(text + "\r\n"))
}

func (s *sshConn) writePrompt(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channel.Write([]byte(text))
}

// readLine reads a line from the SSH channel with basic line editing.
func (s *sshConn) readLine(timeout time.Duration, echo bool) (string, error) {
	var result []byte
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("read timeout")
		}
		buf := make([]byte, 1)
		// Set a shorter read deadline so we can check the overall timeout
		s.channel.(*wrappedChannel).SetReadDeadline(time.Now().Add(time.Second))
		n, err := s.channel.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return "", err
		}
		if n == 0 {
			continue
		}
		b := buf[0]
		switch b {
		case '\r', '\n':
			if echo {
				s.channel.Write([]byte("\r\n"))
			}
			return string(result), nil
		case 127, 8: // backspace/delete
			if len(result) > 0 {
				result = result[:len(result)-1]
				if echo {
					s.channel.Write([]byte("\b \b"))
				}
			}
		default:
			if b >= 32 && b < 127 {
				result = append(result, b)
				if echo {
					s.channel.Write([]byte{b})
				} else {
					s.channel.Write([]byte("*"))
				}
			}
		}
	}
}

// wrappedChannel wraps ssh.Channel to support SetReadDeadline via a pipe.
type wrappedChannel struct {
	ssh.Channel
	reader io.Reader
	done   chan struct{}
}

func newWrappedChannel(ch ssh.Channel) *wrappedChannel {
	return &wrappedChannel{Channel: ch, reader: ch, done: make(chan struct{})}
}

func (w *wrappedChannel) Read(p []byte) (int, error) {
	return w.reader.Read(p)
}

func (w *wrappedChannel) SetReadDeadline(t time.Time) error {
	// SSH channels don't support deadlines natively, use a goroutine
	return nil
}

// ListenSSH starts an SSH server for MUD client connections.
func (s *Server) ListenSSH(addr string) {
	if s.auth == nil {
		log.Println("SSH server disabled: auth service not configured")
		return
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	// Use persistent host key from env, or generate ephemeral one
	var signer ssh.Signer
	if keyPEM := os.Getenv("SSH_HOST_KEY"); keyPEM != "" {
		var err error
		signer, err = ssh.ParsePrivateKey([]byte(keyPEM))
		if err != nil {
			log.Printf("SSH: failed to parse SSH_HOST_KEY: %v", err)
			return
		}
		log.Println("SSH: using persistent host key from SSH_HOST_KEY")
	} else {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			log.Printf("SSH: failed to generate host key: %v", err)
			return
		}
		signer, err = ssh.NewSignerFromKey(priv)
		if err != nil {
			log.Printf("SSH: failed to create signer: %v", err)
			return
		}
		log.Println("SSH: using ephemeral host key (set SSH_HOST_KEY for persistence)")
	}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("SSH: failed to listen on %s: %v", addr, err)
		return
	}
	log.Printf("SSH server listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("SSH: accept error: %v", err)
			continue
		}
		go s.handleSSHConn(conn, config)
	}
}

func (s *Server) handleSSHConn(rawConn net.Conn, config *ssh.ServerConfig) {
	sshServerConn, chans, reqs, err := ssh.NewServerConn(rawConn, config)
	if err != nil {
		log.Printf("SSH: handshake failed from %s: %v", rawConn.RemoteAddr(), err)
		return
	}
	defer sshServerConn.Close()
	log.Printf("SSH: connection from %s", sshServerConn.RemoteAddr())

	// Discard global requests
	go ssh.DiscardRequests(reqs)

	// Handle channel requests
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("SSH: could not accept channel: %v", err)
			continue
		}

		// Handle session requests (pty, shell)
		go func(in <-chan *ssh.Request) {
			for req := range in {
				switch req.Type {
				case "pty-req":
					req.Reply(true, nil)
				case "shell":
					req.Reply(true, nil)
				case "window-change":
					req.Reply(false, nil)
				default:
					req.Reply(false, nil)
				}
			}
		}(requests)

		// Run the game session on this channel with interactive auth
		s.handleSSHSession(channel, rawConn.RemoteAddr().String())
		return // one session per connection
	}
}

func (s *Server) handleSSHSession(channel ssh.Channel, remoteAddr string) {
	defer channel.Close()

	sc := &sshConn{
		channel: newWrappedChannel(channel),
		width:   80,
		addr:    remoteAddr,
	}

	ip := extractIP(remoteAddr)

	// Per-IP connection limit
	s.connMu.Lock()
	if s.connsByIP[ip] >= 5 {
		s.connMu.Unlock()
		sc.writeLine("Too many connections from your address.")
		return
	}
	s.connsByIP[ip]++
	s.connMu.Unlock()
	defer func() {
		s.connMu.Lock()
		s.connsByIP[ip]--
		if s.connsByIP[ip] <= 0 {
			delete(s.connsByIP, ip)
		}
		s.connMu.Unlock()
	}()

	// Banner
	sc.writeLine("")
	sc.writeLine(ansiYellow + `    __                              __` + ansiReset)
	sc.writeLine(ansiYellow + `   / /  ___  ____  ___  ____  ____/ /____` + ansiReset)
	sc.writeLine(ansiYellow + `  / /  / _ \/ __ \/ _ \/ __ \/ __  / ___/` + ansiReset)
	sc.writeLine(ansiYellow + ` / /__/  __/ /_/ /  __/ / / / /_/ (__  )` + ansiReset)
	sc.writeLine(ansiYellow + `/_____|\___|\_,_/\___/_/ /_/\__,_/____/` + ansiReset)
	sc.writeLine(ansiYellow + `    ____  / __/` + ansiReset)
	sc.writeLine(ansiYellow + `   / __ \/ /_` + ansiReset)
	sc.writeLine(ansiYellow + `  / /_/ / __/` + ansiReset)
	sc.writeLine(ansiYellow + ` / .___/_/` + ansiReset)
	sc.writeLine(ansiYellow + `/_/   ____      __` + ansiReset)
	sc.writeLine(ansiYellow + `     / __/_  __/ /___  __________` + ansiReset)
	sc.writeLine(ansiYellow + `    / /_/ / / / __/ / / / ___/ _ \` + ansiReset)
	sc.writeLine(ansiYellow + `   / __/ /_/ / /_/ /_/ / /  /  __/` + ansiReset)
	sc.writeLine(ansiYellow + `  /_/  \__,_/\__/\__,_/_/   \___/` + ansiReset)
	sc.writeLine(ansiYellow + `    ____            __` + ansiReset)
	sc.writeLine(ansiYellow + `   / __ \____ _____/ /_` + ansiReset)
	sc.writeLine(ansiYellow + `  / /_/ / __ ` + "`" + `/ ___/ __/` + ansiReset)
	sc.writeLine(ansiYellow + ` / ____/ /_/ (__  ) /_` + ansiReset)
	sc.writeLine(ansiYellow + `/_/    \__,_/____/\__/` + ansiReset)
	sc.writeLine("")
	sc.writeLine("  The Shattered Realms of Andor await your return...")
	sc.writeLine("")

	// Interactive authentication (same flow as telnet)
	account, player := s.sshAuthenticate(sc)
	if player == nil {
		sc.writeLine("Disconnected.")
		return
	}

	accountID := account.ID.Hex()

	// Check for existing session
	s.mu.Lock()
	if oldSess, ok := s.sessions[player.FirstName]; ok {
		oldSess.Conn.Close()
		delete(s.sessions, player.FirstName)
	}
	s.mu.Unlock()

	session := &Session{
		Player:       player,
		Conn:         sc,
		lastActivity: time.Now(),
	}

	s.mu.Lock()
	s.sessions[player.FirstName] = session
	s.mu.Unlock()

	s.hub.RegisterPlayer(player.FirstName, player.FullName(), player.RoomNumber,
		player.Race, player.RaceName(), player.Position,
		player.IsGM, player.GMHat, player.GMHidden, player.GMInvis, player.Hidden)

	s.gamelog.Log(gamelog.EventLogin, player.FullName(), accountID,
		fmt.Sprintf("ssh from %s (%s)", ip, account.Email), player.RoomNumber, "")
	s.gamelog.Log(gamelog.EventGameEnter, player.FullName(), accountID,
		fmt.Sprintf("ssh from %s", ip), player.RoomNumber, "")

	s.broadcastGlobal(player.FirstName,
		[]string{fmt.Sprintf("** %s has just entered the Realms.", player.FirstName)})
	s.broadcastToRoom(player.RoomNumber, player.FirstName,
		[]string{fmt.Sprintf("%s materializes from the mists.", player.FirstName)})

	sc.writeLine(fmt.Sprintf("\r\nWelcome back, %s the %s!", player.FullName(), player.RaceName()))
	sc.writeLine("")

	ctx := context.Background()
	enterResult := s.engine.EnterRoom(ctx, player)
	if enterResult != nil {
		enterResult.PlayerState = player
		s.sendResult(session, enterResult)
	}

	// Command loop
	s.sshCommandLoop(ctx, session, sc)

	// Cleanup — only if WE are still the active session for this character.
	s.mu.Lock()
	currentSess, isActive := s.sessions[player.FirstName]
	isActive = isActive && currentSess == session
	if isActive {
		delete(s.sessions, player.FirstName)
	}
	s.mu.Unlock()

	if isActive {
		if !session.quitSent {
			s.broadcastGlobal(player.FirstName,
				[]string{fmt.Sprintf("** %s has just left the Realms.", player.FirstName)})
		}
		s.broadcastToRoom(player.RoomNumber, player.FirstName,
			[]string{fmt.Sprintf("%s fades from the Realms.", player.FirstName)})
		s.gamelog.Log(gamelog.EventGameExit, player.FullName(), accountID,
			fmt.Sprintf("ssh from %s", ip), player.RoomNumber, "")
		s.hub.UnregisterPlayer(player.FirstName)
	}

	sc.writeLine("Farewell from the Shattered Realms.")
}

func (s *Server) sshAuthenticate(sc *sshConn) (*auth.Account, *engine.Player) {
	ctx := context.Background()

	// Show active banner before login menu
	if s.engine != nil {
		if banner := s.engine.GetBanner(); banner != "" {
			sc.writeLine("")
			sc.writeLine(ansiYellow + "╔══════════════════════════════════════════════════════════════╗" + ansiReset)
			sc.writeLine(ansiYellow + "║  SERVER NOTICE                                               ║" + ansiReset)
			sc.writeLine(ansiYellow + "║  " + ansiBold + banner + ansiReset + ansiYellow + strings.Repeat(" ", max(0, 60-len(banner))) + "║" + ansiReset)
			sc.writeLine(ansiYellow + "╚══════════════════════════════════════════════════════════════╝" + ansiReset)
			sc.writeLine("")
		}
	}

	for attempts := 0; attempts < 3; attempts++ {
		sc.writeLine("Login options:")
		sc.writeLine("  1) Sign in with email/password")
		sc.writeLine("  2) Create a new account")
		sc.writeLine("  Q) Quit")
		sc.writeLine("")
		sc.writePrompt("Choice: ")

		choice, err := sc.readLine(2*time.Minute, true)
		if err != nil {
			return nil, nil
		}
		choice = strings.TrimSpace(choice)

		switch strings.ToUpper(choice) {
		case "1":
			acct := s.sshLoginByPassword(sc, ctx)
			if acct != nil {
				if !acct.EmailVerified && acct.GoogleID == "" {
					if !s.sshVerifyPrompt(sc, ctx, acct) {
						continue
					}
					acct, _ = s.auth.GetAccount(ctx, acct.ID.Hex())
				}
				p := s.sshCharacterSelect(sc, ctx, acct)
				if p != nil {
					return acct, p
				}
			}
		case "2":
			acct := s.sshRegister(sc, ctx)
			if acct != nil {
				if !s.sshVerifyPrompt(sc, ctx, acct) {
					continue
				}
				acct, _ = s.auth.GetAccount(ctx, acct.ID.Hex())
				p := s.sshCharacterSelect(sc, ctx, acct)
				if p != nil {
					return acct, p
				}
			}
		case "Q", "QUIT":
			return nil, nil
		default:
			sc.writeLine(ansiRed + "Invalid choice." + ansiReset)
		}

		sc.writeLine(ansiRed + "Login failed. Try again." + ansiReset)
		sc.writeLine("")
	}

	sc.writeLine("Too many failed attempts.")
	return nil, nil
}

func (s *Server) sshLoginByPassword(sc *sshConn, ctx context.Context) *auth.Account {
	if s.auth == nil {
		sc.writeLine(ansiRed + "Authentication service is not configured." + ansiReset)
		return nil
	}
	sc.writePrompt("Email: ")
	email, err := sc.readLine(time.Minute, true)
	if err != nil {
		return nil
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return nil
	}

	sc.writePrompt("Password: ")
	password, err := sc.readLine(time.Minute, false)
	if err != nil {
		return nil
	}
	password = strings.TrimSpace(password)

	account, err := s.auth.LoginWithPassword(ctx, email, password)
	if err != nil {
		sc.writeLine(ansiRed + err.Error() + ansiReset)
		return nil
	}
	sc.writeLine(ansiGreen + fmt.Sprintf("Welcome, %s!", account.Name) + ansiReset)
	return account
}

func (s *Server) sshRegister(sc *sshConn, ctx context.Context) *auth.Account {
	if s.auth == nil {
		sc.writeLine(ansiRed + "Authentication service is not configured." + ansiReset)
		return nil
	}
	sc.writeLine("")
	sc.writeLine(ansiBoldCyan + "=== Create Account ===" + ansiReset)
	sc.writeLine("")

	sc.writePrompt("Display name: ")
	name, err := sc.readLine(time.Minute, true)
	if err != nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		sc.writeLine(ansiRed + "Name is required." + ansiReset)
		return nil
	}

	sc.writePrompt("Email: ")
	email, err := sc.readLine(time.Minute, true)
	if err != nil {
		return nil
	}
	email = strings.TrimSpace(email)
	if email == "" {
		sc.writeLine(ansiRed + "Email is required." + ansiReset)
		return nil
	}

	sc.writeLine(ansiYellow + "Password must be 10+ characters with uppercase, lowercase, digit, and special character." + ansiReset)
	sc.writePrompt("Password: ")
	password, err := sc.readLine(time.Minute, false)
	if err != nil {
		return nil
	}
	password = strings.TrimSpace(password)

	sc.writePrompt("Confirm password: ")
	confirm, err := sc.readLine(time.Minute, false)
	if err != nil {
		return nil
	}
	confirm = strings.TrimSpace(confirm)

	if password != confirm {
		sc.writeLine(ansiRed + "Passwords do not match." + ansiReset)
		return nil
	}

	account, verifyToken, verifyCode, err := s.auth.RegisterWithPassword(ctx, email, password, name)
	if err != nil {
		sc.writeLine(ansiRed + err.Error() + ansiReset)
		return nil
	}

	if s.email != nil && s.email.Enabled() {
		if err := s.email.SendVerification(account.Email, verifyToken, verifyCode); err != nil {
			log.Printf("Failed to send verification email to %s: %v", account.Email, err)
		}
		sc.writeLine(ansiGreen + "Account created! A verification email has been sent to " + email + "." + ansiReset)
	} else {
		sc.writeLine(ansiGreen + "Account created!" + ansiReset)
	}
	return account
}

func (s *Server) sshVerifyPrompt(sc *sshConn, ctx context.Context, account *auth.Account) bool {
	sc.writeLine("")
	sc.writeLine(ansiYellow + "Your email address is not yet verified." + ansiReset)
	sc.writeLine("Check your email for a verification code, then enter it below.")
	sc.writeLine("(You can also press Enter to skip and come back later.)")
	sc.writeLine("")

	for i := 0; i < 3; i++ {
		sc.writePrompt("Verification code: ")
		code, err := sc.readLine(2*time.Minute, true)
		if err != nil {
			return false
		}
		code = strings.TrimSpace(code)
		if code == "" {
			sc.writeLine("Skipped. You must verify your email before you can create or play a character.")
			return false
		}
		if err := s.auth.VerifyEmailByCode(ctx, code); err != nil {
			sc.writeLine(ansiRed + err.Error() + ansiReset)
			continue
		}
		sc.writeLine(ansiGreen + "Email verified!" + ansiReset)
		return true
	}
	sc.writeLine(ansiRed + "Too many failed attempts." + ansiReset)
	return false
}

func (s *Server) sshCharacterSelect(sc *sshConn, ctx context.Context, account *auth.Account) *engine.Player {
	accountID := account.ID.Hex()

	for {
		players, _ := s.engine.ListPlayersByAccount(ctx, accountID)

		sc.writeLine("")
		sc.writeLine(ansiBoldCyan + "=== Character Selection ===" + ansiReset)
		if len(players) > 0 {
			for i, p := range players {
				sc.writeLine(fmt.Sprintf("  %d) %s %s (Level %d %s)",
					i+1, p.FirstName, p.LastName, p.Level, engine.RaceNameByID(p.Race)))
			}
		} else {
			sc.writeLine("  No characters yet.")
		}
		sc.writeLine("  N) Create a new character")
		sc.writeLine("  Q) Quit")
		sc.writeLine("")
		sc.writePrompt("Choice: ")

		input, err := sc.readLine(2*time.Minute, true)
		if err != nil {
			return nil
		}
		input = strings.TrimSpace(input)
		upper := strings.ToUpper(input)

		if upper == "Q" {
			return nil
		}
		if upper == "N" {
			p := s.sshCreateCharacter(sc, ctx, accountID)
			if p != nil {
				return p
			}
			continue
		}

		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err == nil && idx >= 1 && idx <= len(players) {
			p := players[idx-1]
			player, err := s.engine.LoadPlayer(ctx, p.FirstName, p.LastName)
			if err != nil {
				sc.writeLine(ansiRed + "Failed to load character." + ansiReset)
				continue
			}
			return player
		}

		sc.writeLine(ansiRed + "Invalid choice." + ansiReset)
	}
}

func (s *Server) sshCreateCharacter(sc *sshConn, ctx context.Context, accountID string) *engine.Player {
	existing, _ := s.engine.ListPlayersByAccount(ctx, accountID)
	if len(existing) >= 8 {
		sc.writeLine(ansiRed + "You can have at most 8 characters." + ansiReset)
		return nil
	}

	sc.writeLine("")
	sc.writeLine(ansiBoldCyan + "=== Create Character ===" + ansiReset)
	sc.writeLine("")

	sc.writePrompt("First name: ")
	firstName, err := sc.readLine(time.Minute, true)
	if err != nil {
		return nil
	}
	firstName = strings.TrimSpace(firstName)

	sc.writePrompt("Last name: ")
	lastName, err := sc.readLine(time.Minute, true)
	if err != nil {
		return nil
	}
	lastName = strings.TrimSpace(lastName)

	sc.writeLine("")
	sc.writeLine("Races:")
	sc.writeLine("  1) Human      2) Aelfen     3) Highlander  4) Wolfling")
	sc.writeLine("  5) Murg       6) Drakin     7) Mechanoid   8) Ephemeral")
	sc.writePrompt("Race (1-8): ")
	raceStr, err := sc.readLine(time.Minute, true)
	if err != nil {
		return nil
	}
	var race int
	fmt.Sscanf(strings.TrimSpace(raceStr), "%d", &race)

	sc.writeLine("Gender: 1) Male  2) Female")
	sc.writePrompt("Gender (1-2): ")
	genderStr, err := sc.readLine(time.Minute, true)
	if err != nil {
		return nil
	}
	var gender int
	fmt.Sscanf(strings.TrimSpace(genderStr), "%d", &gender)

	if err := engine.ValidateCharacterInput(firstName, lastName, race, gender); err != nil {
		sc.writeLine(ansiRed + err.Error() + ansiReset)
		return nil
	}
	taken, _ := s.engine.IsFirstNameTaken(ctx, firstName)
	if taken {
		sc.writeLine(ansiRed + "That first name is already taken." + ansiReset)
		return nil
	}

	player := s.engine.CreateNewPlayer(ctx, firstName, lastName, race, gender, accountID)
	sc.writeLine(ansiGreen + fmt.Sprintf("Welcome to the Shattered Realms, %s the %s!", player.FullName(), engine.RaceNameByID(player.Race)) + ansiReset)
	return player
}

func (s *Server) sshCommandLoop(ctx context.Context, session *Session, sc *sshConn) {
	for {
		prompt := buildPrompt(session.Player, false)
		sc.writePrompt(prompt)
		input, err := sc.readLine(30*time.Minute, true)
		if err != nil {
			return
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		session.lastActivity = time.Now()

		// Rate limiting (same as telnet)
		now := time.Now()
		if now.Sub(session.lastCmdTime) > time.Second {
			session.cmdCount = 0
			session.lastCmdTime = now
		}
		session.cmdCount++
		if session.cmdCount > 4 {
			sc.writeLine("[Slow down! Too many commands.]")
			continue
		}

		cutoff := now.Add(-10 * time.Second)
		var recentCmds []time.Time
		for _, t := range session.cmdTimes {
			if t.After(cutoff) {
				recentCmds = append(recentCmds, t)
			}
		}
		session.cmdTimes = append(recentCmds, now)
		if len(session.cmdTimes) > 10 {
			sc.writeLine("[Slow down! Too many commands.]")
			continue
		}

		result := s.engine.ProcessCommand(ctx, session.Player, input)
		result.PlayerState = session.Player
		result.PromptIndicators = session.Player.PromptIndicators()
		s.sendResult(session, result)

		if len(result.RoomBroadcast) > 0 {
			chatNow := time.Now()
			chatCutoff := chatNow.Add(-10 * time.Second)
			var recent []time.Time
			for _, t := range session.chatTimes {
				if t.After(chatCutoff) {
					recent = append(recent, t)
				}
			}
			session.chatTimes = recent
			if len(session.chatTimes) >= 5 {
				sc.writeLine("[You are sending messages too quickly. Please wait.]")
				continue
			}
			session.chatTimes = append(session.chatTimes, chatNow)
		}

		s.dispatchCommandResult(session, result)

		if result.Quit {
			return
		}
	}
}
