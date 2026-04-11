import { useEffect, useState, useRef } from 'react'
import { useAuth } from '../App'

interface Stats {
  rooms: number
  items: number
  monsters: number
  nouns: number
  adjs: number
  sessions: number
}

interface RoomSummary {
  number: number
  name: string
  terrain: string
  exits: number
  file: string
}

interface RoomExit {
  room: number
  roomName: string
}

interface RoomItemDef {
  name: string
  type: string
  weight: number
  volume: number
  substance: string
  article: string
  wornSlot?: string
  container?: string
  flags?: string[]
  sourceFile: string
}

interface RoomItemDetail {
  ref: number
  archetype: number
  itemDef: RoomItemDef | null
  adj1?: number
  adj1Name?: string
  adj2?: number
  adj2Name?: string
  adj3?: number
  adj3Name?: string
  val1?: number
  val2?: number
  val3?: number
  val4?: number
  val5?: number
  state?: string
  extend?: string
  putIn?: number
  isPut?: boolean
}

interface RoomDetail {
  number: number
  name: string
  description: string
  exits: Record<string, RoomExit>
  items: RoomItemDetail[]
  terrain: string
  lighting: string
  monsterGroup: number
  modifiers?: string[]
  scripts: unknown[]
  sourceFile: string
}

interface PlayerSummary {
  id: string
  firstName: string
  lastName: string
  accountId?: string
  race: number
  gender: number
  level: number
  experience: number
  roomNumber: number
  bodyPoints: number
  maxBodyPoints: number
  isGM: boolean
  dead: boolean
  updatedAt?: string
  createdAt?: string
}

interface PlayerDetail extends PlayerSummary {
  strength: number
  agility: number
  quickness: number
  constitution: number
  perception: number
  willpower: number
  empathy: number
  fatigue: number
  maxFatigue: number
  mana: number
  maxMana: number
  psi: number
  maxPsi: number
  gold: number
  silver: number
  copper: number
  briefMode: boolean
  promptMode: boolean
  hidden: boolean
  bleeding: boolean
  stunned: boolean
  diseased: boolean
  poisoned: boolean
  position: number
  inventory: Array<{ archetype: number }>
  skills: Record<string, number>
}

interface AccountSummary {
  id: string
  googleId: string
  email: string
  name: string
  picture: string
  isAdmin: boolean
  createdAt: string
  updatedAt: string
}

interface AccountDetail {
  account: AccountSummary
  characters: PlayerSummary[]
}

interface ItemSummary {
  number: number
  name: string
  type: string
  weight: number
  substance: string
  sourceFile: string
}

interface ItemDetail {
  number: number
  nameId: number
  resolvedName: string
  type: string
  weight: number
  volume: number
  substance: string
  article: string
  parameter1: number
  parameter2: number
  parameter3: number
  container?: string
  interior?: number
  wornSlot?: string
  flags: string[]
  scripts?: unknown[]
  sourceFile: string
}

interface MonsterSummary {
  number: number
  name: string
  adjective: number
  bodyType: string
  body: number
  attack1: number
  defense: number
  unique: boolean
  sourceFile: string
}

interface MonsterDetail {
  number: number
  name: string
  adjective: number
  adjName: string
  description: string
  bodyType: string
  body: number
  attack1: number
  attack2: number
  defense: number
  strategy: number
  treasure: number
  speed: number
  armor: number
  race: number
  gender: number
  unique: boolean
  scripts?: unknown[]
  sourceFile: string
}

const RaceNames: Record<number, string> = {
  1: 'Human', 2: 'Aelfen', 3: 'Highlander', 4: 'Wolfling',
  5: 'Murg', 6: 'Drakin', 7: 'Mechanoid', 8: 'Ephemeral',
}

const ItemTypeNames: Record<string, string> = {
  AMMO: 'Ammunition', ARMOR: 'Armor', BITE_WEAPON: 'Bite Weapon',
  BOW_WEAPON: 'Bow Weapon', CLAW_WEAPON: 'Claw Weapon', CRUSH_WEAPON: 'Crush Weapon',
  DRAKIN_CRUSH: 'Drakin Crush', DRAKIN_POLE: 'Drakin Pole',
  DRAKIN_SLASH: 'Drakin Slash', DRAKIN_THROWN: 'Drakin Thrown',
  FOOD: 'Food', HANDGUN: 'Handgun', KEY: 'Key',
  LIQCONTAINER: 'Liquid Container', LIQUID: 'Liquid', LOCKPICK: 'Lockpick',
  MINETOOL: 'Mining Tool', MISC: 'Miscellaneous', MONEY: 'Money',
  POLE_WEAPON: 'Pole Weapon', POLETHROWN: 'Pole/Thrown',
  PORTAL_THROUGH: 'Portal (Through)', PORTAL_CLIMB: 'Portal (Climb)',
  PORTAL_UP: 'Portal (Up)', PORTAL_DOWN: 'Portal (Down)', PORTAL: 'Portal',
  PUNCTURE_WEAPON: 'Puncture Weapon', RIFLE: 'Rifle', SCROLL: 'Scroll',
  SHIELD: 'Shield', SLASH_WEAPON: 'Slash Weapon', STABTHROWN: 'Stab/Thrown',
  THROWN_WEAPON: 'Thrown Weapon', TRAP: 'Trap', TWOHAND_WEAPON: 'Two-Handed Weapon',
  ORE: 'Ore',
}

const WornSlotNames: Record<string, string> = {
  WORN_AROUND: 'Around', WORN_BACK: 'Back', WORN_BODY: 'Body', WORN_DON: 'Don',
  WORN_EAR: 'Ear', WORN_FEET1: 'Feet (1)', WORN_FEET2: 'Feet (2)', WORN_HAIR: 'Hair',
  WORN_HANDS: 'Hands', WORN_HEAD: 'Head', WORN_NECK: 'Neck', WORN_RING: 'Ring',
  WORN_TORSO1: 'Torso (1)', WORN_TORSO2: 'Torso (2)', WORN_TORSO3: 'Torso (3)',
  WORN_TRUNK1: 'Trunk (1)', WORN_TRUNK2: 'Trunk (2)', WORN_WRIST: 'Wrist', WORN_ARMOR: 'Armor',
}

const GenderNames: Record<number, string> = { 0: 'Male', 1: 'Female' }

const formatEnum = (value: string | number, map: Record<string | number, string>): string => {
  const label = map[value]
  return label ? `${label} (${value})` : String(value)
}

interface LogEntry {
  id: string
  timestamp: string
  event: string
  player: string
  accountId?: string
  details?: string
  roomNum?: number
  roomName?: string
}

const EventLabels: Record<string, string> = {
  login: 'User Login',
  logout: 'User Logout',
  game_enter: 'Game Enter',
  game_exit: 'Game Exit',
  character_create: 'New Character',
  level_up: 'Level Up',
  gm_grant: 'GM Granted',
  gm_revoke: 'GM Revoked',
  report: 'Player Report',
}

const EventColors: Record<string, string> = {
  login: 'text-green-400',
  logout: 'text-gray-500',
  game_enter: 'text-cyan-400',
  game_exit: 'text-gray-400',
  character_create: 'text-blue-400',
  level_up: 'text-yellow-400',
  gm_grant: 'text-amber-400',
  report: 'text-red-400',
  gm_revoke: 'text-red-400',
}

interface EngineEvent {
  time: string
  category: string
  message: string
}

const CategoryColors: Record<string, string> = {
  system: 'text-blue-400',
  time: 'text-yellow-400',
  monster: 'text-red-400',
  script: 'text-green-400',
  world: 'text-purple-400',
  weather: 'text-cyan-400',
}

type AdminTab = 'rooms' | 'players' | 'users' | 'items' | 'monsters' | 'logs' | 'events'

export default function AdminPanel() {
  const { user } = useAuth()
  const [tab, setTab] = useState<AdminTab>('rooms')
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [stats, setStats] = useState<Stats | null>(null)
  // Rooms
  const [rooms, setRooms] = useState<RoomSummary[]>([])
  const [search, setSearch] = useState('')
  const [selectedRoom, setSelectedRoom] = useState<RoomDetail | null>(null)
  // Players
  const [players, setPlayers] = useState<PlayerSummary[]>([])
  const [playerSearch, setPlayerSearch] = useState('')
  const [selectedPlayer, setSelectedPlayer] = useState<PlayerDetail | null>(null)
  const [reassignAccountId, setReassignAccountId] = useState('')
  const [reassignError, setReassignError] = useState('')
  const [showDeleted, setShowDeleted] = useState(false)
  // Items
  const [items, setItems] = useState<ItemSummary[]>([])
  const [itemSearch, setItemSearch] = useState('')
  const [selectedItem, setSelectedItem] = useState<ItemDetail | null>(null)
  // Monsters
  const [monsters, setMonsters] = useState<MonsterSummary[]>([])
  const [monsterSearch, setMonsterSearch] = useState('')
  const [selectedMonster, setSelectedMonster] = useState<MonsterDetail | null>(null)
  // Users
  const [accounts, setAccounts] = useState<AccountSummary[]>([])
  const [userSearch, setUserSearch] = useState('')
  const [selectedAccount, setSelectedAccount] = useState<AccountDetail | null>(null)
  // Logs
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [logEventFilter, setLogEventFilter] = useState('')
  const [logPlayerFilter, setLogPlayerFilter] = useState('')
  // Event Monitor
  const [events, setEvents] = useState<EngineEvent[]>([])
  const [eventWs, setEventWs] = useState<WebSocket | null>(null)
  const [eventConnected, setEventConnected] = useState(false)
  const [eventCatFilter, setEventCatFilter] = useState('')
  const eventScrollRef = useRef<HTMLDivElement>(null)

  const authHeaders = (): Record<string, string> => {
    if (!user?.token) return {}
    return { Authorization: `Bearer ${user.token}` }
  }

  useEffect(() => {
    fetch('/api/stats', { headers: authHeaders() }).then(r => r.ok ? r.json() : null).then(d => d && setStats(d))
    fetch('/api/rooms', { headers: authHeaders() }).then(r => r.ok ? r.json() : []).then((data: RoomSummary[]) => setRooms(data || []))
  }, [user])

  useEffect(() => {
    if (tab === 'players') {
      const url = showDeleted ? '/api/admin/characters/deleted' : '/api/admin/characters'
      fetch(url, { headers: authHeaders() })
        .then(r => r.json())
        .then((data: PlayerSummary[]) => setPlayers(data || []))
      // Also load accounts so the reassign dropdown is populated
      if (accounts.length === 0) {
        fetch('/api/admin/accounts', { headers: authHeaders() })
          .then(r => r.json())
          .then((data: AccountSummary[]) => setAccounts(data || []))
      }
    }
    if (tab === 'items') {
      fetch('/api/items', { headers: authHeaders() }).then(r => r.ok ? r.json() : []).then((data: ItemSummary[]) => setItems(data || []))
    }
    if (tab === 'monsters') {
      fetch('/api/monsters', { headers: authHeaders() }).then(r => r.ok ? r.json() : []).then((data: MonsterSummary[]) => setMonsters(data || []))
    }
    if (tab === 'users') {
      fetch('/api/admin/accounts', { headers: authHeaders() })
        .then(r => r.json())
        .then((data: AccountSummary[]) => setAccounts(data || []))
    }
    if (tab === 'logs') {
      fetchLogs()
    }
  }, [tab, showDeleted])

  // Auto-scroll events
  useEffect(() => {
    if (eventScrollRef.current) {
      eventScrollRef.current.scrollTop = eventScrollRef.current.scrollHeight
    }
  }, [events])

  // Event monitor WebSocket — connect only when on events tab
  useEffect(() => {
    if (tab !== 'events' || !user?.token) {
      if (eventWs) { eventWs.close(); setEventWs(null); setEventConnected(false) }
      return
    }
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws/events?token=${user.token}`)
    setEventWs(ws)
    ws.onopen = () => setEventConnected(true)
    ws.onmessage = (e) => {
      const msg = JSON.parse(e.data)
      if (msg.type === 'event' && msg.data) {
        setEvents(prev => {
          const next = [...prev, msg.data as EngineEvent]
          return next.length > 500 ? next.slice(-500) : next
        })
      }
    }
    ws.onclose = () => {
      setEventConnected(false)
      setEvents(prev => [...prev, { time: new Date().toISOString(), category: 'system', message: 'Event monitor disconnected.' }])
    }
    return () => { ws.close() }
  }, [tab, user?.token])

  const sortByNumberMatch = <T extends { number: number }>(list: T[], query: string): T[] => {
    const trimmed = query.trim()
    if (!trimmed || !/^\d+$/.test(trimmed)) return list
    const num = parseInt(trimmed, 10)
    return [...list].sort((a, b) => {
      // Exact number match first
      if (a.number === num && b.number !== num) return -1
      if (b.number === num && a.number !== num) return 1
      // Then starts-with match
      const aStarts = a.number.toString().startsWith(trimmed)
      const bStarts = b.number.toString().startsWith(trimmed)
      if (aStarts && !bStarts) return -1
      if (bStarts && !aStarts) return 1
      return 0
    })
  }

  const filteredRooms = sortByNumberMatch(rooms.filter(r =>
    r.name.toLowerCase().includes(search.toLowerCase()) ||
    r.number.toString().includes(search)
  ), search)

  const filteredPlayers = players.filter(p =>
    p.firstName.toLowerCase().includes(playerSearch.toLowerCase()) ||
    p.lastName.toLowerCase().includes(playerSearch.toLowerCase())
  )

  const filteredItems = sortByNumberMatch(items.filter(i =>
    i.name.toLowerCase().includes(itemSearch.toLowerCase()) ||
    i.number.toString().includes(itemSearch) ||
    i.type.toLowerCase().includes(itemSearch.toLowerCase())
  ), itemSearch)

  const filteredMonsters = sortByNumberMatch(monsters.filter(m =>
    m.name.toLowerCase().includes(monsterSearch.toLowerCase()) ||
    m.number.toString().includes(monsterSearch)
  ), monsterSearch)

  const filteredAccounts = accounts.filter(a =>
    a.name.toLowerCase().includes(userSearch.toLowerCase()) ||
    a.email.toLowerCase().includes(userSearch.toLowerCase())
  )

  const selectRoom = (num: number) => {
    setSidebarOpen(false)
    fetch(`/api/rooms/${num}`, { headers: authHeaders() }).then(r => r.json()).then(setSelectedRoom)
  }

  const selectItem = (num: number) => {
    setSidebarOpen(false)
    setSelectedItem(null)
    fetch(`/api/items/${num}`, { headers: authHeaders() })
      .then(r => { if (!r.ok) throw new Error(r.statusText); return r.json() })
      .then(setSelectedItem)
      .catch(err => console.error('Failed to load item', num, err))
  }

  const selectMonster = (num: number) => {
    setSidebarOpen(false)
    setSelectedMonster(null)
    fetch(`/api/monsters/${num}`, { headers: authHeaders() })
      .then(r => { if (!r.ok) throw new Error(r.statusText); return r.json() })
      .then(setSelectedMonster)
      .catch(err => console.error('Failed to load monster', num, err))
  }

  const selectPlayer = (firstName: string) => {
    setSidebarOpen(false)
    setReassignAccountId('')
    setReassignError('')
    fetch(`/api/admin/characters/${firstName}`, { headers: authHeaders() })
      .then(r => r.json())
      .then(setSelectedPlayer)
  }

  const selectAccount = (id: string) => {
    setSidebarOpen(false)
    fetch(`/api/admin/accounts/${id}`, { headers: authHeaders() })
      .then(r => r.json())
      .then(setSelectedAccount)
  }

  const toggleGM = (firstName: string, currentGM: boolean) => {
    fetch(`/api/characters/${firstName}/gm`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ isGM: !currentGM }),
    })
      .then(r => r.json())
      .then((updated: PlayerDetail) => {
        setSelectedPlayer(updated)
        setPlayers(prev => prev.map(p =>
          p.firstName === firstName ? { ...p, isGM: updated.isGM } : p
        ))
      })
  }

  const toggleAdmin = (accountId: string, currentAdmin: boolean) => {
    fetch(`/api/admin/accounts/${accountId}/admin`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ isAdmin: !currentAdmin }),
    })
      .then(r => r.json())
      .then((updated: AccountSummary) => {
        if (selectedAccount) {
          setSelectedAccount({ ...selectedAccount, account: updated })
        }
        setAccounts(prev => prev.map(a =>
          a.id === accountId ? { ...a, isAdmin: updated.isAdmin } : a
        ))
      })
  }

  const reassignCharacter = (firstName: string, newAccountId: string) => {
    setReassignError('')
    fetch(`/api/admin/characters/${firstName}/reassign`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ accountId: newAccountId }),
    })
      .then(async r => {
        if (!r.ok) { setReassignError(await r.text()); return }
        return r.json()
      })
      .then((updated: PlayerDetail | undefined) => {
        if (!updated) return
        setSelectedPlayer(updated)
        setPlayers(prev => prev.map(p =>
          p.firstName === firstName ? { ...p, accountId: updated.accountId } : p
        ))
        setReassignAccountId('')
      })
  }

  const recoverCharacter = (firstName: string) => {
    const newName = prompt(`Recover "${firstName}"? Enter a new first name (or leave blank to keep current):`)
    if (newName === null) return // cancelled
    fetch(`/api/admin/characters/${firstName}/recover`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify({ newFirstName: newName || '' }),
    })
      .then(async r => {
        if (!r.ok) { alert((await r.json()).error || 'Recovery failed'); return }
        alert('Character recovered!')
        setShowDeleted(false) // switch back to active view
      })
  }

  // Navigate from player detail to the owning user
  const navigateToOwner = (accountId: string) => {
    setTab('users')
    setTimeout(() => selectAccount(accountId), 100)
  }

  // Navigate from user detail to a character
  const navigateToCharacter = (firstName: string) => {
    setTab('players')
    setTimeout(() => selectPlayer(firstName), 100)
  }

  const formatDate = (dateStr: string) => {
    if (!dateStr) return 'Unknown'
    const d = new Date(dateStr)
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  const relativeTime = (dateStr?: string) => {
    if (!dateStr) return ''
    const now = Date.now()
    const then = new Date(dateStr).getTime()
    const diffSec = Math.floor((now - then) / 1000)
    if (diffSec < 60) return 'just now'
    const diffMin = Math.floor(diffSec / 60)
    if (diffMin < 60) return `${diffMin}m ago`
    const diffHr = Math.floor(diffMin / 60)
    if (diffHr < 24) return `${diffHr}h ago`
    const diffDay = Math.floor(diffHr / 24)
    if (diffDay < 30) return `${diffDay}d ago`
    const diffMo = Math.floor(diffDay / 30)
    return `${diffMo}mo ago`
  }

  const formatTimestamp = (dateStr: string) => {
    if (!dateStr) return ''
    const d = new Date(dateStr)
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }) + ' ' + d.toLocaleDateString()
  }

  const fetchLogs = () => {
    const params = new URLSearchParams()
    if (logEventFilter) params.set('event', logEventFilter)
    if (logPlayerFilter) params.set('player', logPlayerFilter)
    params.set('limit', '200')
    fetch(`/api/admin/logs?${params}`, { headers: authHeaders() })
      .then(r => r.json())
      .then((data: LogEntry[]) => setLogs(data || []))
  }

  // Find account name for a given accountId
  const getAccountName = (accountId?: string) => {
    if (!accountId) return null
    const acct = accounts.find(a => a.id === accountId)
    return acct?.name || null
  }

  return (
    <div className="flex flex-col h-full font-mono text-sm">
      {/* Tab bar */}
      <div className="flex gap-1 px-2 sm:px-4 py-2 bg-[#111] border-b border-[#333] overflow-x-auto">
        {/* Mobile sidebar toggle */}
        <button
          className="sm:hidden shrink-0 px-2 py-1 rounded text-xs text-gray-400 hover:text-white border border-[#444] mr-1"
          onClick={() => setSidebarOpen(o => !o)}
          title={sidebarOpen ? 'Show detail' : 'Show list'}
        >
          {sidebarOpen ? '→' : '☰'}
        </button>
        {(['rooms', 'items', 'monsters', 'players', 'users', 'logs', 'events'] as AdminTab[]).map(t => (
          <button
            key={t}
            onClick={() => { setTab(t); setSidebarOpen(false) }}
            className={`px-2 sm:px-3 py-1 rounded text-xs capitalize shrink-0 min-h-[32px] ${tab === t ? 'bg-amber-700 text-white' : 'text-gray-400 hover:text-white'}`}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* ===== ROOMS TAB ===== */}
        {tab === 'rooms' && (
          <>
            <div className={`${sidebarOpen ? 'flex' : 'hidden'} sm:flex w-full sm:w-80 border-r border-[#333] flex-col bg-[#111]`}>
              {stats && (
                <div className="p-3 border-b border-[#333] grid grid-cols-3 gap-2">
                  <div className="text-center">
                    <div className="text-amber-400 text-lg font-bold">{stats.rooms}</div>
                    <div className="text-gray-500 text-xs">Rooms</div>
                  </div>
                  <div className="text-center">
                    <div className="text-green-400 text-lg font-bold">{stats.items}</div>
                    <div className="text-gray-500 text-xs">Items</div>
                  </div>
                  <div className="text-center">
                    <div className="text-red-400 text-lg font-bold">{stats.monsters}</div>
                    <div className="text-gray-500 text-xs">Monsters</div>
                  </div>
                </div>
              )}
              <div className="p-2 border-b border-[#333]">
                <input
                  type="text"
                  value={search}
                  onChange={e => setSearch(e.target.value)}
                  placeholder="Search rooms..."
                  className="w-full bg-[#0a0a0a] border border-[#444] rounded px-2 py-1 text-gray-200 focus:border-amber-500 focus:outline-none text-xs"
                />
              </div>
              <div className="flex-1 overflow-y-auto">
                {filteredRooms.slice(0, 200).map(r => (
                  <div
                    key={r.number}
                    onClick={() => selectRoom(r.number)}
                    className={`px-3 py-2 cursor-pointer border-b border-[#222] hover:bg-[#1a1a2e] ${selectedRoom?.number === r.number ? 'bg-[#1a1a2e] border-l-2 border-l-amber-500' : ''}`}
                  >
                    <div className="flex justify-between">
                      <span className="text-gray-300 text-xs truncate">{r.name}</span>
                      <span className="text-gray-600 text-xs">#{r.number}</span>
                    </div>
                    <div className="flex gap-2 mt-0.5">
                      <span className="text-gray-600 text-[10px]">{r.terrain}</span>
                      <span className="text-gray-600 text-[10px]">{r.exits} exits</span>
                      <span className="text-gray-600 text-[10px]">{r.file}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
            <div className={`${sidebarOpen ? 'hidden sm:block' : 'block'} flex-1 overflow-y-auto p-4 sm:p-6`}>
              {selectedRoom ? (
                <div className="space-y-4">
                  <div>
                    <h2 className="text-amber-400 text-xl font-bold">[{selectedRoom.name}]</h2>
                    <span className="text-gray-500 text-xs">Room #{selectedRoom.number} | {selectedRoom.terrain} | {selectedRoom.lighting}</span>
                  </div>
                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">Description</h3>
                    <p className="text-gray-300 leading-relaxed">{selectedRoom.description || '(no description)'}</p>
                  </div>
                  {Object.keys(selectedRoom.exits).length > 0 && (
                    <div className="bg-[#111] border border-[#333] rounded p-4">
                      <h3 className="text-gray-400 text-xs uppercase mb-2">Exits ({Object.keys(selectedRoom.exits).length})</h3>
                      <div className="space-y-1">
                        {Object.entries(selectedRoom.exits).map(([dir, exit]) => (
                          <div
                            key={dir}
                            onClick={() => selectRoom(exit.room)}
                            className="flex items-center gap-2 px-2 py-1.5 bg-[#0a0a0a] border border-[#222] rounded text-xs hover:border-amber-500 cursor-pointer group"
                          >
                            <span className="text-amber-400 font-bold w-12 shrink-0">{dir}</span>
                            <span className="text-gray-500">-&gt;</span>
                            <span className="text-blue-400 group-hover:underline">{exit.roomName || 'Unknown'}</span>
                            <span className="text-gray-600">#{exit.room}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                  {selectedRoom.items && selectedRoom.items.length > 0 && (
                    <div className="bg-[#111] border border-[#333] rounded p-4">
                      <h3 className="text-gray-400 text-xs uppercase mb-2">Items ({selectedRoom.items.length})</h3>
                      <div className="space-y-2">
                        {selectedRoom.items.map((item, i) => {
                          const def = item.itemDef
                          const displayName = [
                            item.adj1Name,
                            def?.name,
                          ].filter(Boolean).join(' ') || `unknown #${item.archetype}`
                          return (
                            <div key={i} className="text-xs bg-[#0a0a0a] border border-[#222] rounded p-2">
                              <div className="flex items-center gap-2 flex-wrap">
                                <span className="text-green-400 font-bold">{displayName}</span>
                                <span className="text-gray-600">Ref {item.ref} | Archetype #{item.archetype}</span>
                              </div>
                              {def && (
                                <div className="flex flex-wrap gap-x-3 gap-y-0.5 mt-1 text-gray-400">
                                  <span>{formatEnum(def.type, ItemTypeNames)}</span>
                                  <span>wt: {def.weight}</span>
                                  <span>vol: {def.volume}</span>
                                  {def.substance && <span>{def.substance}</span>}
                                  {def.article && <span>article: {def.article}</span>}
                                  {def.wornSlot && <span>worn: {formatEnum(def.wornSlot, WornSlotNames)}</span>}
                                  {def.container && <span>container: {def.container}</span>}
                                  <span className="text-gray-600">{def.sourceFile}</span>
                                </div>
                              )}
                              {def?.flags && def.flags.length > 0 && (
                                <div className="flex flex-wrap gap-1 mt-1">
                                  {def.flags.map(f => (
                                    <span key={f} className="text-green-400 text-[10px] bg-green-900/30 px-1 rounded">{f}</span>
                                  ))}
                                </div>
                              )}
                              <div className="flex flex-wrap gap-x-3 gap-y-0.5 mt-1">
                                {item.adj2 ? <span className="text-purple-400">adj2: {item.adj2Name || item.adj2} ({item.adj2})</span> : null}
                                {item.adj3 ? <span className="text-purple-400">adj3: {item.adj3Name || item.adj3} ({item.adj3})</span> : null}
                                {item.val1 ? <span className="text-blue-400">val1={item.val1}</span> : null}
                                {item.val2 ? <span className="text-blue-400">val2={item.val2}</span> : null}
                                {item.val3 ? <span className="text-blue-400">val3={item.val3}</span> : null}
                                {item.val4 ? <span className="text-blue-400">val4={item.val4}</span> : null}
                                {item.val5 ? <span className="text-blue-400">val5={item.val5}</span> : null}
                                {item.state ? <span className="text-yellow-400">state: {item.state}</span> : null}
                                {item.isPut ? <span className="text-cyan-400">inside ref {item.putIn}</span> : null}
                                {item.extend ? <span className="text-gray-500">"{item.extend}"</span> : null}
                              </div>
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  )}
                  {selectedRoom.scripts && selectedRoom.scripts.length > 0 && (
                    <div className="bg-[#111] border border-[#333] rounded p-4">
                      <h3 className="text-gray-400 text-xs uppercase mb-2">Scripts ({selectedRoom.scripts.length} blocks)</h3>
                      <pre className="text-gray-500 text-[10px] overflow-x-auto max-h-60 overflow-y-auto">
                        {JSON.stringify(selectedRoom.scripts, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              ) : (
                <div className="flex items-center justify-center h-full text-gray-600">
                  Select a room to inspect
                </div>
              )}
            </div>
          </>
        )}

        {/* ===== ITEMS TAB ===== */}
        {tab === 'items' && (
          <>
            <div className={`${sidebarOpen ? 'flex' : 'hidden'} sm:flex w-full sm:w-80 border-r border-[#333] flex-col bg-[#111]`}>
              <div className="p-3 border-b border-[#333] text-center">
                <div className="text-green-400 text-lg font-bold">{items.length}</div>
                <div className="text-gray-500 text-xs">Items</div>
              </div>
              <div className="p-2 border-b border-[#333]">
                <input
                  type="text"
                  value={itemSearch}
                  onChange={e => setItemSearch(e.target.value)}
                  placeholder="Search items..."
                  className="w-full bg-[#0a0a0a] border border-[#444] rounded px-2 py-1 text-gray-200 focus:border-amber-500 focus:outline-none text-xs"
                />
              </div>
              <div className="flex-1 overflow-y-auto">
                {filteredItems.slice(0, 300).map(i => (
                  <div
                    key={i.number}
                    onClick={() => selectItem(i.number)}
                    className={`px-3 py-2 cursor-pointer border-b border-[#222] hover:bg-[#1a1a2e] ${selectedItem?.number === i.number ? 'bg-[#1a1a2e] border-l-2 border-l-green-500' : ''}`}
                  >
                    <div className="flex justify-between">
                      <span className="text-gray-300 text-xs truncate">{i.name}</span>
                      <span className="text-gray-600 text-xs">#{i.number}</span>
                    </div>
                    <div className="flex gap-2 mt-0.5">
                      <span className="text-gray-600 text-[10px]">{i.type}</span>
                      <span className="text-gray-600 text-[10px]">wt:{i.weight}</span>
                      {i.substance && <span className="text-gray-600 text-[10px]">{i.substance}</span>}
                      <span className="text-gray-600 text-[10px]">{i.sourceFile}</span>
                    </div>
                  </div>
                ))}
                {filteredItems.length === 0 && (
                  <div className="p-4 text-gray-600 text-center text-xs">No items found</div>
                )}
              </div>
            </div>
            <div className={`${sidebarOpen ? 'hidden sm:block' : 'block'} flex-1 overflow-y-auto p-4 sm:p-6`}>
              {selectedItem ? (
                <div className="space-y-4">
                  <div>
                    <h2 className="text-green-400 text-xl font-bold">
                      {selectedItem.resolvedName}
                    </h2>
                    <span className="text-gray-500 text-xs">
                      Item #{selectedItem.number} | {formatEnum(selectedItem.type, ItemTypeNames)} | {selectedItem.sourceFile}
                    </span>
                  </div>

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">Properties</h3>
                    <div className="grid grid-cols-3 gap-2 text-xs">
                      <div><span className="text-gray-500">Name:</span> <span className="text-gray-200">{selectedItem.resolvedName} ({selectedItem.nameId})</span></div>
                      <div><span className="text-gray-500">Type:</span> <span className="text-gray-200">{formatEnum(selectedItem.type, ItemTypeNames)}</span></div>
                      <div><span className="text-gray-500">Article:</span> <span className="text-gray-200">{selectedItem.article || 'none'}</span></div>
                      <div><span className="text-gray-500">Weight:</span> <span className="text-gray-200">{selectedItem.weight}</span></div>
                      <div><span className="text-gray-500">Volume:</span> <span className="text-gray-200">{selectedItem.volume}</span></div>
                      <div><span className="text-gray-500">Substance:</span> <span className="text-gray-200">{selectedItem.substance || 'none'}</span></div>
                    </div>
                  </div>

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">Parameters</h3>
                    <div className="grid grid-cols-3 gap-2 text-xs">
                      <div><span className="text-gray-500">Param 1:</span> <span className="text-gray-200">{selectedItem.parameter1}</span></div>
                      <div><span className="text-gray-500">Param 2:</span> <span className="text-gray-200">{selectedItem.parameter2}</span></div>
                      <div><span className="text-gray-500">Param 3:</span> <span className="text-gray-200">{selectedItem.parameter3}</span></div>
                    </div>
                  </div>

                  {(selectedItem.container || selectedItem.wornSlot) && (
                    <div className="bg-[#111] border border-[#333] rounded p-4">
                      <h3 className="text-gray-400 text-xs uppercase mb-2">Container / Worn</h3>
                      <div className="grid grid-cols-2 gap-2 text-xs">
                        {selectedItem.container && <div><span className="text-gray-500">Container:</span> <span className="text-gray-200">{selectedItem.container}</span></div>}
                        {selectedItem.interior !== undefined && selectedItem.interior > 0 && <div><span className="text-gray-500">Interior:</span> <span className="text-gray-200">{selectedItem.interior}</span></div>}
                        {selectedItem.wornSlot && <div><span className="text-gray-500">Worn Slot:</span> <span className="text-gray-200">{formatEnum(selectedItem.wornSlot, WornSlotNames)}</span></div>}
                      </div>
                    </div>
                  )}

                  {selectedItem.flags && selectedItem.flags.length > 0 && (
                    <div className="bg-[#111] border border-[#333] rounded p-4">
                      <h3 className="text-gray-400 text-xs uppercase mb-2">Flags</h3>
                      <div className="flex flex-wrap gap-1">
                        {selectedItem.flags.map(f => (
                          <span key={f} className="text-green-400 text-[10px] bg-green-900/30 px-1.5 py-0.5 rounded">{f}</span>
                        ))}
                      </div>
                    </div>
                  )}

                  {selectedItem.scripts && selectedItem.scripts.length > 0 && (
                    <div className="bg-[#111] border border-[#333] rounded p-4">
                      <h3 className="text-gray-400 text-xs uppercase mb-2">Scripts ({selectedItem.scripts.length} blocks)</h3>
                      <pre className="text-gray-500 text-[10px] overflow-x-auto max-h-60 overflow-y-auto">
                        {JSON.stringify(selectedItem.scripts, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              ) : (
                <div className="flex items-center justify-center h-full text-gray-600">
                  Select an item to inspect
                </div>
              )}
            </div>
          </>
        )}

        {/* ===== MONSTERS TAB ===== */}
        {tab === 'monsters' && (
          <>
            <div className={`${sidebarOpen ? 'flex' : 'hidden'} sm:flex w-full sm:w-80 border-r border-[#333] flex-col bg-[#111]`}>
              <div className="p-3 border-b border-[#333] text-center">
                <div className="text-red-400 text-lg font-bold">{monsters.length}</div>
                <div className="text-gray-500 text-xs">Monsters</div>
              </div>
              <div className="p-2 border-b border-[#333]">
                <input
                  type="text"
                  value={monsterSearch}
                  onChange={e => setMonsterSearch(e.target.value)}
                  placeholder="Search monsters..."
                  className="w-full bg-[#0a0a0a] border border-[#444] rounded px-2 py-1 text-gray-200 focus:border-amber-500 focus:outline-none text-xs"
                />
              </div>
              <div className="flex-1 overflow-y-auto">
                {filteredMonsters.slice(0, 300).map(m => (
                  <div
                    key={m.number}
                    onClick={() => selectMonster(m.number)}
                    className={`px-3 py-2 cursor-pointer border-b border-[#222] hover:bg-[#1a1a2e] ${selectedMonster?.number === m.number ? 'bg-[#1a1a2e] border-l-2 border-l-red-500' : ''}`}
                  >
                    <div className="flex justify-between items-center">
                      <span className="text-gray-300 text-xs truncate">{m.name}</span>
                      <div className="flex gap-1 items-center">
                        {m.unique && <span className="text-purple-400 text-[10px] bg-purple-900/30 px-1 rounded">UNIQUE</span>}
                        <span className="text-gray-600 text-xs">#{m.number}</span>
                      </div>
                    </div>
                    <div className="flex gap-2 mt-0.5">
                      <span className="text-gray-600 text-[10px]">{m.bodyType}</span>
                      <span className="text-gray-600 text-[10px]">HP:{m.body}</span>
                      <span className="text-gray-600 text-[10px]">ATK:{m.attack1}</span>
                      <span className="text-gray-600 text-[10px]">DEF:{m.defense}</span>
                      <span className="text-gray-600 text-[10px]">{m.sourceFile}</span>
                    </div>
                  </div>
                ))}
                {filteredMonsters.length === 0 && (
                  <div className="p-4 text-gray-600 text-center text-xs">No monsters found</div>
                )}
              </div>
            </div>
            <div className={`${sidebarOpen ? 'hidden sm:block' : 'block'} flex-1 overflow-y-auto p-4 sm:p-6`}>
              {selectedMonster ? (
                <div className="space-y-4">
                  <div>
                    <h2 className="text-red-400 text-xl font-bold">
                      {selectedMonster.adjName ? `${selectedMonster.adjName} ` : ''}{selectedMonster.name}
                    </h2>
                    <span className="text-gray-500 text-xs">
                      Monster #{selectedMonster.number} | {selectedMonster.bodyType} | {selectedMonster.sourceFile}
                    </span>
                  </div>

                  {selectedMonster.description && (
                    <div className="bg-[#111] border border-[#333] rounded p-4">
                      <h3 className="text-gray-400 text-xs uppercase mb-2">Description</h3>
                      <p className="text-gray-300 leading-relaxed">{selectedMonster.description}</p>
                    </div>
                  )}

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">Combat Stats</h3>
                    <div className="grid grid-cols-4 gap-2 text-xs">
                      <div><span className="text-gray-500">HP:</span> <span className="text-red-400">{selectedMonster.body}</span></div>
                      <div><span className="text-gray-500">Attack 1:</span> <span className="text-gray-200">{selectedMonster.attack1}</span></div>
                      <div><span className="text-gray-500">Attack 2:</span> <span className="text-gray-200">{selectedMonster.attack2}</span></div>
                      <div><span className="text-gray-500">Defense:</span> <span className="text-gray-200">{selectedMonster.defense}</span></div>
                      <div><span className="text-gray-500">Armor:</span> <span className="text-gray-200">{selectedMonster.armor}</span></div>
                      <div><span className="text-gray-500">Speed:</span> <span className="text-gray-200">{selectedMonster.speed}</span></div>
                      <div><span className="text-gray-500">Strategy:</span> <span className="text-gray-200">{selectedMonster.strategy}</span></div>
                      <div><span className="text-gray-500">Treasure:</span> <span className="text-gray-200">{selectedMonster.treasure}</span></div>
                    </div>
                  </div>

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">Properties</h3>
                    <div className="grid grid-cols-3 gap-2 text-xs">
                      <div><span className="text-gray-500">Body Type:</span> <span className="text-gray-200">{selectedMonster.bodyType}</span></div>
                      <div><span className="text-gray-500">Race:</span> <span className="text-gray-200">{formatEnum(selectedMonster.race, RaceNames)}</span></div>
                      <div><span className="text-gray-500">Gender:</span> <span className="text-gray-200">{formatEnum(selectedMonster.gender, GenderNames)}</span></div>
                      {selectedMonster.adjName && (
                        <div><span className="text-gray-500">Adjective:</span> <span className="text-gray-200">{selectedMonster.adjName} ({selectedMonster.adjective})</span></div>
                      )}
                    </div>
                    <div className="flex gap-2 mt-2">
                      {selectedMonster.unique && <span className="text-purple-400 text-[10px] bg-purple-900/30 px-1.5 py-0.5 rounded">UNIQUE</span>}
                    </div>
                  </div>

                  {selectedMonster.scripts && selectedMonster.scripts.length > 0 && (
                    <div className="bg-[#111] border border-[#333] rounded p-4">
                      <h3 className="text-gray-400 text-xs uppercase mb-2">Scripts ({selectedMonster.scripts.length} blocks)</h3>
                      <pre className="text-gray-500 text-[10px] overflow-x-auto max-h-60 overflow-y-auto">
                        {JSON.stringify(selectedMonster.scripts, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              ) : (
                <div className="flex items-center justify-center h-full text-gray-600">
                  Select a monster to inspect
                </div>
              )}
            </div>
          </>
        )}

        {/* ===== PLAYERS TAB ===== */}
        {tab === 'players' && (
          <>
            <div className={`${sidebarOpen ? 'flex' : 'hidden'} sm:flex w-full sm:w-80 border-r border-[#333] flex-col bg-[#111]`}>
              <div className="p-3 border-b border-[#333] text-center">
                <div className="text-amber-400 text-lg font-bold">{players.length}</div>
                <div className="text-gray-500 text-xs">Characters</div>
              </div>
              <div className="p-2 border-b border-[#333]">
                <input
                  type="text"
                  value={playerSearch}
                  onChange={e => setPlayerSearch(e.target.value)}
                  placeholder="Search characters..."
                  className="w-full bg-[#0a0a0a] border border-[#444] rounded px-2 py-1 text-gray-200 focus:border-amber-500 focus:outline-none text-xs"
                />
                <label className="flex items-center gap-1 mt-1 text-gray-500 text-xs cursor-pointer">
                  <input type="checkbox" checked={showDeleted} onChange={e => setShowDeleted(e.target.checked)} className="accent-amber-500" />
                  Show deleted characters
                </label>
              </div>
              <div className="flex-1 overflow-y-auto">
                {filteredPlayers.map(p => (
                  <div
                    key={p.firstName + p.lastName}
                    onClick={() => selectPlayer(p.firstName)}
                    className={`px-3 py-2 cursor-pointer border-b border-[#222] hover:bg-[#1a1a2e] ${selectedPlayer?.firstName === p.firstName ? 'bg-[#1a1a2e] border-l-2 border-l-amber-500' : ''}`}
                  >
                    <div className="flex justify-between items-center">
                      <span className="text-gray-300 text-xs truncate">
                        {p.firstName} {p.lastName}
                      </span>
                      <div className="flex gap-1">
                        {p.isGM && <span className="text-amber-400 text-[10px] bg-amber-900/30 px-1 rounded">GM</span>}
                        {p.dead && <span className="text-red-400 text-[10px] bg-red-900/30 px-1 rounded">DEAD</span>}
                      </div>
                    </div>
                    <div className="flex gap-2 mt-0.5">
                      <span className="text-gray-600 text-[10px]">{RaceNames[p.race] || 'Unknown'}</span>
                      <span className="text-gray-600 text-[10px]">Lvl {p.level}</span>
                      <span className="text-gray-600 text-[10px]">Room {p.roomNumber}</span>
                      <span className="text-gray-600 text-[10px]">BP {p.bodyPoints}/{p.maxBodyPoints}</span>
                      {p.updatedAt && <span className="text-gray-600 text-[10px] ml-auto">{relativeTime(p.updatedAt)}</span>}
                    </div>
                  </div>
                ))}
                {filteredPlayers.length === 0 && (
                  <div className="p-4 text-gray-600 text-center text-xs">No characters found</div>
                )}
              </div>
            </div>
            <div className={`${sidebarOpen ? 'hidden sm:block' : 'block'} flex-1 overflow-y-auto p-4 sm:p-6`}>
              {selectedPlayer ? (
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <h2 className="text-amber-400 text-xl font-bold">
                        {selectedPlayer.firstName} {selectedPlayer.lastName}
                      </h2>
                      <span className="text-gray-500 text-xs">
                        {RaceNames[selectedPlayer.race]} | {selectedPlayer.gender === 0 ? 'Male' : 'Female'} | Level {selectedPlayer.level}
                      </span>
                      {selectedPlayer.accountId && (
                        <div className="mt-1">
                          <span className="text-gray-600 text-xs">Owner: </span>
                          <button
                            onClick={() => navigateToOwner(selectedPlayer.accountId!)}
                            className="text-blue-400 text-xs hover:underline"
                          >
                            {getAccountName(selectedPlayer.accountId) || selectedPlayer.accountId}
                          </button>
                        </div>
                      )}
                    </div>
                    <button
                      onClick={() => toggleGM(selectedPlayer.firstName, selectedPlayer.isGM)}
                      className={`px-3 py-1.5 rounded text-xs font-bold ${
                        selectedPlayer.isGM
                          ? 'bg-amber-700 text-white hover:bg-amber-600'
                          : 'bg-[#222] text-gray-400 border border-[#444] hover:border-amber-500'
                      }`}
                    >
                      {selectedPlayer.isGM ? 'Revoke GM' : 'Grant GM'}
                    </button>
                  </div>

                  {showDeleted && (
                    <button
                      onClick={() => recoverCharacter(selectedPlayer.firstName)}
                      className="px-3 py-1.5 rounded text-xs font-bold bg-green-800 text-white hover:bg-green-700 mb-3"
                    >
                      Recover Character
                    </button>
                  )}

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">Reassign Owner</h3>
                    <div className="flex gap-2 items-center">
                      <select
                        value={reassignAccountId}
                        onChange={e => { setReassignAccountId(e.target.value); setReassignError('') }}
                        className="flex-1 bg-[#0a0a0a] border border-[#444] rounded px-2 py-1 text-gray-200 focus:border-amber-500 focus:outline-none text-xs"
                      >
                        <option value="">— select account —</option>
                        {accounts.map(a => (
                          <option key={a.id} value={a.id}>
                            {a.name} ({a.email})
                          </option>
                        ))}
                      </select>
                      <button
                        onClick={() => reassignAccountId && reassignCharacter(selectedPlayer.firstName, reassignAccountId)}
                        disabled={!reassignAccountId}
                        className="px-3 py-1 bg-[#222] border border-[#444] rounded text-xs text-gray-300 hover:border-amber-500 disabled:opacity-40 disabled:cursor-not-allowed"
                      >
                        Reassign
                      </button>
                    </div>
                    {reassignError && <div className="text-red-400 text-xs mt-1">{reassignError}</div>}
                  </div>

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">Attributes</h3>
                    <div className="grid grid-cols-4 gap-2 text-xs">
                      <div><span className="text-gray-500">STR:</span> <span className="text-gray-200">{selectedPlayer.strength}</span></div>
                      <div><span className="text-gray-500">AGI:</span> <span className="text-gray-200">{selectedPlayer.agility}</span></div>
                      <div><span className="text-gray-500">QUI:</span> <span className="text-gray-200">{selectedPlayer.quickness}</span></div>
                      <div><span className="text-gray-500">CON:</span> <span className="text-gray-200">{selectedPlayer.constitution}</span></div>
                      <div><span className="text-gray-500">PER:</span> <span className="text-gray-200">{selectedPlayer.perception}</span></div>
                      <div><span className="text-gray-500">WIL:</span> <span className="text-gray-200">{selectedPlayer.willpower}</span></div>
                      <div><span className="text-gray-500">EMP:</span> <span className="text-gray-200">{selectedPlayer.empathy}</span></div>
                    </div>
                  </div>

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">Resources</h3>
                    <div className="grid grid-cols-2 gap-2 text-xs">
                      <div><span className="text-red-400">BP:</span> {selectedPlayer.bodyPoints}/{selectedPlayer.maxBodyPoints}</div>
                      <div><span className="text-yellow-400">FT:</span> {selectedPlayer.fatigue}/{selectedPlayer.maxFatigue}</div>
                      <div><span className="text-blue-400">MP:</span> {selectedPlayer.mana}/{selectedPlayer.maxMana}</div>
                      <div><span className="text-purple-400">PSI:</span> {selectedPlayer.psi}/{selectedPlayer.maxPsi}</div>
                    </div>
                  </div>

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">Status</h3>
                    <div className="grid grid-cols-2 gap-2 text-xs">
                      <div><span className="text-gray-500">Room:</span> {selectedPlayer.roomNumber}</div>
                      <div><span className="text-gray-500">XP:</span> {selectedPlayer.experience}</div>
                      <div><span className="text-gray-500">Gold:</span> {selectedPlayer.gold}</div>
                      <div><span className="text-gray-500">Silver:</span> {selectedPlayer.silver}</div>
                      <div><span className="text-gray-500">Copper:</span> {selectedPlayer.copper}</div>
                      <div><span className="text-gray-500">Position:</span> {['Standing','Sitting','Laying','Kneeling','Flying'][selectedPlayer.position] || selectedPlayer.position}</div>
                    </div>
                    <div className="flex gap-2 mt-2">
                      {selectedPlayer.isGM && <span className="text-amber-400 text-[10px] bg-amber-900/30 px-1.5 py-0.5 rounded">GM</span>}
                      {selectedPlayer.dead && <span className="text-red-400 text-[10px] bg-red-900/30 px-1.5 py-0.5 rounded">DEAD</span>}
                      {selectedPlayer.hidden && <span className="text-gray-400 text-[10px] bg-gray-900/30 px-1.5 py-0.5 rounded">HIDDEN</span>}
                      {selectedPlayer.bleeding && <span className="text-red-400 text-[10px] bg-red-900/30 px-1.5 py-0.5 rounded">BLEEDING</span>}
                      {selectedPlayer.stunned && <span className="text-yellow-400 text-[10px] bg-yellow-900/30 px-1.5 py-0.5 rounded">STUNNED</span>}
                      {selectedPlayer.diseased && <span className="text-green-400 text-[10px] bg-green-900/30 px-1.5 py-0.5 rounded">DISEASED</span>}
                      {selectedPlayer.poisoned && <span className="text-green-400 text-[10px] bg-green-900/30 px-1.5 py-0.5 rounded">POISONED</span>}
                    </div>
                  </div>

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">
                      Inventory ({selectedPlayer.inventory?.length || 0} items)
                    </h3>
                    {selectedPlayer.inventory && selectedPlayer.inventory.length > 0 ? (
                      <div className="space-y-1">
                        {selectedPlayer.inventory.map((item, i) => (
                          <div key={i} className="text-xs text-gray-300">
                            {i}. Archetype #{item.archetype}
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="text-xs text-gray-600">(empty)</div>
                    )}
                  </div>

                  {selectedPlayer.skills && Object.keys(selectedPlayer.skills).length > 0 && (
                    <div className="bg-[#111] border border-[#333] rounded p-4">
                      <h3 className="text-gray-400 text-xs uppercase mb-2">Skills</h3>
                      <div className="grid grid-cols-3 gap-1 text-xs">
                        {Object.entries(selectedPlayer.skills).map(([id, level]) => (
                          <div key={id} className="text-gray-300">Skill #{id}: Lvl {level}</div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                <div className="flex items-center justify-center h-full text-gray-600">
                  Select a character to inspect
                </div>
              )}
            </div>
          </>
        )}

        {/* ===== USERS TAB ===== */}
        {tab === 'users' && (
          <>
            <div className={`${sidebarOpen ? 'flex' : 'hidden'} sm:flex w-full sm:w-80 border-r border-[#333] flex-col bg-[#111]`}>
              <div className="p-3 border-b border-[#333] text-center">
                <div className="text-amber-400 text-lg font-bold">{accounts.length}</div>
                <div className="text-gray-500 text-xs">Users</div>
              </div>
              <div className="p-2 border-b border-[#333]">
                <input
                  type="text"
                  value={userSearch}
                  onChange={e => setUserSearch(e.target.value)}
                  placeholder="Search users..."
                  className="w-full bg-[#0a0a0a] border border-[#444] rounded px-2 py-1 text-gray-200 focus:border-amber-500 focus:outline-none text-xs"
                />
              </div>
              <div className="flex-1 overflow-y-auto">
                {filteredAccounts.map(a => (
                  <div
                    key={a.id}
                    onClick={() => selectAccount(a.id)}
                    className={`px-3 py-2 cursor-pointer border-b border-[#222] hover:bg-[#1a1a2e] ${selectedAccount?.account.id === a.id ? 'bg-[#1a1a2e] border-l-2 border-l-amber-500' : ''}`}
                  >
                    <div className="flex justify-between items-center">
                      <div className="flex items-center gap-2 min-w-0">
                        {a.picture && <img src={a.picture} alt="" className="w-5 h-5 rounded-full shrink-0" />}
                        <span className="text-gray-300 text-xs truncate">{a.name}</span>
                      </div>
                      <div className="flex gap-1 shrink-0">
                        {a.isAdmin && <span className="text-red-400 text-[10px] bg-red-900/30 px-1 rounded">ADMIN</span>}
                      </div>
                    </div>
                    <div className="flex justify-between text-gray-600 text-[10px] mt-0.5">
                      <span className="truncate">{a.email}</span>
                      <span className="shrink-0 ml-1">{relativeTime(a.updatedAt)}</span>
                    </div>
                  </div>
                ))}
                {filteredAccounts.length === 0 && (
                  <div className="p-4 text-gray-600 text-center text-xs">No users found</div>
                )}
              </div>
            </div>
            <div className={`${sidebarOpen ? 'hidden sm:block' : 'block'} flex-1 overflow-y-auto p-4 sm:p-6`}>
              {selectedAccount ? (
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      {selectedAccount.account.picture && (
                        <img src={selectedAccount.account.picture} alt="" className="w-12 h-12 rounded-full" />
                      )}
                      <div>
                        <h2 className="text-amber-400 text-xl font-bold">{selectedAccount.account.name}</h2>
                        <span className="text-gray-500 text-xs">{selectedAccount.account.email}</span>
                      </div>
                    </div>
                    <button
                      onClick={() => toggleAdmin(selectedAccount.account.id, selectedAccount.account.isAdmin)}
                      className={`px-3 py-1.5 rounded text-xs font-bold ${
                        selectedAccount.account.isAdmin
                          ? 'bg-red-700 text-white hover:bg-red-600'
                          : 'bg-[#222] text-gray-400 border border-[#444] hover:border-red-500'
                      }`}
                    >
                      {selectedAccount.account.isAdmin ? 'Revoke Admin' : 'Grant Admin'}
                    </button>
                  </div>

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">Account Details</h3>
                    <div className="grid grid-cols-2 gap-2 text-xs">
                      <div><span className="text-gray-500">ID:</span> <span className="text-gray-300">{selectedAccount.account.id}</span></div>
                      <div><span className="text-gray-500">Google ID:</span> <span className="text-gray-300 truncate">{selectedAccount.account.googleId}</span></div>
                      <div><span className="text-gray-500">Created:</span> <span className="text-gray-300">{formatDate(selectedAccount.account.createdAt)}</span></div>
                      <div><span className="text-gray-500">Updated:</span> <span className="text-gray-300">{formatDate(selectedAccount.account.updatedAt)}</span></div>
                    </div>
                    <div className="flex gap-2 mt-2">
                      {selectedAccount.account.isAdmin && <span className="text-red-400 text-[10px] bg-red-900/30 px-1.5 py-0.5 rounded">ADMIN</span>}
                    </div>
                  </div>

                  <div className="bg-[#111] border border-[#333] rounded p-4">
                    <h3 className="text-gray-400 text-xs uppercase mb-2">
                      Characters ({selectedAccount.characters?.length || 0})
                    </h3>
                    {selectedAccount.characters && selectedAccount.characters.length > 0 ? (
                      <div className="space-y-2">
                        {selectedAccount.characters.map(c => (
                          <button
                            key={c.firstName + c.lastName}
                            onClick={() => navigateToCharacter(c.firstName)}
                            className="w-full text-left px-3 py-2 bg-[#0a0a0a] border border-[#333] rounded hover:border-amber-500"
                          >
                            <div className="flex justify-between items-center">
                              <span className="text-blue-400 text-xs hover:underline">
                                {c.firstName} {c.lastName}
                              </span>
                              <div className="flex gap-1">
                                {c.isGM && <span className="text-amber-400 text-[10px] bg-amber-900/30 px-1 rounded">GM</span>}
                                {c.dead && <span className="text-red-400 text-[10px] bg-red-900/30 px-1 rounded">DEAD</span>}
                              </div>
                            </div>
                            <div className="flex gap-2 mt-0.5">
                              <span className="text-gray-600 text-[10px]">{RaceNames[c.race] || 'Unknown'}</span>
                              <span className="text-gray-600 text-[10px]">Lvl {c.level}</span>
                              <span className="text-gray-600 text-[10px]">Room {c.roomNumber}</span>
                              <span className="text-gray-600 text-[10px]">BP {c.bodyPoints}/{c.maxBodyPoints}</span>
                            </div>
                          </button>
                        ))}
                      </div>
                    ) : (
                      <div className="text-xs text-gray-600">No characters</div>
                    )}
                  </div>
                </div>
              ) : (
                <div className="flex items-center justify-center h-full text-gray-600">
                  Select a user to inspect
                </div>
              )}
            </div>
          </>
        )}
        {/* ===== LOGS TAB ===== */}
        {tab === 'logs' && (
          <div className="flex-1 flex flex-col overflow-hidden">
            <div className="flex gap-2 p-3 bg-[#111] border-b border-[#333] items-center">
              <select
                value={logEventFilter}
                onChange={e => setLogEventFilter(e.target.value)}
                className="bg-[#0a0a0a] border border-[#444] rounded px-2 py-1 text-gray-200 focus:border-amber-500 focus:outline-none text-xs"
              >
                <option value="">All events</option>
                {Object.entries(EventLabels).map(([k, v]) => (
                  <option key={k} value={k}>{v}</option>
                ))}
              </select>
              <input
                type="text"
                value={logPlayerFilter}
                onChange={e => setLogPlayerFilter(e.target.value)}
                placeholder="Filter by player..."
                className="bg-[#0a0a0a] border border-[#444] rounded px-2 py-1 text-gray-200 focus:border-amber-500 focus:outline-none text-xs w-48"
              />
              <button
                onClick={fetchLogs}
                className="px-3 py-1 bg-amber-700 text-white rounded text-xs hover:bg-amber-600"
              >
                Refresh
              </button>
              <span className="text-gray-600 text-xs ml-auto">{logs.length} entries</span>
            </div>
            <div className="flex-1 overflow-y-auto">
              <table className="w-full text-xs">
                <thead className="text-gray-500 bg-[#111] sticky top-0">
                  <tr>
                    <th className="text-left px-3 py-2 w-40">Time</th>
                    <th className="text-left px-3 py-2 w-28">Event</th>
                    <th className="text-left px-3 py-2 w-40">Player</th>
                    <th className="text-left px-3 py-2">Details</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((entry, i) => (
                    <tr key={entry.id || i} className="border-b border-[#222] hover:bg-[#1a1a2e]">
                      <td className="px-3 py-1.5 text-gray-500">{formatTimestamp(entry.timestamp)}</td>
                      <td className={`px-3 py-1.5 ${EventColors[entry.event] || 'text-gray-300'}`}>
                        {EventLabels[entry.event] || entry.event}
                      </td>
                      <td className="px-3 py-1.5">
                        {entry.player ? (
                          entry.event === 'login' || entry.event === 'logout' ? (
                            entry.accountId ? (
                              <button onClick={() => { setTab('users'); setTimeout(() => selectAccount(entry.accountId!), 100) }} className="text-blue-400 hover:underline">{entry.player}</button>
                            ) : <span className="text-gray-300">{entry.player}</span>
                          ) : (
                            <button onClick={() => { setTab('players'); setTimeout(() => selectPlayer(entry.player.split(' ')[0]), 100) }} className="text-blue-400 hover:underline">{entry.player}</button>
                          )
                        ) : <span className="text-gray-500">-</span>}
                      </td>
                      <td className="px-3 py-1.5 text-gray-500">
                        {entry.accountId && entry.event !== 'login' && entry.event !== 'logout' && (
                          <button onClick={() => { setTab('users'); setTimeout(() => selectAccount(entry.accountId!), 100) }} className="text-blue-400 hover:underline mr-2">[user]</button>
                        )}
                        {entry.details}
                      </td>
                    </tr>
                  ))}
                  {logs.length === 0 && (
                    <tr><td colSpan={4} className="px-3 py-8 text-center text-gray-600">No log entries</td></tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}
        {/* ===== EVENTS TAB ===== */}
        {tab === 'events' && (
          <div className="flex-1 flex flex-col overflow-hidden">
            <div className="flex gap-2 p-3 bg-[#111] border-b border-[#333] items-center">
              <div className={`w-2 h-2 rounded-full ${eventConnected ? 'bg-green-500' : 'bg-red-500'}`} />
              <span className="text-gray-400 text-xs">{eventConnected ? 'Connected' : 'Disconnected'}</span>
              <select
                value={eventCatFilter}
                onChange={e => setEventCatFilter(e.target.value)}
                className="bg-[#0a0a0a] border border-[#444] rounded px-2 py-1 text-gray-200 focus:border-amber-500 focus:outline-none text-xs"
              >
                <option value="">All categories</option>
                <option value="system">System</option>
                <option value="time">Time</option>
                <option value="monster">Monster</option>
                <option value="script">Script</option>
                <option value="world">World State</option>
                <option value="weather">Weather</option>
              </select>
              <button
                onClick={() => setEvents([])}
                className="px-3 py-1 bg-[#222] border border-[#444] rounded text-xs text-gray-300 hover:border-amber-500"
              >
                Clear
              </button>
              <span className="text-gray-600 text-xs ml-auto">{events.length} events</span>
            </div>
            <div ref={eventScrollRef} className="flex-1 overflow-y-auto font-mono text-xs p-2 bg-[#0a0a0a]">
              {events
                .filter(ev => !eventCatFilter || ev.category === eventCatFilter)
                .map((ev, i) => {
                  const t = new Date(ev.time)
                  const ts = t.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
                  return (
                    <div key={i} className="py-0.5 border-b border-[#111] flex gap-2">
                      <span className="text-gray-600 w-20 shrink-0">{ts}</span>
                      <span className={`w-16 shrink-0 ${CategoryColors[ev.category] || 'text-gray-400'}`}>
                        [{ev.category}]
                      </span>
                      <span className="text-gray-300">{ev.message}</span>
                    </div>
                  )
                })}
              {events.length === 0 && (
                <div className="text-gray-600 text-center py-8">Waiting for events...</div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
