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
  const [loginError, setLoginError] = useState('')

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
    fetch('/api/characters', { headers })
      .then(r => r.json())
      .then((data: SavedPlayer[]) => setPlayers(data || []))
      .catch(() => setPlayers([]))
      .finally(() => setLoading(false))
  }, [user, isLoggedIn])

  const formatDate = (dateStr: string) => {
    if (!dateStr) return 'Unknown'
    const d = new Date(dateStr)
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
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
              <div className="text-gray-500 font-mono text-center">Loading characters...</div>
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
                          Level {p.level} {RACE_NAMES[p.race] || 'Unknown'} &middot; HP {p.bodyPoints}/{p.maxBodyPoints} &middot; Room #{p.roomNumber}
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="text-gray-600 font-mono text-xs">
                          Last played
                        </div>
                        <div className="text-gray-500 font-mono text-xs">
                          {formatDate(p.updatedAt)}
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            ) : null}

            {/* New character button */}
            <button
              onClick={onNewCharacter}
              className="w-full py-4 bg-[#111] border-2 border-dashed border-[#444] hover:border-amber-600 rounded-lg text-gray-400 hover:text-amber-400 font-mono text-lg transition-colors cursor-pointer"
            >
              + Create New Character
            </button>

            {players.length === 0 && !loading && (
              <p className="text-gray-600 font-mono text-xs text-center mt-4">
                No saved characters found. Create one to begin your adventure!
              </p>
            )}
          </>
        )}
        <div className="mt-6 text-center">
          <button onClick={onVersionNotes} className="text-gray-600 hover:text-amber-400 text-xs font-mono">
            Version 0.91 &mdash; Version Notes
          </button>
        </div>
      </div>
    </div>
  )
}
