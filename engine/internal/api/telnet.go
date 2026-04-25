package api

import (
	"bufio"
	"compress/zlib"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jonradoff/lofp/internal/auth"
	"github.com/jonradoff/lofp/internal/engine"
	"github.com/jonradoff/lofp/internal/gamelog"
)

// Telnet protocol constants
const (
	iacByte  = 255
	doByte   = 253
	dontByte = 254
	willByte = 251
	wontByte = 252
	sbByte   = 250 // sub-negotiation begin
	seByte   = 240 // sub-negotiation end

	optEcho  = 1  // Echo
	optSGA   = 3  // Suppress Go-Ahead
	optTType = 24 // Terminal Type (MTTS)
	optNAWS  = 31 // Negotiate About Window Size
	optMSDP  = 69 // MUD Server Data Protocol
	optMSSP  = 70 // MUD Server Status Protocol
	optMCCP2 = 86 // MUD Client Compression Protocol v2
	optMXP   = 91 // MUD eXtension Protocol
	optGMCP  = 201 // Generic MUD Communication Protocol

	// MSSP sub-negotiation
	msspVar = 1
	msspVal = 2

	// MSDP sub-negotiation
	msdpVar        = 1
	msdpVal        = 2
	msdpTableOpen  = 3
	msdpTableClose = 4
	msdpArrayOpen  = 5
	msdpArrayClose = 6

	// MXP line mode escape sequences
	// \033[1z = secure line (allows secure tags like <send> until next newline)
	// \033[7z = lock locked mode (no MXP parsing)
	mxpSecureLine = "\033[1z"
	mxpLockedLine = "\033[7z"
)

// ANSI color codes for MUD output
const (
	ansiReset     = "\033[0m"
	ansiBoldCyan  = "\033[1;36m"
	ansiGreen     = "\033[32m"
	ansiYellow    = "\033[33m"
	ansiRed       = "\033[31m"
	ansiMagenta   = "\033[35m"
	ansiBoldWhite = "\033[1;37m"
	ansiBold      = "\033[1m"
)

// telnetConn implements ClientConn for raw TCP/telnet connections.
type telnetConn struct {
	conn   net.Conn
	reader *bufio.Reader
	mu     sync.Mutex
	width  int
	isTLS  bool

	// Protocol capability flags (set during negotiation)
	gmcpEnabled bool
	mccpEnabled bool
	mxpEnabled  bool
	msdpEnabled bool
	msspEnabled bool
	mttsEnabled bool

	// GMCP: packages the client has subscribed to
	gmcpSupported map[string]bool

	// MSDP: variables the client is reporting on
	msdpReporting map[string]bool

	// Password mode flag
	passwordMode bool

	// MCCP2: compressed writer (nil until activated)
	compWriter   *zlib.Writer
	compActivated bool

	// Server reference for MSSP player count
	server *Server
}

func newTelnetConn(conn net.Conn, isTLS bool) *telnetConn {
	return &telnetConn{
		conn:          conn,
		reader:        bufio.NewReader(conn),
		width:         80,
		isTLS:         isTLS,
		gmcpSupported: make(map[string]bool),
		msdpReporting: make(map[string]bool),
	}
}

// write sends data through the compressor if MCCP2 is active, otherwise raw.
func (t *telnetConn) write(data []byte) (int, error) {
	if t.compActivated && t.compWriter != nil {
		n, err := t.compWriter.Write(data)
		if err != nil {
			return n, err
		}
		return n, t.compWriter.Flush()
	}
	return t.conn.Write(data)
}

func (t *telnetConn) SendResult(result *engine.CommandResult) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var buf strings.Builder

	// The engine's Messages[] already contains the full formatted room output
	// (room name, description, items, players, monsters, exits).
	// We render Messages directly and apply MXP/ANSI coloring inline,
	// rather than rendering from the individual fields (which would duplicate).
	for _, msg := range result.Messages {
		colored := t.colorizeMessage(msg)
		buf.WriteString(colored + "\r\n")
	}
	if result.Error != "" {
		buf.WriteString(ansiRed + result.Error + ansiReset + "\r\n")
	}

	_, err := t.write([]byte(buf.String()))
	return err
}

// colorizeMessage applies ANSI colors and MXP markup to a message line.
func (t *telnetConn) colorizeMessage(msg string) string {
	// Room name lines: [Name]
	if strings.HasPrefix(msg, "[") && strings.HasSuffix(msg, "]") {
		return ansiBoldCyan + msg + ansiReset
	}

	// Obvious exits line
	if strings.HasPrefix(msg, "Obvious exits: ") {
		if t.mxpEnabled {
			exitsPart := strings.TrimPrefix(msg, "Obvious exits: ")
			exitsPart = strings.TrimSuffix(exitsPart, ".")
			exits := strings.Split(exitsPart, ", ")
			var b strings.Builder
			b.WriteString(mxpSecureLine + ansiGreen + "Obvious exits: ")
			for i, exit := range exits {
				if i > 0 {
					b.WriteString(", ")
				}
				exit = strings.TrimSpace(exit)
				b.WriteString("<send href=\"" + exit + "\">" + exit + "</send>")
			}
			b.WriteString("." + ansiReset)
			return b.String()
		}
		return ansiGreen + msg + ansiReset
	}

	// "There are no obvious exits."
	if msg == "There are no obvious exits." {
		return ansiGreen + msg + ansiReset
	}

	return msg
}

func (t *telnetConn) SendBroadcast(messages []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	var buf strings.Builder
	for _, msg := range messages {
		if strings.HasPrefix(msg, "**") {
			buf.WriteString(ansiBoldWhite + msg + ansiReset + "\r\n")
		} else {
			buf.WriteString(msg + "\r\n")
		}
	}
	_, err := t.write([]byte(buf.String()))
	return err
}

func (t *telnetConn) SendTypedMessage(msgType string, payload interface{}) error {
	if msgType == "error" {
		if m, ok := payload.(map[string]interface{}); ok {
			if msg, ok := m["message"].(string); ok {
				t.mu.Lock()
				defer t.mu.Unlock()
				_, err := t.write([]byte(ansiRed + msg + ansiReset + "\r\n"))
				return err
			}
		}
	}
	return nil
}

func (t *telnetConn) Close() error {
	if t.compWriter != nil {
		t.compWriter.Close()
	}
	return t.conn.Close()
}

func (t *telnetConn) RemoteAddr() string {
	return t.conn.RemoteAddr().String()
}

func (t *telnetConn) writeLine(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.write([]byte(text + "\r\n"))
}

func (t *telnetConn) writeRaw(data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.write(data)
}

func (t *telnetConn) writePrompt(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.write([]byte(text))
}

// readLine reads a line from telnet, stripping IAC sequences.
// Client handles its own echo in normal mode (WONT ECHO).
func (t *telnetConn) readLine(timeout time.Duration) (string, error) {
	return t.readLineInternal(timeout)
}

// enterPasswordMode suppresses client echo for password entry.
// Sends WILL ECHO through the appropriate channel (compressor if MCCP2 active).
// Must be called BEFORE writing the prompt text.
func (t *telnetConn) enterPasswordMode() {
	t.passwordMode = true
	// Send WILL ECHO — must go through compressor if MCCP2 is active
	t.write([]byte{iacByte, willByte, optEcho})
	time.Sleep(50 * time.Millisecond)
}

// exitPasswordMode restores normal client echo after password entry.
func (t *telnetConn) exitPasswordMode() {
	t.passwordMode = false
	t.write([]byte{iacByte, wontByte, optEcho})
	t.write([]byte("\r\n"))
}

// readLineEcho is kept for backwards compatibility (SSH code uses it).
func (t *telnetConn) readLineEcho(timeout time.Duration, echo bool) (string, error) {
	if !echo {
		t.enterPasswordMode()
		defer t.exitPasswordMode()
	}
	return t.readLineInternal(timeout)
}

func (t *telnetConn) readLineInternal(timeout time.Duration) (string, error) {
	t.conn.SetReadDeadline(time.Now().Add(timeout))
	var result []byte
	for {
		b, err := t.reader.ReadByte()
		if err != nil {
			log.Printf("telnet readLine: error after %d bytes: %v", len(result), err)
			return "", err
		}
		if b == iacByte {
			cmd, err := t.reader.ReadByte()
			if err != nil {
				return "", err
			}
			switch cmd {
			case willByte, wontByte, doByte, dontByte:
				opt, _ := t.reader.ReadByte()
				log.Printf("telnet readLine: got IAC %d %d (inline negotiation)", cmd, opt)
			case sbByte:
				opt, err := t.reader.ReadByte()
				if err != nil {
					return "", err
				}
				var sbData []byte
				for {
					sb, err := t.reader.ReadByte()
					if err != nil {
						return "", err
					}
					if sb == iacByte {
						se, err := t.reader.ReadByte()
						if err != nil {
							return "", err
						}
						if se == seByte {
							break
						}
						sbData = append(sbData, sb)
						sbData = append(sbData, se)
					} else {
						sbData = append(sbData, sb)
					}
				}
				log.Printf("telnet readLine: got IAC SB %d (%d bytes subneg)", opt, len(sbData))
				t.handleSubneg(opt, sbData)
			case iacByte:
				// Escaped 0xFF — treat as literal
				result = append(result, 0xFF)
			}
			continue
		}
		if b == '\r' {
			// Consume optional trailing \n or \0
			// Use a short deadline to avoid blocking if nothing follows
			t.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			next, err := t.reader.ReadByte()
			if err == nil {
				if next != '\n' && next != 0 {
					t.reader.UnreadByte() // put it back
				}
			}
			// Restore original deadline
			t.conn.SetReadDeadline(time.Now().Add(timeout))
			break
		}
		if b == '\n' {
			break
		}
		if b == 8 || b == 127 {
			if len(result) > 0 {
				result = result[:len(result)-1]
			}
			continue
		}
		if b >= 32 && b < 127 {
			result = append(result, b)
		}
	}
	log.Printf("telnet readLine: returning %q (%d bytes)", string(result), len(result))
	return string(result), nil
}

// handleSubneg processes subnegotiation data received during readLine.
func (t *telnetConn) handleSubneg(opt byte, data []byte) {
	switch opt {
	case optGMCP:
		t.handleGMCPMessage(string(data))
	case optMSDP:
		t.handleMSDPMessage(data)
	case optNAWS:
		if len(data) >= 4 {
			width := int(data[0])<<8 | int(data[1])
			if width > 20 && width < 300 {
				t.width = width
			}
		}
	}
}

// negotiate sends initial telnet option negotiation and handles client responses.
func (t *telnetConn) negotiate() {
	log.Printf("telnet: negotiate start for %s", t.conn.RemoteAddr())

	// Send our options
	// WONT ECHO = client handles its own echo (normal mode)
	// We switch to WILL ECHO only for password prompts
	t.conn.Write([]byte{
		iacByte, wontByte, optEcho, // Client handles echo (normal)
		iacByte, willByte, optSGA,  // Suppress go-ahead
		iacByte, doByte, optNAWS,   // Request terminal size
		iacByte, willByte, optGMCP,  // Offer GMCP
		iacByte, willByte, optMCCP2, // Offer MCCP2
		iacByte, willByte, optMSSP,  // Offer MSSP
		iacByte, willByte, optMSDP,  // Offer MSDP
		iacByte, doByte, optMXP,     // Request MXP
	})

	// Read and respond to client negotiation for up to 2 seconds
	// Mudlet sends many options and subnegotiations that can take >1s
	t.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		if t.reader.Buffered() == 0 {
			_, err := t.reader.Peek(1)
			if err != nil {
				break
			}
		}
		b, err := t.reader.ReadByte()
		if err != nil {
			break
		}
		if b != iacByte {
			continue
		}

		cmd, err := t.reader.ReadByte()
		if err != nil {
			break
		}
		switch cmd {
		case willByte:
			opt, err := t.reader.ReadByte()
			if err != nil {
				break
			}
			switch opt {
			case optNAWS:
				t.conn.Write([]byte{iacByte, doByte, optNAWS})
			case optGMCP:
				t.gmcpEnabled = true
				t.conn.Write([]byte{iacByte, doByte, optGMCP})
				log.Printf("telnet: client supports GMCP")
			case optMXP:
				t.mxpEnabled = true
				t.conn.Write([]byte{iacByte, doByte, optMXP})
				log.Printf("telnet: client supports MXP")
			case optMSDP:
				t.msdpEnabled = true
				t.conn.Write([]byte{iacByte, doByte, optMSDP})
				log.Printf("telnet: client supports MSDP")
			case optTType:
				t.mttsEnabled = true
				t.conn.Write([]byte{iacByte, doByte, optTType})
			default:
				t.conn.Write([]byte{iacByte, dontByte, opt})
			}
		case doByte:
			opt, err := t.reader.ReadByte()
			if err != nil {
				break
			}
			switch opt {
			case optSGA:
				// Already sent WILL SGA — confirmed
			case optEcho:
				// Client says DO ECHO but we said WONT — reject
				t.conn.Write([]byte{iacByte, wontByte, optEcho})
			case optGMCP:
				t.gmcpEnabled = true
				log.Printf("telnet: client confirmed GMCP")
			case optMCCP2:
				t.mccpEnabled = true
				log.Printf("telnet: client confirmed MCCP2")
			case optMSSP:
				t.msspEnabled = true
				log.Printf("telnet: client confirmed MSSP")
			case optMSDP:
				t.msdpEnabled = true
				log.Printf("telnet: client confirmed MSDP")
			default:
				t.conn.Write([]byte{iacByte, wontByte, opt})
			}
		case wontByte, dontByte:
			opt, _ := t.reader.ReadByte()
			_ = opt
		case sbByte:
			opt, err := t.reader.ReadByte()
			if err != nil {
				break
			}
			var sbData []byte
			for {
				sb, err := t.reader.ReadByte()
				if err != nil {
					goto done
				}
				if sb == iacByte {
					se, err := t.reader.ReadByte()
					if err != nil {
						goto done
					}
					if se == seByte {
						break
					}
					sbData = append(sbData, sb)
					sbData = append(sbData, se)
				} else {
					sbData = append(sbData, sb)
				}
			}
			t.handleSubneg(opt, sbData)
		case iacByte:
			// Escaped 0xFF
		}
	}
done:
	// Drain any remaining negotiation data that arrived late
	t.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	for t.reader.Buffered() > 0 {
		b, err := t.reader.ReadByte()
		if err != nil {
			break
		}
		if b == iacByte {
			cmd, err := t.reader.ReadByte()
			if err != nil {
				break
			}
			switch cmd {
			case willByte, wontByte, doByte, dontByte:
				t.reader.ReadByte()
			case sbByte:
				// Consume until IAC SE
				for {
					sb, err := t.reader.ReadByte()
					if err != nil {
						goto drainDone
					}
					if sb == iacByte {
						se, err := t.reader.ReadByte()
						if err != nil {
							goto drainDone
						}
						if se == seByte {
							break
						}
					}
				}
			}
		}
	}
drainDone:
	t.conn.SetReadDeadline(time.Time{})

	// Post-negotiation: send protocol-specific init data
	if t.gmcpEnabled {
		t.sendGMCP("Core.Hello", map[string]string{"client": "LoFP", "version": "11.5.0"})
	}
	if t.msspEnabled {
		t.sendMSSP()
	}
	if t.mccpEnabled {
		t.activateMCCP2()
	}

	log.Printf("telnet: negotiate done for %s (width=%d gmcp=%v mccp=%v mxp=%v msdp=%v mssp=%v)",
		t.conn.RemoteAddr(), t.width, t.gmcpEnabled, t.mccpEnabled, t.mxpEnabled, t.msdpEnabled, t.msspEnabled)
}

// enableEcho resumes server-side echo (normal input).
func (t *telnetConn) enableEcho() {}

// disableEcho suppresses echo for password entry.
func (t *telnetConn) disableEcho() {}

// --- GMCP Implementation ---

// sendGMCP sends a GMCP message: IAC SB GMCP "package json" IAC SE
func (t *telnetConn) sendGMCP(pkg string, data interface{}) {
	if !t.gmcpEnabled {
		return
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	payload := pkg + " " + string(jsonBytes)
	msg := []byte{iacByte, sbByte, optGMCP}
	msg = append(msg, []byte(payload)...)
	msg = append(msg, iacByte, seByte)
	// GMCP uses raw conn even during MCCP2 (it's a control channel)
	t.write(msg)
}

// sendGMCPString sends a GMCP message with a raw string payload (no JSON wrapping).
func (t *telnetConn) sendGMCPString(pkg string, payload string) {
	if !t.gmcpEnabled {
		return
	}
	msg := []byte{iacByte, sbByte, optGMCP}
	msg = append(msg, []byte(pkg+" "+payload)...)
	msg = append(msg, iacByte, seByte)
	t.write(msg)
}

// handleGMCPMessage processes an incoming GMCP message from the client.
func (t *telnetConn) handleGMCPMessage(msg string) {
	parts := strings.SplitN(msg, " ", 2)
	pkg := parts[0]

	switch pkg {
	case "Core.Supports.Set", "Core.Supports.Add":
		if len(parts) > 1 {
			var supports []string
			json.Unmarshal([]byte(parts[1]), &supports)
			for _, s := range supports {
				// Format: "Package 1" (name + version)
				name := strings.Fields(s)[0]
				t.gmcpSupported[name] = true
			}
		}
	case "Core.Supports.Remove":
		if len(parts) > 1 {
			var removes []string
			json.Unmarshal([]byte(parts[1]), &removes)
			for _, s := range removes {
				name := strings.Fields(s)[0]
				delete(t.gmcpSupported, name)
			}
		}
	}
}

// sendGMCPVitals sends Char.Vitals with current player state.
func (t *telnetConn) sendGMCPVitals(p *engine.Player) {
	if !t.gmcpEnabled {
		return
	}
	posNames := []string{"standing", "sitting", "lying down", "kneeling", "flying"}
	pos := "standing"
	if p.Position >= 0 && p.Position < len(posNames) {
		pos = posNames[p.Position]
	}
	t.sendGMCP("Char.Vitals", map[string]interface{}{
		"bp": p.BodyPoints, "maxbp": p.MaxBodyPoints,
		"mana": p.Mana, "maxmana": p.MaxMana,
		"psi": p.Psi, "maxpsi": p.MaxPsi,
		"fatigue": p.Fatigue, "maxfatigue": p.MaxFatigue,
		"position":   pos,
		"conditions": p.PromptIndicators(),
	})
}

// sendGMCPStatus sends Char.Status with name, race, level, currency.
func (t *telnetConn) sendGMCPStatus(p *engine.Player) {
	if !t.gmcpEnabled {
		return
	}
	gender := "Male"
	if p.Gender == 2 {
		gender = "Female"
	}
	t.sendGMCP("Char.Status", map[string]interface{}{
		"name":       p.FirstName,
		"fullname":   p.FullName(),
		"race":       p.RaceName(),
		"gender":     gender,
		"level":      p.Level,
		"experience": p.Experience,
		"gold":       p.Gold,
		"silver":     p.Silver,
		"copper":     p.Copper,
	})
}

// sendGMCPStats sends Char.Stats with attribute values.
func (t *telnetConn) sendGMCPStats(p *engine.Player) {
	if !t.gmcpEnabled {
		return
	}
	t.sendGMCP("Char.Stats", map[string]interface{}{
		"strength":     p.Strength,
		"agility":      p.Agility,
		"quickness":    p.Quickness,
		"constitution": p.Constitution,
		"perception":   p.Perception,
		"willpower":    p.Willpower,
		"empathy":      p.Empathy,
	})
}

// sendGMCPRoomInfo sends Room.Info for the automapper.
func (t *telnetConn) sendGMCPRoomInfo(result *engine.CommandResult, roomNumber int) {
	if !t.gmcpEnabled || result.RoomName == "" {
		return
	}
	area := fmt.Sprintf("Region %d", result.RoomRegion)
	terrain := result.RoomTerrain
	if terrain == "" {
		terrain = "indoor"
	}
	// Clean room name (strip brackets)
	name := strings.TrimPrefix(strings.TrimSuffix(result.RoomName, "]"), "[")
	t.sendGMCP("Room.Info", map[string]interface{}{
		"num":         roomNumber,
		"name":        name,
		"area":        area,
		"environment": terrain,
		"exits":       result.RoomExits,
	})
}

// sendGMCPCommChannel sends a Comm.Channel message.
func (t *telnetConn) sendGMCPCommChannel(channel, player, message string) {
	if !t.gmcpEnabled {
		return
	}
	t.sendGMCP("Comm.Channel", map[string]interface{}{
		"channel": channel,
		"player":  player,
		"message": message,
	})
}

// --- MCCP2 Implementation ---

// activateMCCP2 starts zlib compression for server→client output.
func (t *telnetConn) activateMCCP2() {
	if !t.mccpEnabled {
		return
	}
	// Send the MCCP2 subneg to signal compression start (uncompressed)
	t.conn.Write([]byte{iacByte, sbByte, optMCCP2, iacByte, seByte})
	// Everything after this is compressed
	t.compWriter = zlib.NewWriter(t.conn)
	t.compActivated = true
	log.Printf("telnet: MCCP2 compression activated for %s", t.conn.RemoteAddr())
}

// --- MSSP Implementation ---

// sendMSSP sends MUD Server Status Protocol data.
func (t *telnetConn) sendMSSP() {
	if !t.msspEnabled {
		return
	}
	playerCount := "0"
	if t.server != nil {
		t.server.mu.Lock()
		playerCount = fmt.Sprintf("%d", len(t.server.sessions))
		t.server.mu.Unlock()
	}

	var msg []byte
	msg = append(msg, iacByte, sbByte, optMSSP)
	addMSSPVar := func(name, val string) {
		msg = append(msg, msspVar)
		msg = append(msg, []byte(name)...)
		msg = append(msg, msspVal)
		msg = append(msg, []byte(val)...)
	}
	addMSSPVar("NAME", "Legends of Future Past")
	addMSSPVar("PLAYERS", playerCount)
	addMSSPVar("UPTIME", fmt.Sprintf("%d", time.Now().Unix()))
	addMSSPVar("CODEBASE", "LoFP (Go)")
	addMSSPVar("WEBSITE", "https://lofp.metavert.io")
	addMSSPVar("GENRE", "Fantasy")
	addMSSPVar("GAMEPLAY", "Hack and Slash")
	addMSSPVar("GAMEPLAY", "Roleplaying")
	addMSSPVar("STATUS", "Live")
	addMSSPVar("CREATED", "1992")
	addMSSPVar("MINIMUM AGE", "13")
	addMSSPVar("PORT", "4000")
	addMSSPVar("FAMILY", "Custom")
	msg = append(msg, iacByte, seByte)
	t.conn.Write(msg) // MSSP is sent before compression
}

// --- MSDP Implementation ---

// handleMSDPMessage processes incoming MSDP subnegotiation.
func (t *telnetConn) handleMSDPMessage(data []byte) {
	// Parse MSDP commands: VAR "REPORT" VAL "VARIABLE_NAME"
	i := 0
	for i < len(data) {
		if data[i] != msdpVar {
			i++
			continue
		}
		i++
		// Read variable name until VAL
		varStart := i
		for i < len(data) && data[i] != msdpVal {
			i++
		}
		if i >= len(data) {
			break
		}
		varName := string(data[varStart:i])
		i++ // skip VAL byte
		// Read value until next VAR or end
		valStart := i
		for i < len(data) && data[i] != msdpVar {
			i++
		}
		valName := string(data[valStart:i])

		if varName == "REPORT" {
			t.msdpReporting[valName] = true
		}
	}
}

// sendMSDPUpdate sends MSDP variable updates for subscribed variables.
func (t *telnetConn) sendMSDPUpdate(p *engine.Player, roomNumber int, roomName string) {
	if !t.msdpEnabled || len(t.msdpReporting) == 0 {
		return
	}

	vars := map[string]string{
		"CHARACTER_NAME": p.FirstName,
		"RACE":           p.RaceName(),
		"LEVEL":          fmt.Sprintf("%d", p.Level),
		"EXPERIENCE":     fmt.Sprintf("%d", p.Experience),
		"HEALTH":         fmt.Sprintf("%d", p.BodyPoints),
		"HEALTH_MAX":     fmt.Sprintf("%d", p.MaxBodyPoints),
		"MANA":           fmt.Sprintf("%d", p.Mana),
		"MANA_MAX":       fmt.Sprintf("%d", p.MaxMana),
		"MOVEMENT":       fmt.Sprintf("%d", p.Fatigue),
		"MOVEMENT_MAX":   fmt.Sprintf("%d", p.MaxFatigue),
		"ROOM":           fmt.Sprintf("%d", roomNumber),
		"ROOM_NAME":      roomName,
	}

	var msg []byte
	msg = append(msg, iacByte, sbByte, optMSDP)
	for name, val := range vars {
		if !t.msdpReporting[name] {
			continue
		}
		msg = append(msg, msdpVar)
		msg = append(msg, []byte(name)...)
		msg = append(msg, msdpVal)
		msg = append(msg, []byte(val)...)
	}

	// Room exits as MSDP table
	if t.msdpReporting["ROOM_EXITS"] && roomName != "" {
		msg = append(msg, msdpVar)
		msg = append(msg, []byte("ROOM_EXITS")...)
		msg = append(msg, msdpVal, msdpTableOpen)
		// We'd need exit data here — skip for now, just close table
		msg = append(msg, msdpTableClose)
	}

	msg = append(msg, iacByte, seByte)
	t.mu.Lock()
	t.write(msg)
	t.mu.Unlock()
}

// --- Rich Prompt ---

// buildPrompt creates a status-rich prompt line. If GMCP is active, use simple prompt.
func buildPrompt(p *engine.Player, gmcpEnabled bool) string {
	if gmcpEnabled {
		return "> "
	}

	colorForPct := func(cur, max int) string {
		if max <= 0 {
			return ansiGreen
		}
		pct := (cur * 100) / max
		if pct >= 75 {
			return ansiGreen
		}
		if pct >= 25 {
			return ansiYellow
		}
		return ansiRed + ansiBold
	}

	var b strings.Builder
	b.WriteString("[")
	b.WriteString(colorForPct(p.BodyPoints, p.MaxBodyPoints))
	b.WriteString(fmt.Sprintf("BP:%d/%d", p.BodyPoints, p.MaxBodyPoints))
	b.WriteString(ansiReset)

	if p.MaxMana > 0 {
		b.WriteString(" ")
		b.WriteString(colorForPct(p.Mana, p.MaxMana))
		b.WriteString(fmt.Sprintf("Mana:%d/%d", p.Mana, p.MaxMana))
		b.WriteString(ansiReset)
	}
	if p.MaxPsi > 0 {
		b.WriteString(" ")
		b.WriteString(colorForPct(p.Psi, p.MaxPsi))
		b.WriteString(fmt.Sprintf("Psi:%d/%d", p.Psi, p.MaxPsi))
		b.WriteString(ansiReset)
	}

	b.WriteString(" ")
	b.WriteString(colorForPct(p.Fatigue, p.MaxFatigue))
	b.WriteString(fmt.Sprintf("Ftg:%d/%d", p.Fatigue, p.MaxFatigue))
	b.WriteString(ansiReset)

	b.WriteString("]")

	indicators := p.PromptIndicators()
	if indicators != "" {
		b.WriteString(" " + ansiRed + indicators + ansiReset)
	}

	b.WriteString("> ")
	return b.String()
}

// --- Listeners ---

// ListenTelnet starts a TCP listener for telnet MUD client connections.
func (s *Server) ListenTelnet(addr string) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("telnet: failed to listen on %s: %v", addr, err)
		return
	}
	log.Printf("Telnet server listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("telnet: accept error: %v", err)
			continue
		}
		go s.handleTelnetConn(conn, false)
	}
}

// ListenTelnetTLS starts a TLS-encrypted telnet listener.
func (s *Server) ListenTelnetTLS(addr, certFile, keyFile string) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Printf("telnet-tls: failed to load cert/key: %v", err)
		return
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		log.Printf("telnet-tls: failed to listen on %s: %v", addr, err)
		return
	}
	log.Printf("Telnet TLS server listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("telnet-tls: accept error: %v", err)
			continue
		}
		go s.handleTelnetConn(conn, true)
	}
}

var serverStartTime = time.Now()

func (s *Server) handleTelnetConn(rawConn net.Conn, isTLS bool) {
	log.Printf("telnet: new connection from %s (tls=%v)", rawConn.RemoteAddr(), isTLS)
	tc := newTelnetConn(rawConn, isTLS)
	tc.server = s
	defer tc.Close()

	tc.negotiate()

	// Banner
	tc.writeLine("")
	tc.writeLine(ansiYellow + `    __                              __` + ansiReset)
	tc.writeLine(ansiYellow + `   / /  ___  ____  ___  ____  ____/ /____` + ansiReset)
	tc.writeLine(ansiYellow + `  / /  / _ \/ __ \/ _ \/ __ \/ __  / ___/` + ansiReset)
	tc.writeLine(ansiYellow + ` / /__/  __/ /_/ /  __/ / / / /_/ (__  )` + ansiReset)
	tc.writeLine(ansiYellow + `/_____|\___|\_,_/\___/_/ /_/\__,_/____/` + ansiReset)
	tc.writeLine(ansiYellow + `    ____  / __/` + ansiReset)
	tc.writeLine(ansiYellow + `   / __ \/ /_` + ansiReset)
	tc.writeLine(ansiYellow + `  / /_/ / __/` + ansiReset)
	tc.writeLine(ansiYellow + ` / .___/_/` + ansiReset)
	tc.writeLine(ansiYellow + `/_/   ____      __` + ansiReset)
	tc.writeLine(ansiYellow + `     / __/_  __/ /___  __________` + ansiReset)
	tc.writeLine(ansiYellow + `    / /_/ / / / __/ / / / ___/ _ \` + ansiReset)
	tc.writeLine(ansiYellow + `   / __/ /_/ / /_/ /_/ / /  /  __/` + ansiReset)
	tc.writeLine(ansiYellow + `  /_/  \__,_/\__/\__,_/_/   \___/` + ansiReset)
	tc.writeLine(ansiYellow + `    ____            __` + ansiReset)
	tc.writeLine(ansiYellow + `   / __ \____ _____/ /_` + ansiReset)
	tc.writeLine(ansiYellow + `  / /_/ / __ ` + "`" + `/ ___/ __/` + ansiReset)
	tc.writeLine(ansiYellow + ` / ____/ /_/ (__  ) /_` + ansiReset)
	tc.writeLine(ansiYellow + `/_/    \__,_/____/\__/` + ansiReset)
	tc.writeLine("")
	tc.writeLine("  The Shattered Realms of Andor await your return...")
	tc.writeLine("")

	// Security notice — we can't distinguish TLS-terminated (port 4001) from plain (4000)
	// since Fly.io forwards both to internal port 4000. Show a brief tip instead.
	tc.writeLine(ansiYellow + "  Tip: For encrypted connections, use SSL/TLS on port 4001" + ansiReset)
	tc.writeLine(ansiYellow + "  or SSH on port 4022." + ansiReset)
	tc.writeLine("")

	// Per-IP connection limit
	ip := extractIP(tc.RemoteAddr())
	s.connMu.Lock()
	if s.connsByIP[ip] >= 5 {
		s.connMu.Unlock()
		tc.writeLine("Too many connections from your address. Try again later.")
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

	// Authentication
	account, player, isBot := s.telnetAuthenticate(tc)
	if player == nil {
		tc.writeLine("Disconnected.")
		return
	}
	_ = isBot
	_ = account

	accountID := ""
	if account != nil {
		accountID = account.ID.Hex()
	}
	if player.AccountID != "" {
		accountID = player.AccountID
	}

	// Check for existing session and disconnect it
	s.mu.Lock()
	if oldSess, ok := s.sessions[player.FirstName]; ok {
		oldSess.Conn.Close()
		delete(s.sessions, player.FirstName)
	}
	s.mu.Unlock()

	session := &Session{
		Player:       player,
		Conn:         tc,
		lastActivity: time.Now(),
	}

	s.mu.Lock()
	s.sessions[player.FirstName] = session
	s.mu.Unlock()

	s.hub.RegisterPlayer(player.FirstName, player.FullName(), player.RoomNumber,
		player.Race, player.RaceName(), player.Position,
		player.IsGM, player.GMHat, player.GMHidden, player.GMInvis, player.Hidden)

	s.gamelog.Log(gamelog.EventLogin, player.FullName(), accountID,
		fmt.Sprintf("telnet from %s", ip), player.RoomNumber, "")
	s.gamelog.Log(gamelog.EventGameEnter, player.FullName(), accountID,
		fmt.Sprintf("telnet from %s", ip), player.RoomNumber, "")

	// GMCP login data
	tc.mu.Lock()
	tc.sendGMCPStatus(player)
	tc.sendGMCPStats(player)
	tc.sendGMCPVitals(player)
	tc.mu.Unlock()

	if !player.GMInvis && !player.GMHidden {
		s.broadcastGlobal(player.FirstName,
			[]string{fmt.Sprintf("** %s has just entered the Realms.", player.FirstName)})
		s.broadcastToRoom(player.RoomNumber, player.FirstName,
			[]string{fmt.Sprintf("%s materializes from the mists.", player.FirstName)})
	}

	tc.writeLine(fmt.Sprintf("\r\nWelcome back, %s the %s!", player.FullName(), player.RaceName()))
	tc.writeLine("")

	ctx := context.Background()
	enterResult := s.engine.EnterRoom(ctx, player)
	if enterResult != nil {
		enterResult.PlayerState = player
		s.sendResult(session, enterResult)
		// GMCP Room.Info on initial room entry
		tc.mu.Lock()
		tc.sendGMCPRoomInfo(enterResult, player.RoomNumber)
		tc.mu.Unlock()
	}

	// Command loop
	s.telnetCommandLoop(ctx, session, tc)

	// Cleanup — only if WE are still the active session for this character.
	// A reconnect may have already replaced us; don't nuke the new session.
	s.mu.Lock()
	currentSess, isActive := s.sessions[player.FirstName]
	isActive = isActive && currentSess == session
	if isActive {
		delete(s.sessions, player.FirstName)
	}
	s.mu.Unlock()

	if isActive {
		if !player.GMInvis && !player.GMHidden {
			if !session.quitSent {
				s.broadcastGlobal(player.FirstName,
					[]string{fmt.Sprintf("** %s has just left the Realms.", player.FirstName)})
			}
			s.broadcastToRoom(player.RoomNumber, player.FirstName,
				[]string{fmt.Sprintf("%s fades from the Realms.", player.FirstName)})
		}
		s.gamelog.Log(gamelog.EventGameExit, player.FullName(), accountID,
			fmt.Sprintf("telnet from %s", ip), player.RoomNumber, "")
		s.hub.UnregisterPlayer(player.FirstName)
	}

	tc.writeLine("Farewell from the Shattered Realms.")
}

func (s *Server) telnetAuthenticate(tc *telnetConn) (account *auth.Account, player *engine.Player, isBot bool) {
	ctx := context.Background()

	// Show active banner before login menu
	if s.engine != nil {
		if banner := s.engine.GetBanner(); banner != "" {
			tc.writeLine("")
			tc.writeLine(ansiYellow + "╔══════════════════════════════════════════════════════════════╗" + ansiReset)
			tc.writeLine(ansiYellow + "║  SERVER NOTICE                                               ║" + ansiReset)
			tc.writeLine(ansiYellow + "║  " + ansiBold + banner + ansiReset + ansiYellow + strings.Repeat(" ", max(0, 60-len(banner))) + "║" + ansiReset)
			tc.writeLine(ansiYellow + "╚══════════════════════════════════════════════════════════════╝" + ansiReset)
			tc.writeLine("")
		}
	}

	for attempts := 0; attempts < 3; attempts++ {
		tc.writeLine("Login options:")
		tc.writeLine("  1) Sign in with email/password")
		tc.writeLine("  2) Create a new account")
		tc.writeLine("  Q) Quit")
		tc.writeLine("")
		tc.writePrompt("Choice: ")

		choice, err := tc.readLine(2 * time.Minute)
		if err != nil {
			return nil, nil, false
		}
		choice = strings.TrimSpace(choice)

		switch strings.ToUpper(choice) {
		case "1":
			acct := s.telnetLoginByPassword(tc, ctx)
			if acct != nil {
				if !acct.EmailVerified && acct.GoogleID == "" {
					if !s.telnetVerifyPrompt(tc, ctx, acct) {
						continue
					}
					acct, _ = s.auth.GetAccount(ctx, acct.ID.Hex())
				}
				p := s.telnetCharacterSelect(tc, ctx, acct)
				if p != nil {
					return acct, p, false
				}
			}
		case "2":
			acct := s.telnetRegister(tc, ctx)
			if acct != nil {
				if !s.telnetVerifyPrompt(tc, ctx, acct) {
					continue
				}
				acct, _ = s.auth.GetAccount(ctx, acct.ID.Hex())
				p := s.telnetCharacterSelect(tc, ctx, acct)
				if p != nil {
					return acct, p, false
				}
			}
		case "Q", "QUIT":
			return nil, nil, false
		default:
			tc.writeLine(ansiRed + "Invalid choice." + ansiReset)
		}

		tc.writeLine(ansiRed + "Login failed. Try again." + ansiReset)
		tc.writeLine("")
	}

	tc.writeLine("Too many failed attempts.")
	return nil, nil, false
}

func (s *Server) telnetLoginByPassword(tc *telnetConn, ctx context.Context) *auth.Account {
	if s.auth == nil {
		tc.writeLine(ansiRed + "Authentication service is not configured." + ansiReset)
		return nil
	}
	tc.writePrompt("Email: ")
	email, err := tc.readLine(time.Minute)
	if err != nil {
		return nil
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return nil
	}

	tc.enterPasswordMode()
	tc.writePrompt("Password: ")
	password, err := tc.readLine(time.Minute)
	tc.exitPasswordMode()
	if err != nil {
		return nil
	}
	password = strings.TrimSpace(password)

	account, err := s.auth.LoginWithPassword(ctx, email, password)
	if err != nil {
		tc.writeLine(ansiRed + err.Error() + ansiReset)
		return nil
	}
	tc.writeLine(ansiGreen + fmt.Sprintf("Welcome, %s!", account.Name) + ansiReset)
	return account
}

func (s *Server) telnetRegister(tc *telnetConn, ctx context.Context) *auth.Account {
	if s.auth == nil {
		tc.writeLine(ansiRed + "Authentication service is not configured." + ansiReset)
		return nil
	}
	tc.writeLine("")
	tc.writeLine(ansiBoldCyan + "=== Create Account ===" + ansiReset)
	tc.writeLine("")

	tc.writePrompt("Display name: ")
	name, err := tc.readLine(time.Minute)
	if err != nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		tc.writeLine(ansiRed + "Name is required." + ansiReset)
		return nil
	}

	tc.writePrompt("Email: ")
	email, err := tc.readLine(time.Minute)
	if err != nil {
		return nil
	}
	email = strings.TrimSpace(email)
	if email == "" {
		tc.writeLine(ansiRed + "Email is required." + ansiReset)
		return nil
	}

	tc.writeLine(ansiYellow + "Password must be 10+ characters with uppercase, lowercase, digit, and special character." + ansiReset)
	tc.enterPasswordMode()
	tc.writePrompt("Password: ")
	password, err := tc.readLine(time.Minute)
	tc.exitPasswordMode()
	if err != nil {
		return nil
	}
	password = strings.TrimSpace(password)

	tc.enterPasswordMode()
	tc.writePrompt("Confirm password: ")
	confirm, err := tc.readLine(time.Minute)
	tc.exitPasswordMode()
	if err != nil {
		return nil
	}
	confirm = strings.TrimSpace(confirm)

	if password != confirm {
		tc.writeLine(ansiRed + "Passwords do not match." + ansiReset)
		return nil
	}

	account, verifyToken, verifyCode, err := s.auth.RegisterWithPassword(ctx, email, password, name)
	if err != nil {
		tc.writeLine(ansiRed + err.Error() + ansiReset)
		return nil
	}

	if s.email != nil && s.email.Enabled() {
		if err := s.email.SendVerification(account.Email, verifyToken, verifyCode); err != nil {
			log.Printf("Failed to send verification email to %s: %v", account.Email, err)
		}
		tc.writeLine(ansiGreen + "Account created! A verification email has been sent to " + email + "." + ansiReset)
	} else {
		tc.writeLine(ansiGreen + "Account created!" + ansiReset)
	}
	return account
}

func (s *Server) telnetCharacterSelect(tc *telnetConn, ctx context.Context, account *auth.Account) *engine.Player {
	accountID := account.ID.Hex()

	for {
		players, _ := s.engine.ListPlayersByAccount(ctx, accountID)

		tc.writeLine("")
		tc.writeLine(ansiBoldCyan + "=== Character Selection ===" + ansiReset)
		if len(players) > 0 {
			for i, p := range players {
				tc.writeLine(fmt.Sprintf("  %d) %s %s (Level %d %s)",
					i+1, p.FirstName, p.LastName, p.Level, engine.RaceNameByID(p.Race)))
			}
		} else {
			tc.writeLine("  No characters yet.")
		}
		tc.writeLine("  N) Create a new character")
		tc.writeLine("  Q) Quit")
		tc.writeLine("")
		tc.writePrompt("Choice: ")

		input, err := tc.readLine(2 * time.Minute)
		if err != nil {
			return nil
		}
		input = strings.TrimSpace(input)
		upper := strings.ToUpper(input)

		if upper == "Q" {
			return nil
		}
		if upper == "N" {
			p := s.telnetCreateCharacter(tc, ctx, accountID)
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
				tc.writeLine(ansiRed + "Failed to load character." + ansiReset)
				continue
			}
			return player
		}

		tc.writeLine(ansiRed + "Invalid choice." + ansiReset)
	}
}

func (s *Server) telnetCreateCharacter(tc *telnetConn, ctx context.Context, accountID string) *engine.Player {
	existing, _ := s.engine.ListPlayersByAccount(ctx, accountID)
	if len(existing) >= 8 {
		tc.writeLine(ansiRed + "You can have at most 8 characters." + ansiReset)
		return nil
	}

	tc.writeLine("")
	tc.writeLine(ansiBoldCyan + "=== Create Character ===" + ansiReset)
	tc.writeLine("")

	tc.writePrompt("First name: ")
	firstName, err := tc.readLine(time.Minute)
	if err != nil {
		return nil
	}
	firstName = strings.TrimSpace(firstName)

	tc.writePrompt("Last name: ")
	lastName, err := tc.readLine(time.Minute)
	if err != nil {
		return nil
	}
	lastName = strings.TrimSpace(lastName)

	tc.writeLine("")
	tc.writeLine("Races:")
	tc.writeLine("  1) Human      2) Aelfen     3) Highlander  4) Wolfling")
	tc.writeLine("  5) Murg       6) Drakin     7) Mechanoid   8) Ephemeral")
	tc.writePrompt("Race (1-8): ")
	raceStr, err := tc.readLine(time.Minute)
	if err != nil {
		return nil
	}
	var race int
	fmt.Sscanf(strings.TrimSpace(raceStr), "%d", &race)

	tc.writeLine("Gender: 1) Male  2) Female")
	tc.writePrompt("Gender (1-2): ")
	genderStr, err := tc.readLine(time.Minute)
	if err != nil {
		return nil
	}
	var gender int
	fmt.Sscanf(strings.TrimSpace(genderStr), "%d", &gender)

	if err := engine.ValidateCharacterInput(firstName, lastName, race, gender); err != nil {
		tc.writeLine(ansiRed + err.Error() + ansiReset)
		return nil
	}
	taken, _ := s.engine.IsFirstNameTaken(ctx, firstName)
	if taken {
		tc.writeLine(ansiRed + "That first name is already taken." + ansiReset)
		return nil
	}

	player := s.engine.CreateNewPlayer(ctx, firstName, lastName, race, gender, accountID)
	tc.writeLine(ansiGreen + fmt.Sprintf("Welcome to the Shattered Realms, %s the %s!", player.FullName(), engine.RaceNameByID(player.Race)) + ansiReset)
	return player
}

func (s *Server) telnetVerifyPrompt(tc *telnetConn, ctx context.Context, account *auth.Account) bool {
	tc.writeLine("")
	tc.writeLine(ansiYellow + "Your email address is not yet verified." + ansiReset)
	tc.writeLine("Check your email for a verification code, then enter it below.")
	tc.writeLine("(You can also press Enter to skip and come back later.)")
	tc.writeLine("")

	for i := 0; i < 3; i++ {
		tc.writePrompt("Verification code: ")
		code, err := tc.readLine(2 * time.Minute)
		if err != nil {
			return false
		}
		code = strings.TrimSpace(code)
		if code == "" {
			tc.writeLine("Skipped. You must verify your email before you can create or play a character.")
			return false
		}
		if err := s.auth.VerifyEmailByCode(ctx, code); err != nil {
			tc.writeLine(ansiRed + err.Error() + ansiReset)
			continue
		}
		tc.writeLine(ansiGreen + "Email verified!" + ansiReset)
		return true
	}
	tc.writeLine(ansiRed + "Too many failed attempts." + ansiReset)
	return false
}

func (s *Server) telnetAuthByAPIKey(tc *telnetConn, ctx context.Context) *engine.Player {
	tc.enterPasswordMode()
	tc.writePrompt("API key: ")
	key, err := tc.readLine(time.Minute)
	tc.exitPasswordMode()
	if err != nil {
		return nil
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}

	player, err := s.engine.ValidateAPIKey(ctx, key)
	if err != nil {
		tc.writeLine("Invalid API key.")
		return nil
	}
	return player
}

func (s *Server) telnetCommandLoop(ctx context.Context, session *Session, tc *telnetConn) {
	for {
		prompt := buildPrompt(session.Player, tc.gmcpEnabled)
		tc.writePrompt(prompt)
		input, err := tc.readLine(30 * time.Minute)
		if err != nil {
			return
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		session.lastActivity = time.Now()

		now := time.Now()
		if now.Sub(session.lastCmdTime) > time.Second {
			session.cmdCount = 0
			session.lastCmdTime = now
		}
		session.cmdCount++
		if session.cmdCount > 4 {
			tc.writeLine("[Slow down! Too many commands.]")
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
			tc.writeLine("[Slow down! Too many commands.]")
			continue
		}

		result := s.engine.ProcessCommand(ctx, session.Player, input)
		result.PlayerState = session.Player
		result.PromptIndicators = session.Player.PromptIndicators()
		s.sendResult(session, result)

		// GMCP updates after every command
		tc.mu.Lock()
		tc.sendGMCPVitals(session.Player)
		if result.RoomName != "" {
			tc.sendGMCPRoomInfo(result, session.Player.RoomNumber)
		}
		tc.mu.Unlock()

		// MSDP updates
		tc.sendMSDPUpdate(session.Player, session.Player.RoomNumber, result.RoomName)

		// Chat flood protection
		if len(result.RoomBroadcast) > 0 {
			now := time.Now()
			chatCutoff := now.Add(-10 * time.Second)
			var recent []time.Time
			for _, t := range session.chatTimes {
				if t.After(chatCutoff) {
					recent = append(recent, t)
				}
			}
			session.chatTimes = recent
			if len(session.chatTimes) >= 5 {
				tc.writeLine("[You are sending messages too quickly. Please wait.]")
				continue
			}
			session.chatTimes = append(session.chatTimes, now)
		}

		s.dispatchCommandResult(session, result)

		if result.Quit {
			return
		}
	}
}

// extractIP gets the IP address from a remote address string.
func extractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

// extractFirstWord returns the first word of a string (for MXP item commands).
func extractFirstWord(s string) string {
	// Strip article prefixes
	s = strings.TrimPrefix(s, "a ")
	s = strings.TrimPrefix(s, "an ")
	s = strings.TrimPrefix(s, "the ")
	s = strings.TrimPrefix(s, "some ")
	fields := strings.Fields(s)
	if len(fields) > 0 {
		return fields[0]
	}
	return s
}

// wordWrap wraps text to the given width.
func wordWrap(text string, width int) string {
	if width <= 0 {
		width = 80
	}
	var result strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= width {
			if result.Len() > 0 {
				result.WriteString("\r\n")
			}
			result.WriteString(line)
			continue
		}
		words := strings.Fields(line)
		lineLen := 0
		first := true
		for _, word := range words {
			if !first && lineLen+1+len(word) > width {
				result.WriteString("\r\n")
				lineLen = 0
				first = true
			}
			if !first {
				result.WriteByte(' ')
				lineLen++
			}
			result.WriteString(word)
			lineLen += len(word)
			first = false
		}
		if result.Len() > 0 {
			result.WriteString("\r\n")
		}
	}
	return result.String()
}
