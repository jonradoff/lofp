import { useState } from 'react'

const RACES = [
  {
    id: 1, name: 'Human',
    stats: 'STR 30-100  AGI 30-100  QUI 30-100  CON 30-100  PER 30-100  WIL 40-110  EMP 30-100',
    desc: 'The elder race of the Shattered Realms. Humans have existed since time out of mind and tend to excel at any skill they pursue, attributed to their superior willpower. Cybernetic technology was originally developed for humans, so they may employ any such device. They get along with other races as well as they get along with each other.',
    ability: 'Can use all cybernetic implants.',
  },
  {
    id: 2, name: 'Aelfen',
    stats: 'STR 20-90  AGI 40-110  QUI 40-110  CON 1-70  PER 40-110  WIL 30-100  EMP 40-110',
    desc: 'Tall, slender humanoids with pretty features and pointed ears. Quick, agile and with heightened senses from living in the woodlands. They may live for centuries and are extremely resistant to natural diseases. Carefree and aloof with mercurial tempers, they are lovers of beauty and the arts.',
    ability: 'CALL — summon a woodland creature to serve you (wilderness only).',
  },
  {
    id: 3, name: 'Highlander',
    stats: 'STR 40-110  AGI 20-90  QUI 20-90  CON 50-120  PER 30-100  WIL 30-100  EMP 10-80',
    desc: 'Short, rugged mountain folk known as Children of the Stone. Strong, enduring, and especially resistant to magic. They see well in darkness and are comfortable with technological implements. Their women can be quite attractive, and they certainly do not have beards.',
    ability: 'BLEND into cave/mountain terrain. At level 5: MOLD gemstones into more valuable gems.',
  },
  {
    id: 4, name: 'Wolfling',
    stats: 'STR 30-100  AGI 40-110  QUI 40-110  CON 30-100  PER 40-110  WIL 30-100  EMP 30-100',
    desc: 'Strong humanoids of medium height with distinctly lupine features. They can travel long distances without fatigue and see at night as well as in day. A proud, noble race where honor is extremely important. Calling a wolfling a werewolf is a dangerous proposition.',
    ability: 'TRANSFORM into giant wolf form — fight with claws and teeth, travel long distances.',
  },
  {
    id: 5, name: 'Murg',
    stats: 'STR 40-110  AGI 30-100  QUI 30-100  CON 40-110  PER 40-110  WIL 20-90  EMP 20-90',
    desc: 'Burly humanoids naturally endowed with a desire to compete. Powerful limbs make them skilled climbers. Their eyes can see nearly as well in darkness as daylight. Compulsive practical jokers, they find themselves extremely funny. Broken into clans that constantly fight each other.',
    ability: 'FRENZY — strike with greater ferocity; continue fighting even below 0 HP until death.',
  },
  {
    id: 6, name: 'Drakin',
    stats: 'STR 40-110  AGI 10-80  QUI 40-110  CON 40-110  PER 30-100  WIL 30-100  EMP 40-110',
    desc: 'Dragon-folk covered with tough scales, powerful jaws, reptilian tails and large bat-like wings. They move with lightning speed when necessary. Their culture developed its own school of fighting with unique weapons: katana, wakizashi, sai, kusari-gama, nunchaku, bo-stick, naginata, shuriken. Never wear armor.',
    ability: 'FLY with their wings. Unique drakin weapon style combinable with two-weapon fighting.',
  },
  {
    id: 7, name: 'Mechanoid',
    stats: 'STR 40-110  AGI 30-100  QUI 30-100  CON 40-110  PER 40-110  WIL 30-100  EMP 1-60',
    desc: 'Machines imbued with an ephemeral spirit. Regenerating living tissue covers their mechanical bodies. The least empathic of all races, making the worst spellcasters, but they can turn emotions on and off at will to channel skills to maximum effectiveness. Pragmatic and logical to annoyance.',
    ability: 'EMOTE/UNEMOTE — toggle emotional state for precision skill use.',
  },
  {
    id: 8, name: 'Ephemeral',
    stats: 'STR n/a  AGI 30-100  QUI 50-120  CON 1-10  PER 30-100  WIL 30-100  EMP 30-100',
    desc: 'Wispy apparitions that do not exist completely in the physical plane. Non-magical weapons cannot harm them except when they attack, which draws them onto the physical plane. They cannot manipulate heavy objects. Natural telepaths who usually devote their lives to scholarly pursuits.',
    ability: 'Immune to non-magical weapons (except when attacking). Natural telepathy. See in darkness.',
  },
]

interface Props {
  onCreated: (char: { firstName: string; lastName: string; race: number; gender: number }) => void
  onOpenManual?: () => void
}

export default function CharacterCreate({ onCreated, onOpenManual }: Props) {
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [race, setRace] = useState(1)
  const [gender, setGender] = useState(0)
  const [selectedRace, setSelectedRace] = useState(RACES[0])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!firstName.trim() || !lastName.trim()) return
    onCreated({ firstName: firstName.trim(), lastName: lastName.trim(), race, gender })
  }

  return (
    <div className="flex items-start sm:items-center justify-center h-full pt-4 px-4 pb-4 sm:p-8 overflow-y-auto">
      <div className="max-w-3xl w-full bg-[#111] border border-[#333] rounded-lg p-4 sm:p-8">
        <h2 className="text-amber-400 text-2xl font-mono mb-1 text-center">
          Create Your Character
        </h2>
        <p className="text-gray-500 text-sm font-mono mb-6 text-center">
          Choose wisely — your race and abilities will shape your destiny in the Shattered Realms
        </p>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Name */}
          <div className="bg-[#0a0a0a] border border-amber-900/50 rounded-lg p-3 mb-2">
            <p className="text-gray-400 text-xs font-mono leading-relaxed">
              Legends is a roleplaying game &mdash; please choose a name that fits the fantasy setting.
              Avoid modern names, pop culture references, or joke names.{' '}
              <button type="button" onClick={onOpenManual} className="text-amber-500 hover:text-amber-400 underline cursor-pointer">
                Read more about roleplaying &rarr;
              </button>
            </p>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-gray-400 text-sm font-mono mb-1">First Name</label>
              <input
                type="text"
                value={firstName}
                onChange={e => setFirstName(e.target.value)}
                maxLength={15}
                className="w-full bg-[#0a0a0a] border border-[#444] rounded px-3 py-2 text-gray-200 font-mono focus:border-amber-500 focus:outline-none"
                placeholder="Balthazar"
                autoFocus
              />
            </div>
            <div>
              <label className="block text-gray-400 text-sm font-mono mb-1">Last Name</label>
              <input
                type="text"
                value={lastName}
                onChange={e => setLastName(e.target.value)}
                maxLength={15}
                className="w-full bg-[#0a0a0a] border border-[#444] rounded px-3 py-2 text-gray-200 font-mono focus:border-amber-500 focus:outline-none"
                placeholder="Thinvar"
              />
              <p className="text-gray-600 text-[10px] font-mono mt-1">Hyphens and accents permitted in last name</p>
            </div>
          </div>

          {/* Gender */}
          <div>
            <label className="block text-gray-400 text-sm font-mono mb-1">Gender</label>
            <div className="flex gap-4">
              {[{ v: 0, l: 'Male' }, { v: 1, l: 'Female' }].map(g => (
                <button
                  key={g.v}
                  type="button"
                  onClick={() => setGender(g.v)}
                  className={`px-6 py-2 rounded font-mono text-sm transition-colors ${gender === g.v ? 'bg-amber-700 text-white' : 'bg-[#1a1a1a] text-gray-400 border border-[#444] hover:border-amber-600'}`}
                >
                  {g.l}
                </button>
              ))}
            </div>
          </div>

          {/* Race selection */}
          <div>
            <label className="block text-gray-400 text-sm font-mono mb-2">Race</label>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 mb-3">
              {RACES.map(r => (
                <button
                  key={r.id}
                  type="button"
                  onClick={() => { setRace(r.id); setSelectedRace(r) }}
                  className={`px-2 py-2.5 rounded font-mono text-sm transition-colors ${race === r.id ? 'bg-amber-700 text-white border border-amber-600' : 'bg-[#1a1a1a] text-gray-400 border border-[#444] hover:border-amber-600'}`}
                >
                  {r.name}
                </button>
              ))}
            </div>

            {/* Race detail */}
            <div className="bg-[#0a0a0a] border border-[#333] rounded-lg p-4 space-y-3">
              <div className="flex items-center gap-3">
                <h3 className="text-amber-400 font-mono text-lg font-bold">{selectedRace.name}</h3>
              </div>
              <p className="text-gray-300 font-mono text-xs leading-relaxed">{selectedRace.desc}</p>
              <div className="bg-[#111] border border-[#2a2a2a] rounded p-2">
                <p className="text-green-400 font-mono text-xs">
                  <span className="text-gray-500">Ability:</span> {selectedRace.ability}
                </p>
              </div>
              <div className="bg-[#111] border border-[#2a2a2a] rounded p-2">
                <p className="text-cyan-400 font-mono text-[10px] tracking-wider">{selectedRace.stats}</p>
              </div>
            </div>
          </div>

          <button
            type="submit"
            disabled={!firstName.trim() || !lastName.trim()}
            className="w-full py-3 bg-amber-700 hover:bg-amber-600 disabled:bg-gray-700 disabled:text-gray-500 text-white font-mono rounded text-lg transition-colors cursor-pointer"
          >
            Enter the Shattered Realms
          </button>
        </form>
      </div>
    </div>
  )
}
