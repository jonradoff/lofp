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

type View = 'menu' | 'create' | 'play' | 'admin' | 'version' | 'capture_view' | 'api_docs' | 'verify_email' | 'reset_password' | 'manual'

// Check if URL points to a specific view
function initialViewFromURL(): View {
  const path = window.location.pathname
  if (path === '/version-notes' || path === '/version-notes/') return 'version'
  if (path === '/api-docs' || path === '/api-docs/') return 'api_docs'
  if (path === '/manual' || path === '/manual/') return 'manual'
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
  const [prevView, setPrevView] = useState<View>('menu')
  const setView = (v: View) => {
    if (v === 'manual') {
      setPrevView(view) // remember where we came from
    }
    setViewRaw(v)
    if (v === 'version') {
      window.history.pushState({}, '', '/version-notes')
    } else if (v === 'api_docs') {
      window.history.pushState({}, '', '/api-docs')
    } else if (v === 'manual') {
      window.history.pushState({}, '', '/manual')
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
      <div className="h-screen flex flex-col bg-[#0a0a0a]">
        <div className="flex items-center justify-between px-4 py-2 bg-[#1a1a2e] border-b border-[#333]">
          <h1
            className="text-amber-400 font-bold text-lg tracking-wider font-mono cursor-pointer"
            onClick={() => setView('menu')}
          >
            LEGENDS OF FUTURE PAST
          </h1>
          <div className="flex gap-2 items-center">
            {!backendOnline && (
              <div className="flex items-center gap-1.5 px-3 py-1 text-xs font-mono text-yellow-400">
                <div className="w-2 h-2 bg-yellow-500 rounded-full animate-pulse" />
                Connecting to server...
              </div>
            )}
            <button
              onClick={() => setView('menu')}
              className={`px-3 py-1 text-sm rounded font-mono ${view === 'menu' ? 'bg-amber-700 text-white' : 'text-gray-400 hover:text-white'}`}
            >
              Menu
            </button>
            <button
              onClick={() => character ? setView('play') : null}
              className={`px-3 py-1 text-sm rounded font-mono ${view === 'play' ? 'bg-amber-700 text-white' : 'text-gray-400 hover:text-white'} ${!character ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
            >
              Play
            </button>
            {user?.account?.isAdmin && (
              <button
                onClick={() => setView('admin')}
                className={`px-3 py-1 text-sm rounded font-mono ${view === 'admin' ? 'bg-amber-700 text-white' : 'text-gray-400 hover:text-white'}`}
              >
                Admin
              </button>
            )}
            {character && view === 'play' && (
              <button
                onClick={() => setShowCaptureModal(true)}
                className={`px-3 py-1 text-sm rounded font-mono ${captureRecording ? 'bg-red-700 text-white animate-pulse' : 'text-gray-400 hover:text-white'}`}
              >
                {captureRecording ? 'Recording' : 'Capture'}
              </button>
            )}
            <button
              onClick={() => setView('manual')}
              className={`px-3 py-1 text-sm rounded font-mono ${view === 'manual' ? 'bg-amber-700 text-white' : 'text-gray-400 hover:text-white'}`}
            >
              Manual
            </button>
            {user && (
              <div className="flex items-center gap-2 ml-3 pl-3 border-l border-[#444]">
                <img src={user.account.picture || '/default-avatar.svg'} alt="" className="w-6 h-6 rounded-full" />
                <button
                  onClick={() => setShowAccountModal(true)}
                  className="text-gray-400 hover:text-amber-400 text-xs font-mono underline decoration-dotted cursor-pointer"
                >
                  {user.account.name}
                  {user.account.emailVerified === false && <span className="text-yellow-500 ml-1" title="Email not verified">!</span>}
                </button>
                <button
                  onClick={logout}
                  className="text-gray-500 hover:text-gray-300 text-xs font-mono"
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
          {view === 'create' && <CharacterCreate onCreated={handleCharacterCreated} />}
          {view === 'play' && character && <Terminal character={character} onQuit={() => setView('menu')} wsRefOut={wsRef} onCaptureStatus={(recording, _id) => { setCaptureRecording(recording) }} />}
          {view === 'admin' && <AdminPanel />}
          {view === 'version' && <VersionNotes onBack={() => setView('menu')} />}
          {view === 'api_docs' && <APIDocs onBack={() => setView('menu')} />}
          {view === 'manual' && <Manual onBack={() => setView(prevView)} />}
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
      </div>
    </AuthContext.Provider>
  )
}

export default App
