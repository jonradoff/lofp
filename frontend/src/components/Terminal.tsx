import { useEffect, useRef, useState, useCallback, useContext } from 'react'
import { AuthContext } from '../App'

interface Character {
  firstName: string
  lastName: string
  race: number
  gender: number
}

interface PlayerState {
  bodyPoints: number
  maxBodyPoints: number
  fatigue: number
  maxFatigue: number
  mana: number
  maxMana: number
  psi: number
  maxPsi: number
  roomNumber: number
}

interface CommandResult {
  messages?: string[]
  playerState?: PlayerState
  roomName?: string
  promptIndicators?: string
  error?: string
  quit?: boolean
}

interface Props {
  character: Character
  onQuit: () => void
  wsRefOut?: React.RefObject<WebSocket | null>
  onCaptureStatus?: (recording: boolean, id: string) => void
}

// Detect touch/mobile device — used to avoid auto-opening keyboard on load
const isTouchDevice = () => typeof window !== 'undefined' && ('ontouchstart' in window || navigator.maxTouchPoints > 0)

export default function Terminal({ character, onQuit, wsRefOut, onCaptureStatus }: Props) {
  const { user } = useContext(AuthContext)
  const [lines, setLines] = useState<Array<{ text: string; type: 'output' | 'input' | 'system' | 'room' | 'item' | 'error' | 'broadcast' }>>([])
  const [input, setInput] = useState('')
  const [connected, setConnected] = useState(false)
  const [playerState, setPlayerState] = useState<PlayerState | null>(null)
  const [roomName, setRoomName] = useState<string>('')
  const [promptIndicators, setPromptIndicators] = useState<string>('')
  const [history, setHistory] = useState<string[]>([])
  const [historyIdx, setHistoryIdx] = useState(-1)
  const wsRef = useRef<WebSocket | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const addLines = useCallback((msgs: string[], type: 'output' | 'system' | 'room' | 'item' | 'error' | 'broadcast' = 'output') => {
    setLines(prev => [...prev, ...msgs.map(text => ({ text, type }))])
  }, [])

  useEffect(() => {
    let retryDelay = 1000
    let intentionalClose = false
    let quit = false

    function connect() {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const ws = new WebSocket(`${protocol}//${window.location.host}/ws/game`)
      wsRef.current = ws
      if (wsRefOut) (wsRefOut as React.MutableRefObject<WebSocket | null>).current = ws

      ws.onopen = () => {
        setConnected(true)
        retryDelay = 1000 // reset backoff on successful connect
        if (user?.token) {
          ws.send(JSON.stringify({ type: 'auth', data: { token: user.token } }))
        }
        ws.send(JSON.stringify({ type: 'create_character', data: character }))
      }

      ws.onmessage = (event) => {
        const msg = JSON.parse(event.data)
        if (msg.type === 'result') {
          const result: CommandResult = typeof msg.data === 'string' ? JSON.parse(msg.data) : msg.data
          if (result.error) addLines([result.error], 'error')
          if (result.messages) {
            result.messages.forEach(m => {
              if (m.startsWith('[') && m.endsWith(']')) addLines([m], 'room')
              else if (m.startsWith('You see ')) addLines([m], 'item')
              else if (m.startsWith('Obvious exits:')) addLines([m], 'system')
              else addLines([m], 'output')
            })
          }
          if (result.roomName) setRoomName(result.roomName)
          setPromptIndicators(result.promptIndicators || '')
          if (result.playerState) setPlayerState(result.playerState)
          if (result.quit) {
            quit = true
            intentionalClose = true
            wsRef.current?.close()
            setTimeout(() => onQuit(), 1500)
          }
        } else if (msg.type === 'broadcast') {
          const data = typeof msg.data === 'string' ? JSON.parse(msg.data) : msg.data
          if (data.messages) {
            data.messages.forEach((m: string) => addLines([m], 'broadcast'))
          }
        } else if (msg.type === 'capture_status') {
          const data = typeof msg.data === 'string' ? JSON.parse(msg.data) : msg.data
          if (onCaptureStatus) onCaptureStatus(data.recording, data.id || '')
        }
      }

      ws.onclose = () => {
        setConnected(false)
        if (quit || intentionalClose) return
        addLines([`Connection lost. Reconnecting in ${retryDelay / 1000}s...`], 'system')
        setTimeout(() => {
          if (!intentionalClose) connect()
        }, retryDelay)
        retryDelay = Math.min(retryDelay * 2, 30000) // exponential backoff, max 30s
      }

      ws.onerror = () => {} // onclose handles reconnection
    }

    connect()
    return () => { intentionalClose = true; wsRef.current?.close() }
  }, [character, addLines])

  useEffect(() => {
    scrollRef.current?.scrollTo(0, scrollRef.current.scrollHeight)
  }, [lines])

  // Only auto-focus on non-touch devices — on mobile this would open the keyboard immediately
  useEffect(() => {
    if (!isTouchDevice()) {
      inputRef.current?.focus()
    }
  }, [])

  // Keep input bar visible when mobile keyboard opens by scrolling to bottom
  useEffect(() => {
    if (!isTouchDevice() || !window.visualViewport) return
    const vv = window.visualViewport
    const onResize = () => {
      // When keyboard appears, scroll terminal output to bottom
      scrollRef.current?.scrollTo(0, scrollRef.current.scrollHeight)
    }
    vv.addEventListener('resize', onResize)
    return () => vv.removeEventListener('resize', onResize)
  }, [])

  const sendCommand = (cmd: string) => {
    if (!cmd.trim() || !wsRef.current) return
    setLines(prev => [...prev, { text: `> ${cmd}`, type: 'input' }])
    setHistory(prev => [cmd, ...prev])
    setHistoryIdx(-1)
    wsRef.current.send(JSON.stringify({
      type: 'command',
      data: { input: cmd },
    }))
    setInput('')
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      sendCommand(input)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      if (history.length > 0) {
        const newIdx = Math.min(historyIdx + 1, history.length - 1)
        setHistoryIdx(newIdx)
        setInput(history[newIdx])
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      if (historyIdx > 0) {
        const newIdx = historyIdx - 1
        setHistoryIdx(newIdx)
        setInput(history[newIdx])
      } else {
        setHistoryIdx(-1)
        setInput('')
      }
    }
  }

  const colorMap = {
    output: 'text-gray-300',
    input: 'text-cyan-400',
    system: 'text-yellow-500',
    room: 'text-amber-400 font-bold',
    item: 'text-green-400',
    error: 'text-red-400',
    broadcast: 'text-sky-300',
  }

  return (
    <div className="flex flex-col h-full" onClick={() => {
      // Only refocus input if user clicked empty space (not selecting text) and not on touch device
      if (isTouchDevice()) return
      const selection = window.getSelection()
      if (!selection || selection.toString().length === 0) {
        inputRef.current?.focus()
      }
    }}>
      {/* Status bar — wraps on narrow screens */}
      {playerState && (
        <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 px-3 py-1.5 bg-[#111] border-b border-[#333] font-mono text-xs min-h-0">
          <span className="text-red-400">BP <span className="text-gray-300">{playerState.bodyPoints}/{playerState.maxBodyPoints}</span></span>
          <span className="text-yellow-400">FT <span className="text-gray-300">{playerState.fatigue}/{playerState.maxFatigue}</span></span>
          <span className="text-blue-400">MP <span className="text-gray-300">{playerState.mana}/{playerState.maxMana}</span></span>
          <span className="text-purple-400">PSI <span className="text-gray-300">{playerState.psi}/{playerState.maxPsi}</span></span>
          <span className="text-gray-500 truncate max-w-[140px] sm:max-w-none">{roomName || `Room ${playerState.roomNumber}`}</span>
          <span className={`ml-auto ${connected ? 'text-green-500' : 'text-red-500'}`}>
            {connected ? '●' : '○'}
          </span>
        </div>
      )}

      {/* Terminal output */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto p-3 sm:p-4 font-mono text-xs sm:text-sm leading-relaxed select-text cursor-text">
        {lines.map((line, i) => (
          <div key={i} className={`${colorMap[line.type]} whitespace-pre-wrap`}>
            {line.text || '\u00A0'}
          </div>
        ))}
      </div>

      {/* Input bar */}
      <div className="flex items-center px-3 py-2 sm:px-4 bg-[#111] border-t border-[#333]">
        {promptIndicators && <span className="text-red-400 font-mono mr-1">{promptIndicators}</span>}
        <span className="text-amber-400 font-mono mr-2">&gt;</span>
        <input
          ref={inputRef}
          type="text"
          inputMode="text"
          enterKeyHint="send"
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          className="flex-1 bg-transparent text-gray-200 font-mono text-base sm:text-sm focus:outline-none min-w-0"
          placeholder="Enter command..."
        />
        {/* Send button — visible only on touch devices */}
        <button
          className="sm:hidden ml-2 px-3 py-1 bg-amber-700 text-white font-mono text-sm rounded active:bg-amber-600"
          onPointerDown={e => { e.preventDefault(); sendCommand(input) }}
        >
          ↵
        </button>
      </div>
    </div>
  )
}
