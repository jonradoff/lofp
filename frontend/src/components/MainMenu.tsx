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
  const { user, login } = useAuth()
  const [players, setPlayers] = useState<SavedPlayer[]>([])
  const [loading, setLoading] = useState(true)
  const [backendUp, setBackendUp] = useState(true)
  const [loginError, setLoginError] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [apiKeyModal, setApiKeyModal] = useState<string | null>(null) // firstName for API key
  const [generatedKey, setGeneratedKey] = useState<string | null>(null)
  const [keyAllowGM, setKeyAllowGM] = useState(false)

  const isLoggedIn = !!user

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
    <div className="flex items-start justify-center h-full p-8 overflow-y-auto">
      <div className="max-w-2xl w-full">
        {/* Title art */}
        <div className="text-center mb-8">
          <pre className="text-amber-500 text-xs leading-tight font-mono inline-block text-left">
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
          <p className="text-gray-500 font-mono text-sm mt-4">
            The Shattered Realms of Andor await your return...
          </p>
        </div>

        {/* Login prompt */}
        {!isLoggedIn && (
          <div className="text-center mb-8">
            {GOOGLE_ENABLED ? (
              <>
                <p className="text-gray-400 font-mono text-sm mb-4">Sign in to enter the Shattered Realms</p>
                <div className="flex justify-center">
                  <GoogleLogin
                    onSuccess={handleGoogleSuccess}
                    onError={() => setLoginError('Login failed.')}
                    theme="filled_black"
                    size="large"
                    shape="rectangular"
                    text="signin_with"
                  />
                </div>
                {loginError && (
                  <p className="text-red-400 font-mono text-xs mt-3">{loginError}</p>
                )}
              </>
            ) : (
              <div className="bg-[#1a1a1a] border border-red-900 rounded-lg p-6">
                <p className="text-red-400 font-mono text-sm font-bold mb-2">Authentication Not Configured</p>
                <p className="text-gray-400 font-mono text-xs leading-relaxed">
                  Google login is required but VITE_GOOGLE_CLIENT_ID is not set.
                  <br />
                  Set the environment variable and restart the frontend server.
                </p>
              </div>
            )}
          </div>
        )}

        {/* Saved characters — only show when authenticated */}
        {isLoggedIn && (
          <>
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
                      onClick={() => onSelectCharacter({
                        firstName: p.firstName, lastName: p.lastName,
                        race: p.race, gender: p.gender,
                      })}
                      className="w-full flex items-center justify-between bg-[#111] border border-[#333] hover:border-amber-600 rounded-lg px-5 py-4 text-left transition-colors group cursor-pointer"
                    >
                      <div>
                        <div className="text-amber-400 font-mono font-bold text-lg group-hover:text-amber-300">
                          {p.firstName} {p.lastName}
                        </div>
                        <div className="text-gray-500 font-mono text-xs mt-1">
                          Level {p.level} {RACE_NAMES[p.race] || 'Unknown'} &middot; BP {p.bodyPoints}/{p.maxBodyPoints} &middot; Room #{p.roomNumber}
                        </div>
                      </div>
                      <div className="text-right flex items-center gap-3">
                        <div>
                          <div className="text-gray-600 font-mono text-xs">
                            Last played
                          </div>
                          <div className="text-gray-500 font-mono text-xs">
                            {formatDate(p.updatedAt)}
                          </div>
                        </div>
                        <button
                          onClick={(ev) => { ev.stopPropagation(); setApiKeyModal(p.firstName); setGeneratedKey(null); setKeyAllowGM(false) }}
                          className="text-gray-600 hover:text-amber-400 text-sm font-mono transition-colors px-2 py-2 rounded hover:bg-[#222]"
                          title={p.apiKeyPrefix ? "Manage Bot API Key" : "Generate Bot API Key"}
                        >
                          {p.apiKeyPrefix ? '🤖 Bot' : '⚙ Bot'}
                        </button>
                        <button
                          onClick={(ev) => { ev.stopPropagation(); setDeleteConfirm(p.firstName) }}
                          className="text-gray-600 hover:text-red-400 text-sm font-mono transition-colors px-2 py-2 rounded hover:bg-[#222]"
                          title="Delete character"
                        >
                          ✕ Delete
                        </button>
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            ) : null}

            {/* New character button — only when backend is available */}
            {!loading && backendUp && (
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
            Version 10.0.3 &mdash; Version Notes
          </button>
          <span className="text-gray-700 mx-2">|</span>
          <a href="/api-docs" className="text-gray-600 hover:text-amber-400 text-xs font-mono">
            API Documentation
          </a>
        </div>

        {/* API Key modal */}
        {apiKeyModal && (
          <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
            <div className="bg-[#1a1a1a] border border-amber-900 rounded-lg p-6 max-w-lg">
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
          <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
            <div className="bg-[#1a1a1a] border border-red-900 rounded-lg p-6 max-w-md">
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
