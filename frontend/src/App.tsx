import { useState, useEffect, useRef, createContext, useContext } from 'react'
import Terminal from './components/Terminal'
import CharacterCreate from './components/CharacterCreate'
import MainMenu from './components/MainMenu'
import AdminPanel from './components/AdminPanel'
import VersionNotes from './components/VersionNotes'
import APIDocs from './components/APIDocs'
import CaptureModal from './components/CaptureModal'
import CaptureViewer from './components/CaptureViewer'
import VerifyEmail from './components/VerifyEmail'
import ResetPassword from './components/ResetPassword'
import AccountModal from './components/AccountModal'
import Manual from './components/Manual'

type View = 'menu' | 'create' | 'play' | 'admin' | 'version' | 'capture_view' | 'api_docs' | 'verify_email' | 'reset_password'

// Check if URL points to a specific view
function initialViewFromURL(): View {
  const path = window.location.pathname
  if (path === '/version-notes' || path === '/version-notes/') return 'version'
  if (path === '/api-docs' || path === '/api-docs/') return 'api_docs'
  // /manual is handled as a modal overlay, not a view
  if (path === '/verify-email' || path === '/verify-email/') return 'verify_email'
  if (path === '/reset-password' || path === '/reset-password/') return 'reset_password'
  return 'menu'
}

export interface Character {
  firstName: string
  lastName: string
  race: number
  gender: number
}

export interface AuthUser {
  token: string
  account: {
    id: string
    email: string
    name: string
    picture: string
    isAdmin: boolean
    emailVerified?: boolean
  }
}

interface AuthContextType {
  user: AuthUser | null
  login: (credential: string) => Promise<void>
  loginWithPassword: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, name: string) => Promise<void>
  logout: () => void
}

export const AuthContext = createContext<AuthContextType>({
  user: null,
  login: async () => {},
  loginWithPassword: async () => {},
  register: async () => {},
  logout: () => {},
})

export function useAuth() {
  return useContext(AuthContext)
}

function App() {
  const [view, setViewRaw] = useState<View>(initialViewFromURL())
  const [showManual, setShowManual] = useState(window.location.pathname === '/manual' || window.location.pathname === '/manual/')
  const setView = (v: View) => {
    setViewRaw(v)
    if (v === 'version') {
      window.history.pushState({}, '', '/version-notes')
    } else if (v === 'api_docs') {
      window.history.pushState({}, '', '/api-docs')
    } else if (v === 'verify_email') {
      // keep URL as-is (has token param)
    } else if (v === 'reset_password') {
      // keep URL as-is (has token param)
    } else if (window.location.pathname !== '/') {
      window.history.pushState({}, '', '/')
    }
  }
  const [character, setCharacter] = useState<Character | null>(null)
  const [backendOnline, setBackendOnline] = useState(true)
  const [user, setUser] = useState<AuthUser | null>(null)
  const [authLoading, setAuthLoading] = useState(true)
  const [showCaptureModal, setShowCaptureModal] = useState(false)
  const [showAccountModal, setShowAccountModal] = useState(false)
  const [captureRecording, setCaptureRecording] = useState(false)
  const [viewCaptureId, setViewCaptureId] = useState('')
  const wsRef = useRef<WebSocket | null>(null)

  // Backend health check
  useEffect(() => {
    const check = () => {
      fetch('/healthz').then(r => setBackendOnline(r.ok)).catch(() => setBackendOnline(false))
    }
    check()
    const interval = setInterval(check, 5000)
    return () => clearInterval(interval)
  }, [])

  // Restore session from localStorage
  useEffect(() => {
    const stored = localStorage.getItem('lofp_auth')
    if (stored) {
      try {
        const parsed = JSON.parse(stored)
        fetch('/api/auth/me', {
          headers: { Authorization: `Bearer ${parsed.token}` },
        }).then(r => {
          if (r.ok) {
            return r.json().then((account: AuthUser['account']) => {
              const refreshed = { ...parsed, account }
              setUser(refreshed)
              localStorage.setItem('lofp_auth', JSON.stringify(refreshed))
            })
          } else if (r.status === 401 || r.status === 403) {
            localStorage.removeItem('lofp_auth')
          } else {
            setUser(parsed)
          }
        }).catch(() => {
          setUser(parsed)
        }).finally(() => setAuthLoading(false))
      } catch {
        localStorage.removeItem('lofp_auth')
        setAuthLoading(false)
      }
    } else {
      setAuthLoading(false)
    }
  }, [])

  const setAuthUser = (data: { token: string; account: AuthUser['account'] }) => {
    const authUser: AuthUser = { token: data.token, account: data.account }
    setUser(authUser)
    localStorage.setItem('lofp_auth', JSON.stringify(authUser))
  }

  const login = async (credential: string) => {
    const resp = await fetch('/api/auth/google', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ credential }),
    })
    if (!resp.ok) {
      const errData = await resp.json().catch(() => null)
      throw new Error(errData?.error || `Login failed (${resp.status})`)
    }
    setAuthUser(await resp.json())
  }

  const loginWithPassword = async (email: string, password: string) => {
    const resp = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    if (!resp.ok) {
      const errData = await resp.json().catch(() => null)
      throw new Error(errData?.error || `Login failed (${resp.status})`)
    }
    setAuthUser(await resp.json())
  }

  const register = async (email: string, password: string, name: string) => {
    const resp = await fetch('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, name }),
    })
    if (!resp.ok) {
      const errData = await resp.json().catch(() => null)
      throw new Error(errData?.error || `Registration failed (${resp.status})`)
    }
    setAuthUser(await resp.json())
  }

  const logout = () => {
    setUser(null)
    setCharacter(null)
    localStorage.removeItem('lofp_auth')
    setView('menu')
    document.title = 'Legends of Future Past'
  }

  const handleCharacterCreated = (char: Character) => {
    setCharacter(char)
    setView('play')
    document.title = `${char.firstName} | Legends of Future Past`
  }

  const handleSelectCharacter = (char: Character) => {
    setCharacter(char)
    setView('play')
    document.title = `${char.firstName} | Legends of Future Past`
  }

  if (authLoading) {
    return (
      <div className="h-screen flex items-center justify-center bg-[#0a0a0a]">
        <div className="text-gray-500 font-mono">Loading...</div>
      </div>
    )
  }

  return (
    <AuthContext.Provider value={{ user, login, loginWithPassword, register, logout }}>
      {/* h-dvh accounts for mobile browser chrome (address bar) shrinking the viewport */}
      <div className="h-dvh flex flex-col bg-[#0a0a0a]">
        <div className="flex items-center justify-between px-3 sm:px-4 py-2 bg-[#1a1a2e] border-b border-[#333] min-h-0">
          <h1
            className="text-amber-400 font-bold tracking-wider font-mono cursor-pointer shrink-0"
            onClick={() => setView('menu')}
          >
            {/* Full title on desktop, short on mobile */}
            <span className="hidden sm:inline text-lg">LEGENDS OF FUTURE PAST</span>
            <span className="sm:hidden text-base">LoFP</span>
          </h1>
          <div className="flex gap-1 sm:gap-2 items-center overflow-hidden">
            {!backendOnline && (
              <div className="hidden sm:flex items-center gap-1.5 px-2 py-1 text-xs font-mono text-yellow-400">
                <div className="w-2 h-2 bg-yellow-500 rounded-full animate-pulse" />
                <span className="hidden md:inline">Connecting...</span>
              </div>
            )}
            {!backendOnline && (
              <div className="sm:hidden w-2 h-2 bg-yellow-500 rounded-full animate-pulse" title="Connecting to server..." />
            )}
            <button
              onClick={() => setView('menu')}
              className={`px-2 sm:px-3 py-1 text-xs sm:text-sm rounded font-mono min-h-[36px] ${view === 'menu' ? 'bg-amber-700 text-white' : 'text-gray-400 hover:text-white'}`}
            >
              Menu
            </button>
            <button
              onClick={() => character ? setView('play') : null}
              className={`px-2 sm:px-3 py-1 text-xs sm:text-sm rounded font-mono min-h-[36px] ${view === 'play' ? 'bg-amber-700 text-white' : 'text-gray-400 hover:text-white'} ${!character ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
            >
              Play
            </button>
            {user?.account?.isAdmin && (
              <button
                onClick={() => setView('admin')}
                className={`hidden sm:block px-2 sm:px-3 py-1 text-xs sm:text-sm rounded font-mono min-h-[36px] ${view === 'admin' ? 'bg-amber-700 text-white' : 'text-gray-400 hover:text-white'}`}
              >
                Admin
              </button>
            )}
            {character && view === 'play' && (
              <button
                onClick={() => setShowCaptureModal(true)}
                className={`hidden sm:block px-2 sm:px-3 py-1 text-xs sm:text-sm rounded font-mono min-h-[36px] ${captureRecording ? 'bg-red-700 text-white animate-pulse' : 'text-gray-400 hover:text-white'}`}
              >
                {captureRecording ? '● Rec' : 'Capture'}
              </button>
            )}
            <button
              onClick={() => setShowManual(!showManual)}
              className={`hidden sm:block px-2 sm:px-3 py-1 text-xs sm:text-sm rounded font-mono min-h-[36px] ${showManual ? 'bg-amber-700 text-white' : 'text-gray-400 hover:text-white'}`}
            >
              Manual
            </button>
            {user && (
              <div className="flex items-center gap-1 sm:gap-2 ml-1 sm:ml-3 pl-1 sm:pl-3 border-l border-[#444]">
                {/* Avatar is always a tap target for account modal */}
                <button
                  onClick={() => setShowAccountModal(true)}
                  className="flex items-center gap-1 sm:gap-2 min-h-[36px] px-1"
                  title={user.account.name}
                >
                  <img src={user.account.picture || '/default-avatar.svg'} alt="" className="w-6 h-6 rounded-full" />
                  <span className="hidden sm:inline text-gray-400 hover:text-amber-400 text-xs font-mono underline decoration-dotted">
                    {user.account.name}
                    {user.account.emailVerified === false && <span className="text-yellow-500 ml-1" title="Email not verified">!</span>}
                  </span>
                  {user.account.emailVerified === false && (
                    <span className="sm:hidden text-yellow-500 text-xs" title="Email not verified">!</span>
                  )}
                </button>
                <button
                  onClick={logout}
                  className="hidden sm:block text-gray-500 hover:text-gray-300 text-xs font-mono min-h-[36px] px-1"
                >
                  Logout
                </button>
              </div>
            )}
          </div>
        </div>

        <div className="flex-1 overflow-hidden">
          {view === 'menu' && (
            <MainMenu
              onNewCharacter={() => setView('create')}
              onSelectCharacter={handleSelectCharacter}
              onVersionNotes={() => setView('version')}
            />
          )}
          {view === 'create' && <CharacterCreate onCreated={handleCharacterCreated} onOpenManual={() => setShowManual(true)} />}
          {view === 'play' && character && <Terminal character={character} onQuit={() => setView('menu')} wsRefOut={wsRef} onCaptureStatus={(recording, _id) => { setCaptureRecording(recording) }} />}
          {view === 'admin' && <AdminPanel />}
          {view === 'version' && <VersionNotes onBack={() => setView('menu')} />}
          {view === 'api_docs' && <APIDocs onBack={() => setView('menu')} />}
          {view === 'capture_view' && <CaptureViewer captureId={viewCaptureId} onBack={() => setView('play')} />}
          {view === 'verify_email' && <VerifyEmail onBack={() => setView('menu')} />}
          {view === 'reset_password' && <ResetPassword onBack={() => setView('menu')} />}
        </div>
        {showCaptureModal && (
          <CaptureModal
            wsRef={wsRef}
            recording={captureRecording}
            onClose={() => setShowCaptureModal(false)}
            onViewCapture={(id) => { setViewCaptureId(id); setShowCaptureModal(false); setView('capture_view') }}
          />
        )}
        {showAccountModal && (
          <AccountModal onClose={() => setShowAccountModal(false)} />
        )}
        {showManual && (
          <div className="fixed inset-0 z-50 bg-black/80 flex items-stretch justify-center">
            <div className="w-full max-w-6xl flex flex-col bg-[#0a0a0a] border-x border-[#333]">
              <Manual onBack={() => setShowManual(false)} />
            </div>
          </div>
        )}
      </div>
    </AuthContext.Provider>
  )
}

export default App
