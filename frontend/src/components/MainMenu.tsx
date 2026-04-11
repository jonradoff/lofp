import { useEffect, useState } from 'react'
import { GoogleLogin } from '@react-oauth/google'
import type { Character } from '../App'
import { useAuth } from '../App'

const RACE_NAMES: Record<number, string> = {
  1: 'Human', 2: 'Aelfen', 3: 'Highlander', 4: 'Wolfling',
  5: 'Murg', 6: 'Drakin', 7: 'Mechanoid', 8: 'Ephemeral',
}

const GOOGLE_ENABLED = !!import.meta.env.VITE_GOOGLE_CLIENT_ID

interface SavedPlayer {
  id: string
  firstName: string
  lastName: string
  race: number
  gender: number
  level: number
  roomNumber: number
  bodyPoints: number
  maxBodyPoints: number
  updatedAt: string
  apiKeyPrefix?: string
  isGM?: boolean
}

interface Props {
  onNewCharacter: () => void
  onSelectCharacter: (char: Character) => void
  onVersionNotes: () => void
}

export default function MainMenu({ onNewCharacter, onSelectCharacter, onVersionNotes }: Props) {
  const { user, login, loginWithPassword, register } = useAuth()
  const [players, setPlayers] = useState<SavedPlayer[]>([])
  const [loading, setLoading] = useState(true)
  const [backendUp, setBackendUp] = useState(true)
  const [loginError, setLoginError] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [apiKeyModal, setApiKeyModal] = useState<string | null>(null)
  const [generatedKey, setGeneratedKey] = useState<string | null>(null)
  const [keyAllowGM, setKeyAllowGM] = useState(false)
  const [authMode, setAuthMode] = useState<'choose' | 'login' | 'register' | 'forgot'>('choose')
  const [emailInput, setEmailInput] = useState('')
  const [passwordInput, setPasswordInput] = useState('')
  const [nameInput, setNameInput] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [forgotSent, setForgotSent] = useState(false)
  const [showMudInfo, setShowMudInfo] = useState(false)
  const [banner, setBanner] = useState('')

  const isLoggedIn = !!user

  useEffect(() => {
    fetch('/api/banner')
      .then(r => r.ok ? r.json() : { banner: '' })
      .then(data => setBanner(data.banner || ''))
      .catch(() => {})
  }, [])

  useEffect(() => {
    if (!isLoggedIn) {
      setLoading(false)
      return
    }
    const headers: Record<string, string> = {}
    if (user?.token) {
      headers['Authorization'] = `Bearer ${user.token}`
    }
    setLoading(true)
    setBackendUp(true)
    const tryFetch = () => {
      fetch('/api/characters', { headers })
        .then(r => {
          if (!r.ok) throw new Error('not ok')
          return r.json()
        })
        .then((data: SavedPlayer[]) => { setPlayers(data || []); setBackendUp(true); setLoading(false) })
        .catch(() => { setBackendUp(false); setTimeout(tryFetch, 3000) })
    }
    tryFetch()
  }, [user, isLoggedIn])

  const formatDate = (dateStr: string) => {
    if (!dateStr) return 'Unknown'
    const d = new Date(dateStr)
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  const handleDeleteCharacter = async (firstName: string) => {
    setDeleting(true)
    try {
      const headers: Record<string, string> = {}
      if (user?.token) headers['Authorization'] = `Bearer ${user.token}`
      const r = await fetch(`/api/characters/${firstName}`, { method: 'DELETE', headers })
      if (r.ok) {
        setPlayers(players.filter(p => p.firstName !== firstName))
      }
    } catch (_) { /* ignore */ }
    setDeleting(false)
    setDeleteConfirm(null)
  }

  const handleGenerateAPIKey = async (firstName: string) => {
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (user?.token) headers['Authorization'] = `Bearer ${user.token}`
      const r = await fetch(`/api/characters/${firstName}/apikey`, {
        method: 'POST', headers,
        body: JSON.stringify({ allowGM: keyAllowGM }),
      })
      const data = await r.json()
      if (data.key) {
        setGeneratedKey(data.key)
        // Update the player list so the icon changes immediately
        setPlayers(prev => prev.map(p =>
          p.firstName === firstName ? { ...p, apiKeyPrefix: data.key.substring(0, 13) } : p
        ))
      } else {
        alert(data.error || 'Failed to generate key')
      }
    } catch (_) { alert('Failed to generate key') }
  }

  const handleRevokeAPIKey = async (firstName: string) => {
    const headers: Record<string, string> = {}
    if (user?.token) headers['Authorization'] = `Bearer ${user.token}`
    await fetch(`/api/characters/${firstName}/apikey`, { method: 'DELETE', headers })
    setPlayers(prev => prev.map(p =>
      p.firstName === firstName ? { ...p, apiKeyPrefix: undefined } : p
    ))
    setApiKeyModal(null)
    setGeneratedKey(null)
  }

  const handleEmailLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setLoginError('')
    try {
      await loginWithPassword(emailInput, passwordInput)
    } catch (err) {
      setLoginError(err instanceof Error ? err.message : 'Login failed')
    }
    setSubmitting(false)
  }

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setLoginError('')
    try {
      await register(emailInput, passwordInput, nameInput)
    } catch (err) {
      setLoginError(err instanceof Error ? err.message : 'Registration failed')
    }
    setSubmitting(false)
  }

  const handleForgotPassword = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    setLoginError('')
    try {
      await fetch('/api/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: emailInput }),
      })
      setForgotSent(true)
    } catch {
      setLoginError('Failed to send reset email')
    }
    setSubmitting(false)
  }

  const handleGoogleSuccess = async (credentialResponse: { credential?: string }) => {
    if (!credentialResponse.credential) return
    try {
      setLoginError('')
      await login(credentialResponse.credential)
    } catch (err) {
      setLoginError(err instanceof Error ? err.message : 'Login failed. Please try again.')
    }
  }

  return (
    <div className="flex items-start justify-center h-full p-4 sm:p-8 overflow-y-auto">
      <div className="max-w-2xl w-full">
        {/* Title art — hidden on small screens, replaced with text title */}
        <div className="text-center mb-6 sm:mb-8">
          <pre className="hidden sm:inline-block text-amber-500 text-xs leading-tight font-mono text-left">
{`    __                              __
   / /  ___  ____  ___  ____  ____/ /____
  / /  / _ \\/ __ \\/ _ \\/ __ \\/ __  / ___/
 / /__/  __/ /_/ /  __/ / / / /_/ (__  )
/_____|\\___/\\__, /\\___/_/ /_/\\__,_/____/
    ____  / __/
   / __ \\/ /_
  / /_/ / __/
 / .___/_/
/_/   ____      __
     / __/_  __/ /___  __________
    / /_/ / / / __/ / / / ___/ _ \\
   / __/ /_/ / /_/ /_/ / /  /  __/
  /_/  \\__,_/\\__/\\__,_/_/   \\___/
    ____            __
   / __ \\____ _____/ /_
  / /_/ / __ \`/ ___/ __/
 / ____/ /_/ (__  ) /_
/_/    \\__,_/____/\\__/`}
          </pre>
          {/* Mobile title */}
          <div className="sm:hidden">
            <div className="text-amber-500 font-mono font-bold text-2xl tracking-widest">LEGENDS</div>
            <div className="text-amber-400 font-mono text-sm tracking-wider">of Future Past</div>
          </div>
          <p className="text-gray-500 font-mono text-sm mt-4">
            The Shattered Realms of Andor await your return...
          </p>
        </div>

        {/* Server banner */}
        {banner && (
          <div className="mb-6 border border-yellow-600 bg-yellow-950/40 rounded-lg px-5 py-4">
            <div className="text-yellow-400 font-mono text-xs font-bold uppercase tracking-widest mb-1">Server Notice</div>
            <div className="text-yellow-200 font-mono text-sm">{banner}</div>
          </div>
        )}

        {/* Login prompt */}
        {!isLoggedIn && (
          <div className="mb-8">
            {authMode === 'choose' && (
              <div className="text-center">
                <p className="text-gray-400 font-mono text-sm mb-4">Sign in to enter the Shattered Realms</p>
                <div className="flex flex-col items-center gap-3 max-w-xs mx-auto">
                  <button
                    onClick={() => setAuthMode('login')}
                    className="w-full py-3 bg-amber-700 hover:bg-amber-600 text-white font-mono text-sm rounded transition-colors"
                  >
                    Sign in with Email
                  </button>
                  <button
                    onClick={() => setAuthMode('register')}
                    className="w-full py-3 bg-[#222] hover:bg-[#333] text-gray-300 font-mono text-sm rounded border border-[#444] transition-colors"
                  >
                    Create Account
                  </button>
                  {GOOGLE_ENABLED && (
                    <>
                      <div className="flex items-center gap-3 w-full my-1">
                        <div className="flex-1 border-t border-[#333]" />
                        <span className="text-gray-600 font-mono text-xs">or</span>
                        <div className="flex-1 border-t border-[#333]" />
                      </div>
                      <GoogleLogin
                        onSuccess={handleGoogleSuccess}
                        onError={() => setLoginError('Login failed.')}
                        theme="filled_black"
                        size="large"
                        shape="rectangular"
                        text="signin_with"
                      />
                    </>
                  )}
                </div>
                {loginError && (
                  <p className="text-red-400 font-mono text-xs mt-3">{loginError}</p>
                )}
              </div>
            )}
            {authMode === 'login' && (
              <div className="max-w-xs mx-auto">
                <h2 className="text-amber-400 font-mono font-bold text-lg mb-4 text-center">Sign In</h2>
                <form onSubmit={handleEmailLogin} className="space-y-3">
                  <input
                    type="email" placeholder="Email" value={emailInput}
                    onChange={e => setEmailInput(e.target.value)}
                    className="w-full px-3 py-2 bg-[#111] border border-[#444] rounded font-mono text-sm text-gray-200 focus:border-amber-600 focus:outline-none"
                    autoFocus
                  />
                  <input
                    type="password" placeholder="Password" value={passwordInput}
                    onChange={e => setPasswordInput(e.target.value)}
                    className="w-full px-3 py-2 bg-[#111] border border-[#444] rounded font-mono text-sm text-gray-200 focus:border-amber-600 focus:outline-none"
                  />
                  {loginError && <p className="text-red-400 font-mono text-xs">{loginError}</p>}
                  <button type="submit" disabled={submitting}
                    className="w-full py-2 bg-amber-700 hover:bg-amber-600 text-white font-mono text-sm rounded disabled:opacity-50 transition-colors">
                    {submitting ? 'Signing in...' : 'Sign In'}
                  </button>
                </form>
                <div className="flex justify-between mt-3">
                  <button onClick={() => { setAuthMode('choose'); setLoginError('') }} className="text-gray-500 hover:text-gray-300 font-mono text-xs">Back</button>
                  <button onClick={() => { setAuthMode('forgot'); setLoginError('') }} className="text-amber-600 hover:text-amber-400 font-mono text-xs">Forgot password?</button>
                </div>
              </div>
            )}
            {authMode === 'register' && (
              <div className="max-w-xs mx-auto">
                <h2 className="text-amber-400 font-mono font-bold text-lg mb-4 text-center">Create Account</h2>
                <form onSubmit={handleRegister} className="space-y-3">
                  <input
                    type="text" placeholder="Display Name" value={nameInput}
                    onChange={e => setNameInput(e.target.value)}
                    className="w-full px-3 py-2 bg-[#111] border border-[#444] rounded font-mono text-sm text-gray-200 focus:border-amber-600 focus:outline-none"
                    autoFocus
                  />
                  <input
                    type="email" placeholder="Email" value={emailInput}
                    onChange={e => setEmailInput(e.target.value)}
                    className="w-full px-3 py-2 bg-[#111] border border-[#444] rounded font-mono text-sm text-gray-200 focus:border-amber-600 focus:outline-none"
                  />
                  <input
                    type="password" placeholder="Password (10+ chars, mixed case, digit, special)" value={passwordInput}
                    onChange={e => setPasswordInput(e.target.value)}
                    className="w-full px-3 py-2 bg-[#111] border border-[#444] rounded font-mono text-sm text-gray-200 focus:border-amber-600 focus:outline-none"
                  />
                  {loginError && <p className="text-red-400 font-mono text-xs">{loginError}</p>}
                  <button type="submit" disabled={submitting}
                    className="w-full py-2 bg-amber-700 hover:bg-amber-600 text-white font-mono text-sm rounded disabled:opacity-50 transition-colors">
                    {submitting ? 'Creating...' : 'Create Account'}
                  </button>
                </form>
                <div className="mt-3">
                  <button onClick={() => { setAuthMode('choose'); setLoginError('') }} className="text-gray-500 hover:text-gray-300 font-mono text-xs">Back</button>
                </div>
              </div>
            )}
            {authMode === 'forgot' && (
              <div className="max-w-xs mx-auto">
                <h2 className="text-amber-400 font-mono font-bold text-lg mb-4 text-center">Reset Password</h2>
                {forgotSent ? (
                  <p className="text-green-400 font-mono text-sm text-center">If an account exists with that email, a reset link has been sent.</p>
                ) : (
                  <form onSubmit={handleForgotPassword} className="space-y-3">
                    <input
                      type="email" placeholder="Email" value={emailInput}
                      onChange={e => setEmailInput(e.target.value)}
                      className="w-full px-3 py-2 bg-[#111] border border-[#444] rounded font-mono text-sm text-gray-200 focus:border-amber-600 focus:outline-none"
                      autoFocus
                    />
                    {loginError && <p className="text-red-400 font-mono text-xs">{loginError}</p>}
                    <button type="submit" disabled={submitting}
                      className="w-full py-2 bg-amber-700 hover:bg-amber-600 text-white font-mono text-sm rounded disabled:opacity-50 transition-colors">
                      {submitting ? 'Sending...' : 'Send Reset Link'}
                    </button>
                  </form>
                )}
                <div className="mt-3">
                  <button onClick={() => { setAuthMode('choose'); setLoginError(''); setForgotSent(false) }} className="text-gray-500 hover:text-gray-300 font-mono text-xs">Back</button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Saved characters — only show when authenticated */}
        {isLoggedIn && (
          <>
            {user?.account?.emailVerified === false && (
              <div className="bg-[#1a1a1a] border border-yellow-800 rounded-lg p-4 mb-4">
                <p className="text-yellow-400 font-mono text-sm font-bold mb-1">Email not verified</p>
                <p className="text-gray-400 font-mono text-xs mb-3">
                  Check your email for a verification link or code. You must verify your email before creating or playing characters.
                </p>
                <p className="text-gray-500 font-mono text-xs">
                  Click your name in the top-right corner to enter a verification code or resend the email.
                </p>
              </div>
            )}
            {loading ? (
              <div className="text-gray-500 font-mono text-center py-4">
                {backendUp ? 'Loading characters...' : (
                  <div className="flex items-center justify-center gap-2">
                    <div className="w-2 h-2 bg-yellow-500 rounded-full animate-pulse" />
                    <span>Connecting to server...</span>
                  </div>
                )}
              </div>
            ) : players.length > 0 ? (
              <div className="mb-6">
                <h2 className="text-gray-400 font-mono text-sm uppercase tracking-wider mb-3">
                  Your Characters
                </h2>
                <div className="space-y-2">
                  {players.map(p => (
                    <button
                      key={p.id}
                      onClick={() => {
                        if (user?.account?.emailVerified === false) return
                        onSelectCharacter({
                          firstName: p.firstName, lastName: p.lastName,
                          race: p.race, gender: p.gender,
                        })
                      }}
                      className={`w-full flex items-center justify-between bg-[#111] border border-[#333] rounded-lg px-5 py-4 text-left transition-colors group ${user?.account?.emailVerified === false ? 'opacity-50 cursor-not-allowed' : 'hover:border-amber-600 cursor-pointer'}`}
                    >
                      <div>
                        <div className="text-amber-400 font-mono font-bold text-lg group-hover:text-amber-300">
                          {p.firstName} {p.lastName}
                        </div>
                        <div className="text-gray-500 font-mono text-xs mt-1">
                          Level {p.level} {RACE_NAMES[p.race] || 'Unknown'} &middot; BP {p.bodyPoints}/{p.maxBodyPoints} &middot; Room #{p.roomNumber}
                        </div>
                      </div>
                      <div className="flex flex-col sm:flex-row items-end sm:items-center gap-1 sm:gap-3 shrink-0">
                        <div className="text-right hidden sm:block">
                          <div className="text-gray-600 font-mono text-xs">Last played</div>
                          <div className="text-gray-500 font-mono text-xs">{formatDate(p.updatedAt)}</div>
                        </div>
                        <div className="flex gap-1">
                          <button
                            onClick={(ev) => { ev.stopPropagation(); setApiKeyModal(p.firstName); setGeneratedKey(null); setKeyAllowGM(false) }}
                            className="text-gray-600 hover:text-amber-400 text-xs font-mono transition-colors px-2 py-2 min-h-[36px] rounded hover:bg-[#222]"
                            title={p.apiKeyPrefix ? "Manage Bot API Key" : "Generate Bot API Key"}
                          >
                            {p.apiKeyPrefix ? '🤖' : '⚙'}
                            <span className="hidden sm:inline"> Bot</span>
                          </button>
                          <button
                            onClick={(ev) => { ev.stopPropagation(); setDeleteConfirm(p.firstName) }}
                            className="text-gray-600 hover:text-red-400 text-xs font-mono transition-colors px-2 py-2 min-h-[36px] rounded hover:bg-[#222]"
                            title="Delete character"
                          >
                            ✕<span className="hidden sm:inline"> Delete</span>
                          </button>
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            ) : null}

            {/* New character button — only when backend is available and email verified */}
            {!loading && backendUp && user?.account?.emailVerified !== false && (
              <button
                onClick={onNewCharacter}
                className="w-full py-4 bg-[#111] border-2 border-dashed border-[#444] hover:border-amber-600 rounded-lg text-gray-400 hover:text-amber-400 font-mono text-lg transition-colors cursor-pointer"
              >
                + Create New Character
              </button>
            )}

            {players.length === 0 && !loading && backendUp && (
              <p className="text-gray-600 font-mono text-xs text-center mt-4">
                No saved characters found. Create one to begin your adventure!
              </p>
            )}
          </>
        )}
        <div className="mt-6 text-center">
          <button onClick={onVersionNotes} className="text-gray-600 hover:text-amber-400 text-xs font-mono">
            Version 11.2.3 &mdash; Version Notes
          </button>
          <span className="text-gray-700 mx-2">|</span>
          <a href="/manual" className="text-gray-600 hover:text-amber-400 text-xs font-mono">
            Manual
          </a>
          <span className="text-gray-700 mx-2">|</span>
          <a href="/api-docs" className="text-gray-600 hover:text-amber-400 text-xs font-mono">
            API Documentation
          </a>
          <span className="text-gray-700 mx-2">|</span>
          <button onClick={() => setShowMudInfo(true)} className="text-gray-600 hover:text-amber-400 text-xs font-mono">
            MUD Client Access
          </button>
          <br className="sm:hidden" />
          <span className="text-gray-700 mx-2 hidden sm:inline">|</span>
          <a href="https://www.metavert.io/privacy-policy" target="_blank" rel="noopener noreferrer" className="text-gray-600 hover:text-amber-400 text-xs font-mono">
            Privacy Policy
          </a>
          <span className="text-gray-700 mx-2">|</span>
          <a href="https://www.metavert.io/terms-of-service" target="_blank" rel="noopener noreferrer" className="text-gray-600 hover:text-amber-400 text-xs font-mono">
            Terms of Service
          </a>
        </div>

        {/* MUD Client info modal */}
        {showMudInfo && (
          <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4" onClick={() => setShowMudInfo(false)}>
            <div className="bg-[#1a1a1a] border border-amber-900 rounded-lg p-4 sm:p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
              <h3 className="text-amber-400 font-mono font-bold text-lg mb-3">Connect with a MUD Client</h3>
              <p className="text-gray-300 font-mono text-sm mb-4">
                You can play Legends of Future Past using any standard MUD client (Mudlet, TinTin++, etc.) or a terminal.
              </p>
              <div className="bg-black border border-[#444] rounded p-4 font-mono text-sm mb-4">
                <div className="text-gray-400 mb-1">Telnet:</div>
                <div className="text-green-400 select-all mb-1">telnet lofp.metavert.io 4000</div>
                <div className="text-gray-500 text-xs mb-3">Unencrypted. Use port 4001 with SSL/TLS for a secure connection.</div>
                <div className="text-gray-400 mb-1">SSH:</div>
                <div className="text-green-400 select-all mb-3">ssh -p 4022 play@lofp.metavert.io</div>
                <div className="text-gray-400 mb-1">Mudlet / other MUD clients:</div>
                <div className="text-gray-300 text-xs">Server: <span className="text-green-400 select-all">lofp.metavert.io</span></div>
                <div className="text-gray-300 text-xs">Port: <span className="text-green-400 select-all">4000</span> (plain) or <span className="text-green-400 select-all">4001</span> (SSL/TLS &mdash; check &ldquo;Secure&rdquo; in Mudlet)</div>
              </div>
              <p className="text-gray-500 font-mono text-xs mb-4">
                Log in with the same email and password you use on this site. You'll need to set a password in Account Settings if you only use Google login.
              </p>
              <div className="text-right">
                <button onClick={() => setShowMudInfo(false)}
                  className="px-4 py-2 bg-[#333] hover:bg-[#444] text-gray-300 font-mono text-sm rounded">
                  Close
                </button>
              </div>
            </div>
          </div>
        )}

        {/* API Key modal */}
        {apiKeyModal && (
          <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
            <div className="bg-[#1a1a1a] border border-amber-900 rounded-lg p-4 sm:p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto">
              <h3 className="text-amber-400 font-mono font-bold text-lg mb-3">Bot API Key — {apiKeyModal}</h3>
              {generatedKey ? (
                <div>
                  <p className="text-gray-300 font-mono text-sm mb-2">Your API key (copy it now — it won't be shown again):</p>
                  <div className="bg-black border border-[#444] rounded p-3 font-mono text-xs text-green-400 break-all select-all mb-4">
                    {generatedKey}
                  </div>
                  <p className="text-gray-500 font-mono text-xs mb-4">Use this key to connect a bot via WebSocket. See the /bots directory for examples.</p>
                  <button onClick={() => { setApiKeyModal(null); setGeneratedKey(null) }}
                    className="px-4 py-2 bg-[#333] hover:bg-[#444] text-gray-300 font-mono text-sm rounded">
                    Done
                  </button>
                </div>
              ) : (
                <div>
                  <p className="text-gray-300 font-mono text-sm mb-3">
                    Generate an API key to control this character via a bot program.
                    {players.find(p => p.firstName === apiKeyModal)?.apiKeyPrefix && (
                      <span className="text-yellow-400"> This character already has a key ({players.find(p => p.firstName === apiKeyModal)?.apiKeyPrefix}...). Generating a new one will replace it.</span>
                    )}
                  </p>
                  {players.find(p => p.firstName === apiKeyModal && (p as any).isGM) && (
                    <label className="flex items-center gap-2 mb-3 text-gray-400 text-xs font-mono cursor-pointer">
                      <input type="checkbox" checked={keyAllowGM} onChange={e => setKeyAllowGM(e.target.checked)} className="accent-amber-500" />
                      Allow bot to use GM commands
                    </label>
                  )}
                  <div className="flex gap-3 justify-end">
                    <button onClick={() => setApiKeyModal(null)}
                      className="px-4 py-2 bg-[#333] hover:bg-[#444] text-gray-300 font-mono text-sm rounded">
                      Cancel
                    </button>
                    {players.find(p => p.firstName === apiKeyModal)?.apiKeyPrefix && (
                      <button onClick={() => handleRevokeAPIKey(apiKeyModal)}
                        className="px-4 py-2 bg-red-900 hover:bg-red-800 text-red-200 font-mono text-sm rounded">
                        Revoke Key
                      </button>
                    )}
                    <button onClick={() => handleGenerateAPIKey(apiKeyModal)}
                      className="px-4 py-2 bg-amber-700 hover:bg-amber-600 text-white font-mono text-sm rounded">
                      Generate Key
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Delete confirmation modal */}
        {deleteConfirm && (
          <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
            <div className="bg-[#1a1a1a] border border-red-900 rounded-lg p-4 sm:p-6 w-full max-w-md">
              <h3 className="text-red-400 font-mono font-bold text-lg mb-3">Delete Character</h3>
              <p className="text-gray-300 font-mono text-sm mb-4">
                Are you sure you want to delete <span className="text-amber-400">{deleteConfirm}</span>?
                This action cannot be undone.
              </p>
              <div className="flex gap-3 justify-end">
                <button
                  onClick={() => setDeleteConfirm(null)}
                  className="px-4 py-2 bg-[#333] hover:bg-[#444] text-gray-300 font-mono text-sm rounded"
                >
                  Cancel
                </button>
                <button
                  onClick={() => handleDeleteCharacter(deleteConfirm)}
                  disabled={deleting}
                  className="px-4 py-2 bg-red-900 hover:bg-red-800 text-red-200 font-mono text-sm rounded disabled:opacity-50"
                >
                  {deleting ? 'Deleting...' : 'Delete Forever'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
