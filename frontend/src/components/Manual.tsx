import { useState, useEffect, useRef } from 'react'

interface Section {
  id: string
  title: string
  level: number
}

const sections: Section[] = [
  { id: 'introduction', title: 'Introduction', level: 2 },
  { id: 'city-of-fayd', title: 'The City-State of Fayd', level: 2 },
  { id: 'history', title: 'History', level: 3 },
  { id: 'money', title: 'Money', level: 3 },
  { id: 'extraordinary-substances', title: 'Extraordinary Andoran Substances', level: 3 },
  { id: 'your-character', title: 'Your Character', level: 2 },
  { id: 'people-of-the-realms', title: 'People of the Realms', level: 2 },
  { id: 'statistics', title: 'Statistics', level: 2 },
  { id: 'advancement', title: 'Advancement', level: 2 },
  { id: 'organizations', title: 'Organizations', level: 2 },
  { id: 'religious-organizations', title: 'Religious Organizations', level: 3 },
  { id: 'skills', title: 'Skills', level: 2 },
  { id: 'player-interaction', title: 'Player Interaction', level: 2 },
  { id: 'settings', title: 'Settings', level: 3 },
  { id: 'movement', title: 'Movement', level: 3 },
  { id: 'command-glossary', title: 'Command Glossary', level: 2 },
  { id: 'other-verbs', title: 'Other Verbs', level: 3 },
  { id: 'combat', title: 'Combat', level: 2 },
  { id: 'armor', title: 'Armor', level: 3 },
  { id: 'missile-thrown-weapons', title: 'Missile and Thrown Weapons', level: 3 },
  { id: 'backstab', title: 'Backstab', level: 3 },
  { id: 'death', title: 'Death', level: 3 },
  { id: 'psionics', title: 'Psionics', level: 2 },
  { id: 'spellcasting', title: 'Spellcasting', level: 2 },
  { id: 'conjuration-spells', title: 'Conjuration Spells', level: 3 },
  { id: 'enchantment-spells', title: 'Enchantment Spells', level: 3 },
  { id: 'necromancy', title: 'Necromancy', level: 3 },
  { id: 'general-magic', title: 'General Magic', level: 3 },
  { id: 'druidic-spells', title: 'Druidic Spells', level: 3 },
  { id: 'art-of-roleplaying', title: 'The Art of Roleplaying', level: 2 },
  { id: 'game-policies', title: 'Game Policies', level: 2 },
]

export default function Manual({ onBack }: { onBack: () => void }) {
  const [activeSection, setActiveSection] = useState('introduction')
  const [tocOpen, setTocOpen] = useState(false)
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const container = contentRef.current
    if (!container) return
    const headings = container.querySelectorAll('h2[id], h3[id]')
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveSection(entry.target.id)
            break
          }
        }
      },
      { root: container, rootMargin: '-10% 0px -80% 0px', threshold: 0 }
    )
    headings.forEach((h) => observer.observe(h))
    return () => observer.disconnect()
  }, [])

  const scrollTo = (id: string) => {
    const el = document.getElementById(id)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
      setActiveSection(id)
      setTocOpen(false)
    }
  }

  return (
    <div className="h-full flex flex-col bg-[#0a0a0a]">
      {/* Mobile TOC toggle */}
      <div className="lg:hidden flex items-center justify-between px-4 py-2 border-b border-[#333] bg-[#111]">
        <span className="text-amber-500 font-mono font-bold text-sm">Player Manual</span>
        <button onClick={onBack} className="px-3 py-1 bg-[#333] hover:bg-[#444] text-gray-300 font-mono text-sm rounded border border-[#555]">
          &times; Close
        </button>
        <button
          onClick={() => setTocOpen(!tocOpen)}
          className="text-gray-400 hover:text-amber-400 font-mono text-sm"
        >
          {tocOpen ? 'Close' : 'Contents'}
        </button>
      </div>

      {/* Mobile TOC dropdown */}
      {tocOpen && (
        <div className="lg:hidden bg-[#111] border-b border-[#333] max-h-[60vh] overflow-y-auto px-4 py-3">
          <nav className="space-y-1">
            {sections.map((s) => (
              <button
                key={s.id}
                onClick={() => scrollTo(s.id)}
                className={`block w-full text-left font-mono text-xs py-1 transition-colors ${
                  s.level === 3 ? 'pl-4' : ''
                } ${
                  activeSection === s.id
                    ? 'text-amber-400'
                    : 'text-gray-500 hover:text-gray-300'
                }`}
              >
                {s.title}
              </button>
            ))}
          </nav>
        </div>
      )}

      <div className="flex-1 flex overflow-hidden">
        {/* Desktop sidebar TOC */}
        <aside className="hidden lg:block w-64 flex-shrink-0 border-r border-[#333] bg-[#111] overflow-y-auto">
          <div className="sticky top-0 bg-[#111] border-b border-[#222] px-4 py-3">
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-amber-500 font-mono font-bold text-sm">Player Manual v3.1</h2>
              <button
                onClick={onBack}
                className="px-3 py-1 bg-[#333] hover:bg-[#444] text-gray-300 font-mono text-xs rounded border border-[#555]"
              >
                &times; Close
              </button>
            </div>
          </div>
          <nav className="px-4 py-3 space-y-0.5">
            {sections.map((s) => (
              <button
                key={s.id}
                onClick={() => scrollTo(s.id)}
                className={`block w-full text-left font-mono text-xs py-1 transition-colors rounded px-2 ${
                  s.level === 3 ? 'pl-5' : ''
                } ${
                  activeSection === s.id
                    ? 'text-amber-400 bg-amber-900/20'
                    : 'text-gray-500 hover:text-gray-300 hover:bg-[#1a1a1a]'
                }`}
              >
                {s.title}
              </button>
            ))}
          </nav>
        </aside>

        {/* Main content */}
        <div ref={contentRef} className="flex-1 overflow-y-auto scroll-smooth">
          <div className="max-w-3xl mx-auto px-6 py-8 font-mono text-sm">
            {/* Header */}
            <div className="text-center mb-10">
              <h1 className="text-amber-500 text-2xl font-bold mb-2">
                Legends of Future Past
              </h1>
              <p className="text-gray-500 text-xs">
                Player Manual V3.1 &mdash; &copy;1994 Inner Circle Technologies
              </p>
            </div>

            {/* Introduction */}
            <h2 id="introduction" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Introduction
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                Welcome to the Shattered Realms! A place of art and science,
                wizards and warriors, beauty and romance, magic and technology,
                mutants and monsters.
              </p>
              <p>
                Andor is our planet, a small part of which makes up the Shattered
                Realms. Around us is a vast desert wasteland, the legacy of an
                ancient Cataclysm that destroyed the once highly technological
                society that thrived here.
              </p>
              <p>
                In the Realms, life has taken hold again. You live in the City of
                Fayd, an old metropolis predating the Cataclysm. There are many
                ancient places here including catacombs beneath the city itself, and
                vast subterranean areas far beneath the surface.
              </p>
              <p>
                To the north lies a great range of mountains known as the
                Teeth of Shartan, it too, is honeycombed with caves. Rolling hills
                and grassy plains, as well as the great Inner Sea, lay to the east
                and south. The great forest of Grymwood stands to the west. Only the
                dangerous and forbidding Great Waste lies beyond these borders.
              </p>
              <p>
                A variety of creatures populate the Shattered Realms. The most
                common is the elder race: humans. Their forefathers lived alone in
                the pre-Cataclysmic world. Now, humans share the world with the
                aelfen of the forests, the highlanders of the mountain, the murg
                of the caves, and the drakin of the skies. In addition, you will
                meet wolflings, a race of shape changers that can assume human or
                wolf form; ephemerals, a race of noncorporeal entities and their
                relatives the mechanoids, a race of robots imbued with a spirit.
                You may even encounter those with the arcane knowledge of becoming
                a vampire. Besides these, the world is rich in flora and fauna;
                thousands of creatures, both good and evil, beautiful and monstrous,
                inhabit the land.
              </p>
              <p>
                The Realms are rich in magic, the existence of which became known
                at the time of the Change. Sorcerers everywhere ply their mystic
                talents in four basic schools: druidic, enchantment, necromancy and
                conjuration. Others delve into history to find the clues that would
                allow them to understand the most ancient of all magic &mdash; technology.
              </p>
              <p>
                Cybernetics, formerly a lost science, has regained popularity,
                since it provides humans with an improvement over biology.
                Psionics, the methodology of mind over matter, has also come to the
                fore, as those with great force of will seek to bend nature to
                their course.
              </p>
            </div>

            {/* The City-State of Fayd */}
            <h2 id="city-of-fayd" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              The City-State of Fayd
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                Fayd is built within the walls of an ancient, pre-cataclysmic
                city now surrounded by rolling hills and grasslands on all sides.
                The Teeth of Shartan, a great mountain range, lies to the north.
                Some say it is the location of the lost city of Mirdaal. Northwest
                is the dense Grymwood forest. To the south are the Inner
                Sea, the entrance to mysterious caverns and the wasteland.
                Beyond these, nothing more is known to the people of the
                Realms, since it is several hundred leagues inland from an ocean
                and twice as far from another city-state. Fayd is ruled by a town
                government of appointed and elected officials who run the day to
                day mercantile and law-enforcement operations. The Royal Government
                creates new laws and declares war.
              </p>
            </div>

            {/* History */}
            <h3 id="history" className="text-amber-300 text-base font-bold mt-8 mb-3">
              History
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                According to the teachings of the oral historians, just over
                a millennium ago, the planet of Andor was a much different place.
                Desert was almost nonexistent, and humans were the only intelligent
                race to walk the land. Humankind developed great sciences,
                engineering the means to improve body and mind. It is said that
                they built ships that could sail away from Andor and into the stars.
              </p>
              <p>
                All this came to an end following a terrible event known as
                the Cataclysm. Oral history chronicles the horror of fiery rocks
                streaming from the heavens, pummeling the land and destroying much
                of the surface of the world. It also tells of Andorans who knew
                what was coming and prepared. A tale is told of a chosen few, who
                escaped the planet on silvery starships. Nothing has been learned
                of what became of these people. There are rumors that others built
                great complexes beneath the ground in which they remained safe from
                the destruction. Most unfortunate Andorans simply took shelter in
                the surface cities, few survived.
              </p>
              <p>
                The Cataclysm tore the world apart, and chaos spread like wildfire.
                Soon, armies of men formed, and within the span of a decade, a
                hundred wars were fought to control what remained. Epidemics of
                conflict spread untold destruction across the land. Eventually, the
                tattered remnants of mankind lacked either the willpower or the
                means to continue their internecine conflict. Indeed, there was
                little to fight over. Then, the Change occurred. No one knew what
                caused the Change. Some say it was material that fell from the sky
                at the onset of the Cataclysm. Others blame the terrible weapons
                used by the humans in their wars. The religious often feel that the
                massive death toll released fundamental energies into the world.
                Maybe it was all three of these or none.
              </p>
              <p>
                During the Change, oral historians tell us some life forms
                remained the same while others experienced millions of years of
                evolution in a single century. It was this acceleration that
                created the new races now sharing the Realms with humans and
                animals. At the same time, the Change awoke untapped powers in all
                the people of the Realms. Some learned to channel the energy of
                the mind to perform feats of destruction. Others found they could
                touch upon the primal forces of the universe and bend the physical
                laws to their own will.
              </p>
              <p>
                For many centuries after the Cataclysm and the Change, Andor was in
                a great dark age. The works of the Elder Humans were lost. All that
                is known and believed today has been passed down in the prose of
                oral historians. The scattered remnants of mankind and their new
                neighbors struggled to relearn the basics of civilization such as
                farming, domesticating livestock and building simple communities.
                However, parts of pre-cataclysmic Fayd and other structures remain,
                standing like grim monuments of the past. Such places were viewed
                with fear and few were daring enough to explore them. These early
                years were dangerous, and the people were ripe for the conquering
                by powerful users of magic. The most noteworthy figure in the
                history of early Fayd is the tale of the dread necromancer
                Malthesia. In the year 424, he came across the Great Waste and
                arrived in Fayd with great riches and powerful artifacts. None knew
                of his true origins, and none dared to guess how he survived a
                trek across the Waste alone. Using magic and his wealth, he
                constructed a keep which was completed as soon as 426. Many spoke
                of horrible experiments Malthesia was perpetrating within his keep,
                and some accused him of summoning demons and creating undead
                horrors. Yet these voices were stifled and ignored, and in the year
                427 a host of evil fell upon the city of Fayd. After two weeks of
                brutal slaughter, the tales speak of how the great warrior, Orlin
                Thinvar, set out with his magic sword Sunstrike, and penetrated the
                keep to confront the Necromancer. No one knows how the battle went,
                but as the night approached, an inhuman roar emanated from the Keep
                that shook the very foundations of Fayd, and Malthesia&apos;s demonic
                generals were sent back to whence they came. Today, a sickly green
                mist still surrounds the Keep, preventing entry, but the
                Necromancer is no more.
              </p>
              <p>
                The first exploration into the ancient lore of technology occurred
                just two hundred and fifty years ago and ended in disaster and
                bloodshed. Alkaryn, a great warlord, discovered a huge cache of
                powerful, ancient weapons in an area near the city. He swiftly
                used them to forge a vast empire, which he ruled with an iron fist.
                Within a few years however, rebellions sprang up across his domain.
                Many of his foe sought the ancient technologies in order to counter
                Alkaryn. This culminated in the Great Technomancy War of 783, and
                lead to the deaths of tens of thousands. The war resulted in the
                eradication of most of the ancient technology as well as the
                displacement of millions.
              </p>
              <p>
                The mechanoid race came to be around this time. They originated as
                ephemerals, who for the purpose of battling technology, took on
                mechanical form. During the many years of the war, their leaders
                too were destroyed. The secrets of mechanoid technology were lost.
                Today&apos;s mechanoids can replicate the original patterns to create
                new members of their race, but they cannot improve upon or
                add to their technology. Thus the mechanoid race, the greatest
                legacy of the war and a seeming anomaly in the Realms, cannot be
                understood even by themselves. Alkaryn was finally repulsed when
                the emperor Regulus rose to power in 789. Regulus united the
                fractured rebel forces against Alkaryn, and swept his armies from
                Fayd. Regulus himself was known to have used great weapons of mass
                destruction, but they were never seen again after these last
                battles.
              </p>
              <p>
                With the overthrow of Alkaryn, people expected a happier
                life, however, Regulus proved to a cruel dictator who would stand
                for no resistance and no inefficiency. Those who were not deemed
                worthy members of the society were eliminated with a wave of a
                hand. Regulus&apos; previous allies slowly turned against him. His end
                came upon his confrontation with the young warrior-wizard, Frederick
                in 795.
              </p>
              <p>
                Frederick used spells of great power and ejected Regulus from Fayd.
                Regulus has not been seen since. King Frederick was said to be kind
                and just, and his rule is known as the Golden Age of Fayd. He
                retained his youth to an extremely old age. At age 120 (year 902),
                despite his health, he passed the throne to his son, Frederick II.
                Frederick then left for a journey to the Teeth of Shartan from which
                he has not yet returned. Frederick II lived to a similarly old age
                before dying in year 976 and passing the Kingship to Frederick III.
                Frederick III was recently assassinated along with most of his family.
                His son Prince Hadan, now king Hadan, rules Fayd and its surrounding
                area.
              </p>
            </div>

            {/* Money */}
            <h3 id="money" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Money
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                The currency metals are mined from the Teeth of Shartan and
                transported to Fayd for minting into gold crowns, silver shillings
                and copper pennies. The average income of a working-class citizen
                of Fayd is 3 crowns/week, and the average income for an
                artisan/merchant-class citizen is 20 crowns/week.
              </p>
              <p>
                The currency of the Deep Realms is called a Kragenmark. You can find
                Kragenmarks on Deep Realms creatures and use them to buy goods in the
                Deeplander city of Kragenholm. They are worth about 1/10th of a gold
                crown.
              </p>
              <p>
                The last form of currency is called Shards. This is the currency of the
                outer planes. It is found there and used there as well. Shards are good
                for nothing but decoration on Andor and are worth about 1/20th of a gold
                crown.
              </p>
            </div>

            {/* Extraordinary Andoran Substances */}
            <h3 id="extraordinary-substances" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Extraordinary Andoran Substances
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                <span className="text-green-400">Alzyron</span> is the hardest and most indestructible substance known in
                the Realms. All the Alzyron in the world was produced in the days
                before the Cataclysm: the art of creating alzyron has been lost.
                It appears to be a composite of synthetic diamond and titanium
                steel micro filaments. It has strong anti-magic properties.
                Alzyron is so rare that a chest full of the material would easily
                out value all the wealth in Fayd.
              </p>
              <p>
                <span className="text-green-400">Randar</span> is an extremely rare volcanic substance. It is quite strong
                and can be formed into tools and weapons by highly skilled
                craftsmen. It looks like a heavy, black metal.
              </p>
              <p>
                <span className="text-green-400">Elkyri</span>, also known as Truesteel, is a rare metal found deep within
                the mountains. It is valued for its exceptional strength and
                resiliency. A small percent of Truesteel has been tainted by the
                unholy touch of Dahkahn, and is known as Black Truesteel or
                Shadowsteel. Fortunately, normal elkyri can be blessed and
                purified through a long process to produce White Truesteel. Also
                called Albescent, it is stronger than normal Truesteel and as
                unsoiled as the Black Truesteel is tainted.
              </p>
              <p>
                <span className="text-green-400">Urimar</span>, highly prized for its lightweight resiliency, can be
                found in small quantities in the hearts of certain kinds of great
                oak. It is hard to come by because there is no easy way to
                identify the exact tree it might exist in. Additionally, most
                aelfen will protect these trees if they learn of them, keeping the
                sites secret. Urimar is found in a clump at the center of a stump
                and has a coloring strange for metal: dark red, olive green or deep
                purple. It is possible to work the substance into superior weapons,
                tools and armor, though it requires a knowledgeable smith. Urimar
                is reputed to have a neutralizing effect on poison.
              </p>
              <p>
                The original <span className="text-green-400">modo</span> was made from the tooth of a dragon. Tales are
                told of a heroic aelfen warrior, Anderon the Red, and his deadly
                modo dagger. The modo daggers found in the Realms are not true
                modo, but a synthetic substance made of volcanic glass and flecks
                of elkyri. The material is as iridescent as the legendary modo,
                but it lacks any magical quality.
              </p>
              <p>
                <span className="text-green-400">Fiery and Icy metals</span> can be mined by lucky and experienced miners.
                These metals, besides being fairly aesthetic, can imbue weapons
                with fire and ice properties.
              </p>
            </div>

            {/* Your Character */}
            <h2 id="your-character" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Your Character
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                The first time you play Legends of Future Past, you will have the
                fun of creating your premier character. We suggest you don&apos;t
                worry about it too much the first time and just create any old
                character. After you tour the Realms for a bit, you can always
                create a new character and you can have up to three characters per
                account.
              </p>
              <p>
                When rolling a character you will be required to choose a full
                name (first and last). You may have up to fifteen letters in each
                one. Punctuation such as hyphens and accents are permitted in your
                last name only. We recommend you choose a name that suits the
                fantasy genre. Legends of Future Past will prompt you for a
                different name if the one you choose is already in use.
              </p>
              <p>
                After choosing a name, you will first decide what race your
                character will belong to, your decision will affect certain special
                abilities and physical attributes. (See People of the Realms)
              </p>
              <p>
                Next, your character&apos;s statistics are generated randomly on a
                one-hundred sided die. After being modified by your chosen race,
                the numbers are displayed and you are queried for approval. You can
                continue to reroll these statistics until you like what you see.
                (See Stats for greater detail about statistics.)
              </p>
              <p>
                Lastly, you will be granted 30 build points to allocate for your
                initial skills. You will be able to learn these skills from guilds,
                organizations and other characters within the City of Fayd. Choose
                wisely, for these abilities will greatly affect your survivability
                as a low-level character and your power later on. Once you have
                made your choice, the decision is final. In the case of a typo,
                you may type ASSIST.
              </p>
              <p>
                Head for the Bazaar Center when you are new to Fayd. Adventurers
                can usually be found there with time enough to show you around.
                It is one of the king&apos;s strictest laws not to harass new Faydians
                in any way.
              </p>
            </div>

            {/* People of the Realms */}
            <h2 id="people-of-the-realms" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              People of the Realms
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                A number of fascinating races populate the Shattered Realms.
                Each has its own advantages and drawbacks.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Aelfen</h4>
              <p>
                The aelfen are tall, slender humanoids that could be mistaken
                for humans at a distance. However, their pretty features and
                pointed ears give them away as one draws closer. The aelfen are
                quick, agile and have heightened senses from living in the
                woodlands. They may live for centuries and are extremely resistant
                to natural diseases. Aelfen have the power to <span className="text-amber-300">CALL</span> a woodland
                creature to serve them. This works only in wilderness. The aelfen
                are carefree and aloof with mercurial tempers. They are quick to
                love or hate, but they almost never hold a grudge. They dislike
                violence but are prepared to do what they must in times of need.
                They are lovers of beauty and the arts and they usually adore
                magic. They dislike murg who they find ugly, crude and wasteful of
                natural resources. They also distrust drakin, because they too
                seem ugly to them.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Drakin</h4>
              <p>
                Also known as dragon-men, drakin are humanoids covered with tough
                scales. They have powerful jaws, reptilian tails and large bat-like
                wings that allow them to <span className="text-amber-300">FLY</span>. Many people have made the mistake of
                thinking the drakin to be slow &mdash; they move little unless they need
                to &mdash; and learned too late that drakin can move with lightning
                speed when necessary. The drakin culture has developed its own
                school of fighting, employing techniques that differ from the
                fighting styles of any other in the land. This caused the drakin
                to invent new weapons specifically designed to work well with their
                fighting style. They are generous with their knowledge, and anyone
                can learn drakin weapons. Drakin weaponry includes the katana,
                wakizashi, sai, kusari-gama, nunchaku, bo-stick, naginata and
                shuriken. Drakin style may be combined with two weapon fighting.
                Drakin never wear armor and great swings in temperature are more
                deadly to them than other races. The common speech of the Realms
                is difficult for drakin, so they avoid speaking altogether. When
                they have something to say it is usually important or wise. They
                are highly aggressive, but battles between one drakin and another
                outside of tournaments are nearly unheard of. In fact, should one
                drakin dare to threaten another with physical harm, that individual
                may be subject to excommunication from their race. The drakin have a
                rich history of ancestor worship.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Ephemeral</h4>
              <p>
                Mystery shrouds the origin of the ephemeral race. Described as
                phantoms, ephemerals are wispy apparitions of human-like people.
                They do not exist completely in the physical plane. As a result,
                non-magical weapons do not harm them except when they themselves
                go on the offense with weapons, magic or psionics. This draws
                them onto the physical plane and causes them to become vulnerable
                to mundane attacks. Still, they do not bleed. Because they only
                have semi-substance, they cannot manipulate objects that weigh more
                than a couple of pounds. Ephemerals can see well in the dark.
                Besides being able to speak, they are also natural telepaths.
                Ephemerals are not a violent people and usually devote their lives
                to scholarly pursuits. They regard mechanoids coldly for forsaking
                their existence as ephemerals.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Highlander</h4>
              <p>
                The highlanders are short and rugged people who live in and on the
                mountains. Like murg, they can see well in darkness. They are
                strong, enduring and are especially resistant to magic. Similarly,
                magic is not easy for them to use. On the other hand, they are
                very comfortable with technological implements and devices. Known
                as Children of the Stone, highlanders wield special powers over the
                elements of earth. When in caves, they employ a chameleon-like
                power. When they <span className="text-amber-300">BLEND</span> into the rocks around them, they become
                harder to attack. At fifth level they gain the ability to hold a
                gemstone of any sort in their hands and <span className="text-amber-300">MOLD</span> it into a more
                valuable gem. Occasionally this backfires and ruins a gem.
                Highlanders are a robust people who often make good fighters and
                psionicists. Their women can be quite attractive contrary to the
                jokes made about them, and they certainly do not have beards.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Mechanoid</h4>
              <p>
                The roots of the mechanoid race originate in the Great Technomancy
                War of 783, when a group of ephemerals fighting for survival sought
                a more substantial and robust existence by constructing robotic
                housings for their essence. These creatures were the original
                mechanoids: machines imbued with an ephemeral spirit. Regenerating
                living tissue covers their mechanical bodies. Due to the type of
                existence they have chosen, mechanoids are the least empathic of
                all races and thus make the worst spellcasters. However, they can
                turn their emotions on and off at will, thus channeling their
                skills and powers to maximum effectiveness. Mechanoids are often
                seen as cold and unfeeling since they are machines in body, if not
                in spirit. Their ability to turn off emotions does not help this
                image (see <span className="text-amber-300">EMOTE</span> and <span className="text-amber-300">UNEMOTE</span>). They regard other life forms,
                especially true ephemerals, as inferior. They have a tendency to be
                pragmatic and logical to annoyance.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Murg</h4>
              <p>
                Murg are burly humanoids naturally endowed with a desire to
                compete. They are enduring, strong, and vicious warriors. Their
                powerful limbs make them skilled climbers. Their homeland is deep
                in the mountains. Their eyes have gained sensitivity in darkness,
                allowing them to see nearly as well as in daylight, giving birth to
                the expression, &ldquo;I&apos;d sooner meet a murg in a dark alley.&rdquo; Murg can
                enter a state of battle <span className="text-amber-300">FRENZY</span>. This allows them to strike out
                with greater ferocity, but places them at greater risk of becoming
                injured. Further, if they are injured while frenzied, even if that
                injury would normally render them unconscious, they can continue
                to fight on until they literally die standing up. The murg as a
                whole tend to be quarrelsome and crude. They are compulsive
                practical jokers, and they find themselves extremely funny. Don&apos;t
                take an insult from a murg too seriously. Murg are broken up into
                clans that constantly fight each other (even more so than humans
                fight amongst themselves). They believe that exposing themselves
                to constant battles will increase the strength of their people.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Wolfling</h4>
              <p>
                Strong humanoids of medium height, wolflings look like humans with
                distinctly lupine features and greater than normal amounts of body
                hair. They can polymorph into a giant wolf (see command <span className="text-amber-300">TRANSFORM</span>).
                They can travel long distances without fatigue and battle using
                their claws and teeth when in wolf form. Wolflings see at night as
                well as in day as long as there is at least starlight. Quickness
                and agility are the hallmarks of a wolfling. Wolflings are a proud,
                noble race. Honor is extremely important to them. Above all else
                they hate comparisons to werewolves (a totally corrupt and cursed
                monster which shares similar abilities). Calling a wolfling a
                werewolf is a dangerous proposition: it could instigate the race
                as a whole to do something to protect the honor and integrity of
                their people. The wolflings have a homeland in the forest and
                frequently go there to leave the bustling city.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Human</h4>
              <p>
                The elder race of the Shattered Realms, humans have existed since
                time out of mind. They are physically average, but tend to excel at
                any skill they pursue. This is attributed to their superior
                willpower. Cybernetic technology was originally developed for
                humans, so they may employ any such device. There is no easy way
                to describe the personality of humans as a whole. They all act
                differently, but most feel individualism is important. They get
                along with other races as well as they get along with each other:
                humans will battle or befriend others seemingly at random. Humans
                have an extraordinary tendency to fight each other, despite the
                detriment to their race.
              </p>
            </div>

            {/* Statistics */}
            <h2 id="statistics" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Statistics
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                Your character has seven statistics that define his or her
                abilities. The average range for these statistics is 30-100. This
                may change based on your race. Wealthier characters may raise these
                stats by purchasing and eating various herbs. Naturally this works
                only for races that have the ability to eat.
              </p>
            </div>
            <div className="my-4 overflow-x-auto">
              <pre className="text-green-400 text-xs leading-snug bg-[#111] border border-[#333] rounded p-4">{`Initial Racial Statistics Ranges
Race        Strength  Agility   Quick   Const    Per     Will      Emp
Aelfen        20-90   40-110  40-110    1-70   40-110  30-100   40-110
Drakin        40-110   10-80  40-110   40-110  30-100  30-100   40-110
Ephemeral      n/a    30-100  50-120    1-10   30-100  30-100   30-100
Highlander    40-110   20-90   20-90   50-120  30-100  30-100    10-80
Human         30-100  30-100  30-100   30-100  30-100  40-110   30-100
Mechanoid     40-110  30-100  30-100   40-110  40-110  30-100     1-60
Murg          40-110  30-100  30-100   40-110  40-110   20-90    20-90
Wolfling      30-100  40-110  40-110   30-100  40-110  30-100   30-100`}</pre>
            </div>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                <span className="text-green-400">Strength:</span> Physical muscle power and the ability to make good use of the
                power you have. The stronger you are, the more you can carry and
                lift before you begin to fatigue, also, greater damage you do with
                melee weapons and the more power you can apply in situations such
                as forcing a jammed door open.
              </p>
              <p>
                <span className="text-green-400">Agility:</span> Dexterity, litheness, aim, nimbleness. The more agile you are, the
                greater your ability to perform activities requiring stealth. Agile
                people aim their weapons more precisely and make better climbers.
              </p>
              <p>
                <span className="text-green-400">Quickness:</span> General speed and reaction time. Quick people have a better
                chance of avoiding a trap just as it springs, can move faster,
                dodge attacks better and have shorter round times for physical
                activities.
              </p>
              <p>
                <span className="text-green-400">Constitution:</span> Physical endurance, health, resistance. Those with high
                constitution are less susceptible to disease and poison. They may
                perform strenuous activity for longer amounts of time without rest,
                and can take more physical damage before collapsing. Constitution
                directly determines how many body points you have, which will
                determine how much harm you can take before dying or going
                unconscious. Lastly, constitution directly affects the amount of
                fatigue points gained when making a level and the rate at which
                fatigue points return.
              </p>
              <p>
                <span className="text-green-400">Perception:</span> This is the measure of how well honed your natural senses are
                and the ability to pick out small details. The greater your
                perception, the higher your chance of avoiding surprise attacks,
                finding hidden objects, locating and disarming traps, finding
                secret portals, etc.
              </p>
              <p>
                <span className="text-green-400">Willpower:</span> This is a measure of mental strength and the ability to push
                your physical powers to their limit. Willpower is important in
                performing and resisting psionic attacks. It factors directly into
                how much psi you have, which limits how much psionic power you can
                exert in a given amount of time.
              </p>
              <p>
                <span className="text-green-400">Empathy:</span> Your ability to attune yourself to the emotions of others and
                touch upon the fundamental forces of nature with your inner self.
                Empathy allows people to wield the forces of magic, and directly
                factors into how much mana you have, which limits how many spells
                you can cast during a given period.
              </p>
            </div>

            {/* Advancement */}
            <h2 id="advancement" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Advancement
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                You accumulate experience points as you perform skills, defeat
                enemies, cast spells, solve puzzles and make items. For every amount of
                experience points you gain, based on your level, you receive a
                build point toward purchasing skills and increasing the statistics
                of your character. You gain a level every 10 build points (whether
                you spend those points or not). You may spend build points
                immediately in the middle of your advancement through a level, or
                save them up to acquire costly skills (some skills may require
                several levels to develop). Training after a character reaches
                fourth level costs five gold crowns per training level. Characters
                may teach each other skills to avoid this cost.
              </p>
            </div>

            {/* Organizations */}
            <h2 id="organizations" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Organizations
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                In any world it&apos;s not what you know, but who you know. The City of
                Fayd has provided the Adventurers&apos; Guild to its citizens in
                order to encourage more adventuring, which is both good for the
                economy and protects the city from evil in a cost effective manner.
                The Adventurers&apos; Guild trains in most skills up to level ten. After
                that, those who wish specialized training, must seek it out from
                private sources. There are several organizations and churches to
                choose from. Rank within an organization depends on how much
                training you do there.
              </p>
              <p>
                Some organizations are open to the public, while others are by
                invitation only. Invitations are based on roleplaying. If you
                desire membership in one of these organizations, you should ask
                around and try to find other members of the guild. They will tell
                you what you need to do to become a member. You may be a member of
                only one organization at a time.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Adventurer&apos;s Guild</h4>
              <p>
                This is a neutral organization of intrepid thrill-seekers and
                treasure-hunters. It offers education in almost every skill,
                though not to a particularly high level in any specific skill.
                Anyone may join the adventurer&apos;s guild, but it is not necessary in
                order to train there.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Crimson Band</h4>
              <p>
                Situated in a citadel in southwestern Fayd, this organization
                maintains high standards in the arts of combat. The warriors of
                the Crimson Band are feared and respected throughout Fayd.
                Long-time protectors of the Royal family, The Band helps to maintain
                order in and around Fayd, including the apprehension of warranted
                criminals. Membership is by invitation only.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Eliditur</h4>
              <p>
                The ELIDITUR is an organization of seekers after knowledge
                who study the arts of magic and psionic power. They follow many
                paths and their goals are not known to the world at large. Some
                consider them benign and others malign, but the truth is known
                only to the organization&apos;s members. Membership is by invitation only.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Fellowship of Rakes</h4>
              <p>
                No one except its own members can know just who belongs to
                this secret organization. While some claim that it does not exist,
                others are convinced that it is the largest guild in Fayd. Rakes train
                in a swashbuckling style of combat and stealth. The Fellowship has
                often acted as a vigilante organization to the annoyance of the
                Lawkeepers and the enmity of the Thieves&apos; Guild. Membership is
                by invitation only.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Foresters</h4>
              <p className="italic text-gray-400">
                To preserve, protect and nurture Nature&apos;s kingdom. That is
                our creed. Shirk not from our task.
              </p>
              <p>
                The Forester&apos;s guild is an organization dedicated to
                protecting the wildlands. They are pathfinders and guides who
                patrol the surrounding lands of Fayd from those who would
                cause harm. Their members range from crafts people to warriors
                and spell casters and there is no one set of skills which they
                seek from potential members. They ask only that their hearts
                are true and that they are dedicated to their cause. Their
                headquarters is located in the sacred grove of the Silver Oak,
                northeast of Fayd.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Lawkeepers</h4>
              <p>
                This is Fayd&apos;s law-keeping organization. Lawkeepers are authorized
                to <span className="text-amber-300">ARREST</span> wanted criminals. High-ranking Lawkeepers may issue
                warrants for arrest. This organization offers exceptional education
                in combat related skills. Membership is by invitation only.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Order of Silver Arcana</h4>
              <p>
                This organization is stationed in the Tower of Silver Arcana,
                the tallest and one of the most ancient structures in Fayd. The
                Order teaches skill in magic and psionics in an effort to thwart
                the forces of evil and chaos in the world. Membership is by
                invitation only.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Order of the Way</h4>
              <p>
                The Order of the Way is located in a castle to the south of Fayd. Its
                membership is predominantly Drakin, but the guild is open to all races.
                The Way guild is neutral in orientation. They strive to maintain harmony
                in all things, but have been known to act as the allies to light or to
                darkness, depending on where the balance lies. Most members of the
                Order worship their Ancestors and the guild scholars have painstakingly
                researched an ancient Drakin language and have documented clan lineage.
                Membership is by invitation only.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Physicians Guild</h4>
              <p>
                This guild offers training in the healing arts. Anyone may join this guild.
                The Physician&apos;s Guild will aid any in need (except undead), but will charge
                a modest fee for salves, splints and bandages. Mental afflictions cannot
                be cured here.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Technologists</h4>
              <p>
                The Technologists Guild believes that the future of Fayd lies in the
                return of technology to Andor. They view magic as a backward and an
                evil force which will keep Fayd in the Dark Ages. Through archaeology
                and research they try to uncover technological items and spread the
                influence of science as an alternative to the superstitious practices
                of mages. Their accomplishments include the archaeological dig that
                resulted in the discovery of Fayd&apos;s subway system as well as the
                rebuilding of Cybertech.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Thieves&apos; Guild</h4>
              <p>
                This &ldquo;guild&rdquo; is the center of racketeering, extortion and theft in
                Fayd. Its headquarters are kept secret by its members and said to
                be protected by many traps.
              </p>
            </div>

            {/* Religious Organizations */}
            <h3 id="religious-organizations" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Religious Organizations
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                There are six main religions in the Shattered Realms. With
                the exception of the Cult of Dahkahn, they generally coexist in
                peace, though heated theological disputes are common. All teach
                necromancy, however each teaches a different weapon skill. Career
                minded cleric types should tour the temples before making a
                decision.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Church of Shemri, the Earth Mother</h4>
              <p>
                Shemri is goddess of mercy, fertility, and motherhood. She gives
                life to the land. Her husband is Amilor. It is said that she
                drives the natural cycle of birth, death and rebirth. Perennial
                disputes arise between the Church of Shemri and the Temple of Rorin
                pertaining to the existence of the vast deserts in the Shattered
                Realms. Followers of Shemri often claim that Rorin is killing the
                world, where the worshippers of Rorin respond that he has not
                forgiven the world yet. Her priests wear ruddy-brown robes and
                their holy symbol is a golden great oak inlaid onto a silver disk.
                The open hand symbol is also associated with this church.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Cult of Dahkahn, The Devourer</h4>
              <p>
                Dahkahn is a name spoken only in whispered tongues amongst the
                people of the Realms. It is a name to strike fear into the heart
                of anyone. Dahkahn is usually referred to by other names: the
                Devourer of Souls, Lifesbane, Lord of the Grave, King of Dread, or
                The Enemy. Those who worship Dahkahn do so in secret, and only for
                the most selfish of reasons. Dahkahn is said to be incredibly cruel
                and demanding of his servants in return for the powers he bestows
                on them. He is recognized by many as the source of all death,
                decay and disease. Membership in Dahkahn is secret and by invitation
                only.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Temple of Amilor, Lord of the Deep</h4>
              <p>
                Amilor is the god of the waters, emotions and strong
                passion. He is husband to Shemri the earth mother. He is seen as both
                nurturer, bringing warm rains to the earth, and as a destroyer of his
                enemies in a ravaging storm of fury. The church of Amilor is composed
                of the orders of the priests and the knights. Often Amilor&apos;s followers
                are driven by strong passions and can be as mercurial as their god,
                bringing both healing and peace to the land or wreaking terrible
                vengeance on the forces of evil.
              </p>

              <h4 className="text-green-400 font-bold mt-4">Temple of Rorin, the Sun, the Lord of Light</h4>
              <p>
                Rorin lights the day, heats the world, and helps the plants grow,
                yet he is master of the deserts and can burn life away. His
                worshippers seek both to appease his wrath and secure his help in
                making crops prosper. They believe in generational responsibility,
                that the state of the world is what it is because of the sins of
                their ancestors. Rorin&apos;s followers believe that when all people
                of the world acknowledge Rorin as the greatest god, &ldquo;The deserts
                shall be swept away like so much sand from a table, and life shall
                prosper forevermore in the glory of the Lord of Light.&rdquo; Priests of
                Rorin wear bright orange raiment and carry a golden eight-rayed sun
                for a symbol. When depicted in art, Rorin is shown as a sparsely
                dressed man surrounded by swirling flames.
              </p>
            </div>

            {/* Skills */}
            <h2 id="skills" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Skills
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p className="text-gray-400 italic">
                Format: SKILLNAME (Cost of first skill level / Development skill level cost)
              </p>

              <SkillEntry name="Alchemy" cost="15/5">
                This skill enables the character to create potions. Potions are formed
                by mixing alchemical reagents. These reagents may be actual chemical
                compounds, or they may be certain magical herbs or other organic byproducts.
                It is up to the alchemist to learn what substances have magical
                properties and what substances can be combined to form useful potions.
                Each level of skill in Alchemy allows the creation of increasingly
                powerful potions. Creating a potion requires that three reagents be combined,
                plus some sort of liquid container (such as a flask). Each type of reagent
                contains one specific type of &ldquo;essence.&rdquo; You ALWAYS will need at least one
                reagent that is a catalyst. Mandrake root is a well-known but weak catalyst.
                Prerequisite: must be level two or higher.
              </SkillEntry>

              <SkillEntry name="Backstab" cost="12/5">
                The art of the deadly sneak attack, backstab may only be performed
                when hidden and only with a puncture weapon such as a dagger or
                rapier. A successful attack always causes damage to the back. It
                increases this damage by level of backstab.
              </SkillEntry>

              <SkillEntry name="Claws" cost="10/3">
                The natural claws of certain races or artificial cyberclaws can
                become vicious weapons. Races who have naturally occurring claws
                may use them without training. Drakin, Murgs and Wolflings in wolf
                form have claws they can use. Wolves and Drakin can also use their
                teeth with this skill.
              </SkillEntry>

              <SkillEntry name="Climbing" cost="6/3">
                The more well developed the climbing skill, the more treacherous
                the terrain a character can overcome without tools or footholds.
              </SkillEntry>

              <SkillEntry name="Combat Maneuvering" cost="10/5">
                This skill enables a warrior to maneuver around the battle field more
                effectively. Benefits include: each level decreases wielding roundtime
                one second; learning to guard inanimate objects; +5 bonus per level to
                guarding rolls; increased carry weight before fatigue penalty in battle;
                a chance to escape confinement (2% per level); and a chance to avoid
                monster special attacks (2% per level, max 95%).
              </SkillEntry>

              <SkillEntry name="Conjuration" cost="10/5">
                This school of magic teaches a character to cast spells that summon
                primal elemental forces, objects and creatures. Increased levels allow
                more powerful conjuration spells. Prerequisite: Spellcraft.
              </SkillEntry>

              <SkillEntry name="Crushing Weapons" cost="12/5">
                Skill in the specialized use of crushing weapons such as
                the club, warhammer, mace or morning star. Each level of training
                adds +5 to your attack modifier. Cannot be used with two-weapon skill.
              </SkillEntry>

              <SkillEntry name="Disguise" cost="15/10">
                Each rank provides new ways you might disguise yourself.
                Typing the DISGUISE command will provide instructions. Prerequisite: Stealth.
              </SkillEntry>

              <SkillEntry name="Dodging and Parrying" cost="8/4">
                Each level adds +5 to defense modifier. Prerequisite: at least one level of
                Edged, Drakin, Crushing, Pole Arm, Missile Weapon, Claws or Martial Arts.
              </SkillEntry>

              <SkillEntry name="Drakin Weapons" cost="12/5">
                Specialization in weapons developed by the Drakin culture: bo-stick,
                nunchaku, naginata, katana, sai, wakizashi, kusari-gama, and shuriken.
                Each level adds +5 to attack modifier. Can be used with two-weapon skill.
              </SkillEntry>

              <SkillEntry name="Druidic" cost="10/5">
                This school of magic teaches spells dealing with nature and the weather.
                Prerequisite: Spellcraft.
              </SkillEntry>

              <SkillEntry name="Dyeing/Weaving" cost="8/4">
                Create thread and fabric from raw materials such as wool,
                then create clothing and other woven items. Can also color fabrics
                using naturally occurring dyes. Prerequisite: must be level two or higher.
              </SkillEntry>

              <SkillEntry name="Edged Weapons" cost="12/5">
                Skill in swords, daggers and axes. Each level adds +5 to attack modifier.
                Can be used with two-weapon skill.
              </SkillEntry>

              <SkillEntry name="Endurance" cost="8/4">
                Rigorous physical training. Gain 4 body points per level. Each level
                also provides 1% reduction in elemental damage (fire, lightning, cold),
                to a maximum of 50%.
              </SkillEntry>

              <SkillEntry name="Enchantment" cost="10/5">
                Spells that alter the nature of physical objects and the minds of creatures.
                Prerequisite: Spellcraft.
              </SkillEntry>

              <SkillEntry name="Healing" cost="20/2">
                Administer first aid and heal wounds. It is easier to heal simple injuries
                than complex ones, and easier to heal someone of the same race.
              </SkillEntry>

              <SkillEntry name="Jeweler" cost="10/5">
                Craft baubles and bangles. May also ENCRUST, INLAY and INSET items with gems.
                Freshly made jewelry is automatically imbued with a small amount of magical power.
                Prerequisite: must be level two or higher.
              </SkillEntry>

              <SkillEntry name="Legerdemain" cost="12/4">
                Sleight of hand, picking pockets and shoplifting. Higher levels improve
                ability to determine item worth.
              </SkillEntry>

              <SkillEntry name="Lockpicking" cost="10/4">
                The skill of opening locks without the proper key.
              </SkillEntry>

              <SkillEntry name="Martial Arts" cost="12/5">
                Turns bare hands and feet into deadly weapons. Each level adds +5
                to attack modifier when unarmed. Beyond 10 ranks, can strike creatures
                requiring magical weapons. Each rank also adds +2 defense while unarmed.
              </SkillEntry>

              <SkillEntry name="Mind Over Matter" cost="10/5">
                Provides access to psionic disciplines pertaining to energy and matter.
              </SkillEntry>

              <SkillEntry name="Mind Over Mind" cost="10/5">
                Provides access to psionic disciplines pertaining to the mind.
              </SkillEntry>

              <SkillEntry name="Mining" cost="8/4">
                Survey areas for useful minerals and mine ores. Higher levels provide
                a better chance of finding valuable ores. Prerequisite: must be level two or higher.
              </SkillEntry>

              <SkillEntry name="Missile Weapons" cost="12/5">
                Longbow, blow-gun, sling, gun and crossbow. Cannot be used with two-weapon
                skill or shields.
              </SkillEntry>

              <SkillEntry name="Necromancy" cost="10/5">
                Spells that hold sway over the forces of life and death. Casting can
                affect alignment. Prerequisite: Spellcraft.
              </SkillEntry>

              <SkillEntry name="Polearms" cost="12/5">
                Halberds, pikes and lances. Cannot be used with two-weapon skill.
                Considered 2-handed weapons; may not be used with shields.
              </SkillEntry>

              <SkillEntry name="Psionics" cost="20/5">
                Opens the character&apos;s mind to perform supernatural feats based on psychic will.
              </SkillEntry>

              <SkillEntry name="Sagecraft" cost="5/3">
                Knowledge of local history, mythology, folklore, nature and science.
                Use the RECALL command for additional information. Also helps skin more
                specimens from dead creatures.
              </SkillEntry>

              <SkillEntry name="Spellcraft" cost="20/5">
                Basic principles of magic. Allows use of magical items such as wands.
                Each level increases casting success chance by 5%, to a maximum of 95%.
              </SkillEntry>

              <SkillEntry name="Stealth" cost="10/4">
                Hiding in shadows and staying hidden while moving silently. Success depends
                on lighting, terrain and the perception of others.
              </SkillEntry>

              <SkillEntry name="Thrown Weapons" cost="12/5">
                Rocks, spears and javelins. Cannot be used with two-weapon skill.
              </SkillEntry>

              <SkillEntry name="Transcendence" cost="10/2">
                Enables the planar voyager to retain skills when traveling to other planes.
                Without it, you are limited to your transcendence-level of skill in all
                other skills when on another plane.
              </SkillEntry>

              <SkillEntry name="Trap and Poison Lore" cost="6/3">
                Finding, setting, and disarming traps. Also allows anointing weapons
                with poison &mdash; each level allows a higher level poison to be applied.
              </SkillEntry>

              <SkillEntry name="Two Weapon" cost="10/4">
                Fight with a weapon in each hand. The secondary weapon&apos;s attack bonus equals
                levels of two weapon skill, limited by skill in that weapon type, multiplied by +5.
              </SkillEntry>

              <SkillEntry name="Weaponsmithing" cost="10/5">
                Repair damaged weapons and craft new ones. Can also smelt ore into usable metal.
                Advanced weaponsmiths can work with superior substances to create magical weapons.
                Prerequisite: must be level two or higher.
              </SkillEntry>

              <SkillEntry name="Woodlore" cost="8/3">
                Forage for useful wilderness products, craft missile weapons and arrows.
                Works with sagecraft for skinning. Prerequisite: must be level two or higher.
              </SkillEntry>
            </div>

            {/* Player Interaction */}
            <h2 id="player-interaction" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Player Interaction
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                You communicate with Legends of Future Past through an English
                parser. If you wish to do something try typing a sentence describing
                what you want to do. If you see a closed door and want to open it,
                type <span className="text-amber-300">OPEN DOOR</span>. If you see a monster and you want to kill it, type
                {' '}<span className="text-amber-300">KILL MONSTER</span>. Depending on the command, only one or two letters may be
                needed: OPEN DOOR could be typed as OP DO in most instances.
              </p>
              <p>
                In the case where there is more than one item in an area, you can use
                words to distinguish them. Say you were in a room with 2 doors, one
                a large door and the other a teak door&hellip;
              </p>
              <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-3 my-3">{`Go door        - takes you through the first door
Go 2 door      - takes you through the second door
Go second door - also works
Go teak door   - also works`}</pre>
              <p>
                To speak to others, start your text with an apostrophe (<span className="text-amber-300">&apos;</span>)
                and the text that follows will be displayed to everyone nearby.
              </p>
              <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-3 my-3">{`You type:       'Hello there.
Everyone sees:  Balthazar says, "Hello there"

You type:       'Hey, wait a sec!
Everyone sees:  Balthazar exclaims, "Hey, wait a sec!"

You type:       'Where is everybody going?
Everyone sees:  Balthazar asks, "Where is everybody going?"`}</pre>
              <p>
                To repeat the last command you entered, type a period (<span className="text-amber-300">.</span>) by itself.
              </p>
              <p>
                Other important commands: <span className="text-amber-300">QUIT</span> or <span className="text-amber-300">EXIT</span> to leave the game,
                {' '}<span className="text-amber-300">WHO</span> to see online players, <span className="text-amber-300">STATUS</span> for your character description,
                {' '}<span className="text-amber-300">VERSION</span> for version notes, <span className="text-amber-300">ASSIST</span> to summon a GameMaster,
                {' '}<span className="text-amber-300">REPORT</span> to report bugs, <span className="text-amber-300">HELP</span> for a list of verbs, and
                {' '}<span className="text-amber-300">ADVICE</span> for new player help.
              </p>
            </div>

            {/* Settings */}
            <h3 id="settings" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Settings
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                Type <span className="text-amber-300">SET</span> by itself to display all current and possible settings.
                Toggle with <span className="text-amber-300">SET &lt;setting name&gt; [on/off]</span>.
              </p>
              <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-3 my-3">{`Full          Full room descriptions
Brief         Brief room descriptions
Echo          Toggle local echo
Prompt        See prompt codes
Logon         See logon messages
Logoff        See logoff messages
Disconnect    See disconnect messages
RPbrief       Hide RP verbs to other people
Battlebrief   Hide RP actions by monsters, reduce combat scroll
Actionbrief   Filter healing, spell casting, eating by others
Actbrief      Show/hide parentheses around ACT commands`}</pre>
            </div>

            {/* Movement */}
            <h3 id="movement" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Movement
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                As you move around, you will be given a description of the area
                and a list of obvious exits.
              </p>
              <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-3 my-3">{`[Dark Tower]
You are standing at the foot of a menacing tower. Thick forest
surrounds you on all sides. You see a rickety iron door.
Obvious exits: north, south, east.
>`}</pre>
              <p>
                Type compass directions to move (e.g., <span className="text-amber-300">north</span> or <span className="text-amber-300">n</span>).
                Read room descriptions for portals: type <span className="text-amber-300">GO DOOR</span> to move through doors.
              </p>
              <p>
                Certain races can <span className="text-amber-300">FLY</span>, and there are magical and technological methods.
                While flying you can <span className="text-amber-300">ASCEND</span> and <span className="text-amber-300">DESCEND</span> through different altitude maps,
                using compass directions for heading.
              </p>
            </div>

            {/* Command Glossary */}
            <h2 id="command-glossary" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Command Glossary
            </h2>
            <div className="text-gray-300 space-y-1 leading-relaxed">
              <CommandEntry cmd="ACT <sentence>" desc="Replaces 'act' with your character's name in brackets." />
              <CommandEntry cmd="ADVANCE <target>" desc="Move towards a creature to engage at melee range." />
              <CommandEntry cmd="ADVICE" desc="Special help for level 1-2 characters." />
              <CommandEntry cmd="ANALYZE <item>" desc="Examine an item for alchemical properties. Requires Alchemy skill." />
              <CommandEntry cmd="ANOINT <weapon> WITH <poison>" desc="Coat a weapon with poison. Requires Trap & Poison skill." />
              <CommandEntry cmd="ARREST <target>" desc="Lawkeepers only. Used on someone with a pending warrant." />
              <CommandEntry cmd="ASCEND" desc="Move vertically upwards while flying." />
              <CommandEntry cmd="ASSIST" desc="Notifies GMs that you require assistance." />
              <CommandEntry cmd="ATTACK <target>" desc="Attack with intent to kill. Synonyms: A, KILL, SLAY, SMITE." />
              <CommandEntry cmd="AVOID <target>" desc="Actively attempt to avoid a creature or person." />
              <CommandEntry cmd="BACKSTAB <target>" desc="Deadly sneak attack. Must be hidden first." />
              <CommandEntry cmd="BALANCE" desc="View bank account balance and stored items." />
              <CommandEntry cmd="BERSERK" desc="Maximum offense, greatly lowered defense." />
              <CommandEntry cmd="BITE <target>" desc="Jaw attack for races with effective bite." />
              <CommandEntry cmd="BLEND" desc="Highlander cave/mountain camouflage." />
              <CommandEntry cmd="BREAK <obj> WITH <obj>" desc="Destroy an item with a stronger item." />
              <CommandEntry cmd="BREW <reagent> IN <container>" desc="Create potion. Requires Alchemy skill." />
              <CommandEntry cmd="BRIEF" desc="Shorten room descriptions. FULL reverses this." />
              <CommandEntry cmd="CALL" desc="Aelfen: call a woodland creature as familiar." />
              <CommandEntry cmd="CANT <message>" desc="Covert message. Requires Legerdemain 6+." />
              <CommandEntry cmd="CAST <target>" desc="Cast an invoked spell at the target." />
              <CommandEntry cmd="CHANT <scroll>" desc="Cast the spell on a scroll." />
              <CommandEntry cmd="CLIMB <object>" desc="Negotiate an obstacle by climbing." />
              <CommandEntry cmd="CLOSE <object>" desc="Shut a door, close a container." />
              <CommandEntry cmd="COMMAND <instructions>" desc="Instruct a conjured/dominated creature: BEGONE, direction, FOLLOW, GUARD, KILL, LOOK, WATCH, ECHO." />
              <CommandEntry cmd="CRAFT <item name>" desc="List craftable items or begin constructing one." />
              <CommandEntry cmd="DEFENSIVE" desc="More defensive combat strategy." />
              <CommandEntry cmd="DEPOSIT <amount|item>" desc="Store cash/valuables in the bank (amounts in copper)." />
              <CommandEntry cmd="DESCEND" desc="Move vertically downwards while flying." />
              <CommandEntry cmd="DISARM <object>" desc="Locate and disarm traps." />
              <CommandEntry cmd="DISBAND" desc="Release all followers." />
              <CommandEntry cmd="DISGUISE" desc="Disguise yourself (requires Disguise skill)." />
              <CommandEntry cmd="DRINK <object>" desc="Consume a liquid. Synonyms: SIP, QUAFF." />
              <CommandEntry cmd="DROP <object>" desc="Put an object on the ground." />
              <CommandEntry cmd="DYE <item> WITH <item>" desc="Color material with a pigment. Requires Weaving/Dyeing." />
              <CommandEntry cmd="EAT <object>" desc="Consume a food." />
              <CommandEntry cmd="EMOTE" desc="Mechanoids: engage emotional half of brain." />
              <CommandEntry cmd="ENCRUST <item> WITH <gem>" desc="Adorn an item with a gem. Requires Jeweler." />
              <CommandEntry cmd="ENROLL" desc="Join an open organization." />
              <CommandEntry cmd="EXTINGUISH <object>" desc="Put out a torch, lantern, etc." />
              <CommandEntry cmd="FILL <container> WITH <liquid>" desc="Fill a liquid container." />
              <CommandEntry cmd="FLY" desc="Take off into the air." />
              <CommandEntry cmd="FORAGE" desc="Search wilderness for useful substances. Requires Wood Lore." />
              <CommandEntry cmd="FRENZY" desc="Murg battle frenzy: more damage, easier to hit." />
              <CommandEntry cmd="FULL" desc="Full room descriptions. Reversed by BRIEF." />
              <CommandEntry cmd="GET <object>" desc="Pick up an object or unconscious/dead person. Synonym: TAKE." />
              <CommandEntry cmd="GO <object>" desc="Move through a portal, down a trail, etc." />
              <CommandEntry cmd="GUARD <target>" desc="Guard someone from melee engagement." />
              <CommandEntry cmd="HEALTH" desc="Show health status. Synonym: DIAGNOSE." />
              <CommandEntry cmd="HELP" desc="Access the help system." />
              <CommandEntry cmd="HIDE" desc="Attempt to conceal yourself." />
              <CommandEntry cmd="INITIATE <player>" desc="Initiate someone into your organization (target must kneel)." />
              <CommandEntry cmd="INLAY/INSET <item> WITH <gem>" desc="Adorn items with gems. Requires Jeweler." />
              <CommandEntry cmd="INVENTORY" desc="View what you are carrying/wearing. Synonym: I." />
              <CommandEntry cmd="INVOKE <spell code>" desc="Prepare a spell. Synonyms: CHANT, PREPARE." />
              <CommandEntry cmd="JOIN <target>" desc="Join someone's party. Synonym: FOLLOW." />
              <CommandEntry cmd="LEAVE" desc="Leave your current group." />
              <CommandEntry cmd="LIGHT <object>" desc="Light a lamp or candle. Synonym: IGNITE." />
              <CommandEntry cmd="LOCK <obj> WITH <obj>" desc="Lock an object with the proper key." />
              <CommandEntry cmd="LOOK [in|on|under] <object>" desc="Examine surroundings or objects. Synonyms: L, EXAMINE, INSPECT." />
              <CommandEntry cmd="MARK <#>" desc="Mark your location (1-10) for use with spells." />
              <CommandEntry cmd="MASTER <spell>" desc="Specialize in a spell for increased damage/duration." />
              <CommandEntry cmd="MINE WITH <tool>" desc="Mine ore. Requires Mining skill and a pick-axe." />
              <CommandEntry cmd="MOLD <gem>" desc="Highlander: improve a gem." />
              <CommandEntry cmd="NOCK <bow> WITH <arrow>" desc="Ready a missile weapon. Synonym: LOAD." />
              <CommandEntry cmd="NORMAL" desc="Equal balance between offense and defense." />
              <CommandEntry cmd="OFFENSIVE" desc="More offensive, between berserk and normal." />
              <CommandEntry cmd="OPEN <object>" desc="Open a door, container, etc." />
              <CommandEntry cmd="ORDER <item>" desc="Buy an item at a shop. Synonym: BUY." />
              <CommandEntry cmd="PICK <obj> WITH <obj>" desc="Pick a lock with a lockpick." />
              <CommandEntry cmd="POISON <weapon> WITH <potion>" desc="Anoint weapon with poison. Synonym: ANOINT." />
              <CommandEntry cmd="POUR <container> IN <container>" desc="Transfer or combine liquids." />
              <CommandEntry cmd="PROMPT" desc="Show status codes before the prompt." />
              <CommandEntry cmd="PUT <obj> IN|ON|UNDER <obj>" desc="Place an object. Synonym: PLACE." />
              <CommandEntry cmd="QUIT" desc="Leave the game. Synonyms: EXIT, Q, X." />
              <CommandEntry cmd="READ <item>" desc="Read text written on an item." />
              <CommandEntry cmd="RECALL" desc="Use sagecraft for additional area/object knowledge." />
              <CommandEntry cmd="REMOVE <object>" desc="Take off something you are wearing." />
              <CommandEntry cmd="REPAIR <item>" desc="Fix damaged items. Weaponsmithing needed for weapons." />
              <CommandEntry cmd="RETREAT" desc="Back off from melee engagement." />
              <CommandEntry cmd="SEARCH <corpse>" desc="Remove items from a corpse." />
              <CommandEntry cmd="SELFTRAIN <skill#>" desc="Train yourself at +1 build point cost." />
              <CommandEntry cmd="SKILL" desc="Display all current skills." />
              <CommandEntry cmd="SKIN <corpse>" desc="Eviscerate for specimens. Requires Woodlore or Sagecraft." />
              <CommandEntry cmd="SMELT <ore>" desc="Extract metal from raw ore. Requires Weaponsmith." />
              <CommandEntry cmd="SNEAK <direction>" desc="Move while staying hidden." />
              <CommandEntry cmd="SPECIALIZE <weapon>" desc="Increase chance to hit vital areas with a specific weapon." />
              <CommandEntry cmd="SPELL" desc="View all spells you know." />
              <CommandEntry cmd="SPLIT <amount> <coin type>" desc="Divide coins between party members." />
              <CommandEntry cmd="STALK <target>" desc="Secretly track someone." />
              <CommandEntry cmd="STATUS" desc="View your statistics." />
              <CommandEntry cmd="STEAL <target>" desc="Pick pockets. Synonyms: FILCH, ROB." />
              <CommandEntry cmd="SUBMIT" desc="Submit to arrest or lower spell resistance to zero." />
              <CommandEntry cmd="SURVEY" desc="Survey area for minerals. Requires Mining." />
              <CommandEntry cmd="TEACH <skill|spell#>" desc="Teach a skill to others in the room." />
              <CommandEntry cmd="TELEPATHY" desc="Toggle telepathy on/off." />
              <CommandEntry cmd="TEND <target>" desc="Use Healing skill to remove injuries." />
              <CommandEntry cmd="THINK <message>" desc="Transmit a telepathic message." />
              <CommandEntry cmd="TRAIN <skill#>" desc="Exchange build points to learn/improve a skill." />
              <CommandEntry cmd="TRANSFORM" desc="Wolfling: toggle between wolf and humanoid form." />
              <CommandEntry cmd="TRAP <obj> WITH <obj>" desc="Place a trap on a container." />
              <CommandEntry cmd="UNDRESS" desc="Remove outermost layer of clothing." />
              <CommandEntry cmd="UNEMOTE" desc="Mechanoids: switch to analytical mode." />
              <CommandEntry cmd="UNLEARN <skill#>" desc="Unlearn one rank, once per level. Refund cost minus one." />
              <CommandEntry cmd="UNLOCK <obj> WITH <obj>" desc="Unlock something with the appropriate key." />
              <CommandEntry cmd="UNPROMPT" desc="Remove the status prompt." />
              <CommandEntry cmd="UNWIELD <object>" desc="Stop wielding a weapon. Synonym: UNUSE." />
              <CommandEntry cmd="VERSION" desc="Display version and notes. Synonyms: NEWS, NOTES." />
              <CommandEntry cmd="WARY" desc="Slightly more defensive than normal." />
              <CommandEntry cmd="WEAR <object>" desc="Put on wearable equipment. Synonym: DON." />
              <CommandEntry cmd="WHISPER [TO] <target> <msg>" desc="Say something privately to someone nearby." />
              <CommandEntry cmd="WHO" desc="List everyone currently playing." />
              <CommandEntry cmd="WIELD <object>" desc="Ready a weapon. Synonym: USE." />
              <CommandEntry cmd="WITHDRAW <amount|item>" desc="Remove money/items from bank." />
              <CommandEntry cmd="WORK <material>" desc="Work material into a crafted item. Requires tools and skill." />
              <CommandEntry cmd="YELL <message>" desc="Shout so everyone nearby hears. Synonym: SHOUT." />

              <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-3 my-3">{`Prompt status codes:
  !     bleeding          s     sitting
  S     stunned           D     diseased
  P     poisoned          J     joined in a group
  K     kneeling          L     laying
  R     in roundtime      H     hidden or invisible
  U     unconscious       I     immobilized
  DEAD  dead`}</pre>
            </div>

            {/* Other Verbs */}
            <h3 id="other-verbs" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Other Verbs
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                Legends of Future Past is intended to be intuitive and there are
                over two hundred verbs that it understands. Try any verb which
                comes to your mind! Here are some examples:
              </p>
              <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-3 my-3">{`babble  beam  bow  chuckle  cry  dance  feel  fidget  frown  groan
handshake  headshake  hug  kiss  kneel  lay  mumble  poke  pull
push  roll  sit  shiver  snort  stand  swoon  toast  wink  laugh
shudder  tickle  raise (eyebrow)  howl  turn  point  kick  knock
touch  rub  carry  get  pet  slap  punch  roll  spit  wave  tap
gaze  glare  scowl  furrow (brow)  sing  recite  pace  comfort`}</pre>
            </div>

            {/* Combat */}
            <h2 id="combat" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Combat
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                Success in battle is determined by a myriad of variables.
                All your statistics, skill, weapon type, armor, weather, visibility,
                lighting, terrain, range and other values factor in.
              </p>
              <p>
                When an attack is made, the system calculates your To Hit value. You
                must roll over that number on a d100 to score a hit. The
                minimum To Hit is always 5 (always a chance to miss) and the maximum
                is 95 (always a slight chance to hit).
              </p>
              <p>
                Damage is based on the weapon you wield. High strength increases melee
                and polearm damage. High agility increases damage from bows and thrown weapons.
              </p>
              <p>
                Melee attacks cause fatigue loss. The amount depends on equipment weight
                relative to your strength and constitution. At half fatigue, a -10 To Hit
                penalty is imposed; at quarter fatigue, a -25 penalty. Missile and Thrown
                Weapons do not use fatigue.
              </p>
              <p>
                Psionics and magic do not require a To Hit roll. Instead, the target gets
                a chance to resist based on their statistics.
              </p>
              <p>
                <span className="text-green-400">Weapon Specialization:</span> After training to 10th level in a weapon category,
                you may specialize in a specific weapon. Each rank increases vital area
                hit chance by 15%, max 75% (5 ranks max). First rank costs 10 build points;
                additional ranks cost 5 each. Max 5 different weapon specializations.
              </p>
              <p>
                Stronger hits may cause wounds: lacerations, gashes, etc. Bleeding wounds
                drain body points until healed. Rolls over 95 may stun (-20 defense) or
                knock you down (-50 defense).
              </p>
              <p>
                <span className="text-green-400">Weapon Clash:</span> On attack rolls less than 3, weapons can be damaged
                in a &ldquo;weapon clash.&rdquo; The chance is based on a 2d100 roll versus weapon strength
                (material, size, power). Damaged weapons have a -10 ToHit penalty and can
                be repaired by a weaponsmith. A damaged weapon that encounters another clash
                will break permanently!
              </p>
              <p>
                To engage in melee, <span className="text-amber-300">ADVANCE</span> towards the target. To disengage, <span className="text-amber-300">RETREAT</span>.
                Missile weapons, thrown weapons, and polearms can be used at range.
              </p>
              <p>
                Battle strategies: <span className="text-amber-300">NORMAL</span> (balanced), <span className="text-amber-300">OFFENSIVE</span>, <span className="text-amber-300">DEFENSIVE</span>,
                {' '}<span className="text-amber-300">BERSERK</span> (max offense, minimal defense), <span className="text-amber-300">WARY</span> (cautious).
                Use <span className="text-amber-300">GUARD &lt;target&gt;</span> to protect a companion (ranged/spells still get through).
              </p>
            </div>

            {/* Armor */}
            <h3 id="armor" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Armor
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                Wearing armor reduces damage from attacks. Heavier armor provides
                more protection but causes higher fatigue rates. Different armors
                protect against certain attacks better than others: plate armor is
                not nearly as effective against crushing weapons as it is against
                slashing. Some races have natural armor that improves with level.
                All armor other than leather detrimentally affects spellcasting.
              </p>
            </div>

            {/* Missile and Thrown Weapons */}
            <h3 id="missile-thrown-weapons" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Missile and Thrown Weapons
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                Missile weapons (bows, firearms) launch projectiles at range
                without needing to ADVANCE to melee. You must load your weapon with
                {' '}<span className="text-amber-300">LOAD</span> or <span className="text-amber-300">NOCK</span> (e.g., <span className="text-amber-300">NOCK BOW WITH ARROW</span>).
              </p>
              <p>
                Thrown weapons must be picked up and re-wielded between throws.
                When in melee range, slashing/piercing thrown weapons count as melee
                attacks and use fatigue.
              </p>
            </div>

            {/* Backstab */}
            <h3 id="backstab" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Backstab
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                After successfully hiding, you can attempt a <span className="text-amber-300">BACKSTAB</span> on your opponent.
                Each level of backstab skill increases total damage. Backstabs can only
                be performed with puncture melee weapons, such as daggers and rapiers.
              </p>
            </div>

            {/* Death */}
            <h3 id="death" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Death
            </h3>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                If you drop below 0 body points, you are unconscious. You may drift
                in and out of consciousness and can potentially be healed or eat
                babich root when you come to. After a while, you may die or receive
                a coup de grace. Someone with Breath of Life can restore you.
              </p>
              <p>
                If not raised, you may <span className="text-amber-300">DEPART</span>. Eternity, Inc. will restore you
                (new citizens begin with five free treatments). Biological forms are
                cloned; ephemerals are reintegrated; mechanoids are repaired. Your
                possessions are returned.
              </p>
              <p>
                Death results in loss of up to 90% of experience towards your next
                build point. This is softened to 20% if resurrected with Breath of Life.
                Some dangerous areas cannot be departed from. The Deep Realms require
                different insurance payments.
              </p>
            </div>

            {/* Psionics */}
            <h2 id="psionics" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Psionics
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                <span className="text-green-400">Psionics (20/5):</span> Prerequisite for other psionic skills. Each level
                aids invocation success and overcoming psionic resistance.
              </p>
              <p>
                <span className="text-green-400">Mind Over Matter (10/5)</span> and <span className="text-green-400">Mind Over Mind (10/5)</span> provide
                access to the two psionic sciences.
              </p>
              <p>
                Disciplines are &ldquo;instantaneous&rdquo; or &ldquo;maintained.&rdquo; Maintained disciplines
                cost 1 psi per 5 levels of the discipline (rounded up, minimum 1) continuously.
              </p>
              <p>
                To invoke: <span className="text-amber-300">PSI &lt;Discipline#&gt;</span>. To project at a target: <span className="text-amber-300">PROJECT &lt;target&gt;</span>.
                You don&apos;t need to re-prepare each time, but switching disciplines incurs roundtime.
                Prepare roundtime is 5 seconds; project roundtime is 5 seconds. Exceptions:
                Matter/Mind Awakening and Call on Inner Resources have 20 second roundtime;
                psychic attack disciplines have no prepare roundtime.
              </p>
              <p>
                <span className="text-green-400">Resistance:</span> Players resist at 2% per level. Physical psionic attacks use
                the monster&apos;s normal magic resistance. Mental attacks may use a different
                resistance. Undead and creatures without nervous systems are typically immune
                to mental attacks.
              </p>

              <h4 className="text-green-400 font-bold mt-6 mb-2">Mind Over Matter Disciplines</h4>
              <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-4 my-3 leading-relaxed">{`#1  Matter Awakening          Lev 1   Learn new MoM disciplines
#2  Kinetic Thrust            Lev 1   Crushing force damage
#3  Levitate                  Lev 2   Carry 50% more, reduce fatigue
#4  Pyrokinetics              Lev 3   Heat damage
#5  Cryokinetics              Lev 4   Cold damage
#6  Capacitance               Lev 5   Drain charge from item for 5 psi
#7  Electrify                 Lev 6   Electrical damage
#8  Strengthen Steel          Lev 7   +15 weapon bonus
#9  Manipulate Lock           Lev 8   Chance to unlock
#10 Wall of Force             Lev 9   +25 defense bonus
#11 Flight                    Lev 10  Fly
#42 Speed                     Lev 12  Blurring speed movement
#12 Call on Inner Power       Lev 13  10 fatigue -> 10 psi (20s RT)
#13 Transform Steel           Lev 15  Alter weapon metal (bonus 3-17)
#14 Teleportation             Lev 18  Teleport to marked location
#15 Force Field               Lev 20  +75 defense, -75 offense
#16 Immobilize                Lev 22  Prevent movement
#17 Ethereal Projection       Lev 24  Nearly invisible/undetectable`}</pre>

              <h4 className="text-green-400 font-bold mt-6 mb-2">Mind Over Mind Disciplines</h4>
              <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-4 my-3 leading-relaxed">{`#18 Mind Awakening            Lev 1   Learn new MoMind disciplines
#19 Telepathy                 Lev 1   Communicate via telepathic ether
#20 Contact                   Lev 2   Private telepathic sendings
#21 Psychic Probe             Lev 3   Remove victim's psi resistance
#22 Psychic Blast             Lev 4   Stun 6-12 seconds
#23 Psychic Screen            Lev 5   Shield from mental attacks
#24 Psychic Penetration       Lev 6   Crumple psychic screen
#25 Clairvoyance              Lev 7   Vision of a marked location
#26 Eject                     Lev 8   Remove hostile probes
#27 Cloak Aura                Lev 9   Block Aura Sense, Truename
#28 Psychic Shield            Lev 10  Superior screen (immune to Pen.)
#29 Psychic Crush             Lev 11  Stun 6-21 seconds
#30 Psychic Invasion          Lev 12  Crumple screen or shield
#31 Confuse                   Lev 13  -40 to victim's next skill roll
#32 Focus Skill               Lev 14  +25 to your next skill roll
#33 Terror                    Lev 15  Stun 6-21s + -25 defense
#34 Psychic Barrier           Lev 16  Immune to Invasion and Pen.
#35 Psychic Insinuation       Lev 17  Crumple any psi defense
#36 Domination                Lev 18  Control weak-minded creature
#37 Disruption                Lev 19  Cancel maintained disciplines
#38 Psychic Fortress          Lev 20  Greatest psi defense
#39 Warp Mind                 Lev 22  Disable attack/cast/psionics
#41 Pain                      Lev 21  20 dmg + stun + kneel
#40 Psychic Siege             Lev 25  Destroy any psi defense`}</pre>
            </div>

            {/* Spellcasting */}
            <h2 id="spellcasting" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Spellcasting
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                There are four schools of magic: <span className="text-green-400">Enchantment</span>, <span className="text-green-400">Necromancy</span>,
                {' '}<span className="text-green-400">Druidic</span> and <span className="text-green-400">Conjuration</span>.
              </p>
              <p>
                To know a spell of a certain level, you must be trained to at least
                that level in the appropriate school. Simply training in a school does
                not teach you spells &mdash; they must be found on scrolls or taught by others.
              </p>
              <p>
                Casting costs mana equal to the spell&apos;s level. Your base invocation success
                is 25% + 10% of empathy score, +5% per level of Spellcraft (max 95%).
                A failed roll fizzles; a roll over 98% is a fumble with harmful side-effects;
                a roll of 2% or below is a spectacular success (double effect).
              </p>
              <p>
                <span className="text-green-400">Spell Mastery:</span> Specialize in specific spells. Initial mastery costs 8 build points;
                additional ranks cost 4 each. Each rank increases damage/duration proportionally
                and reduces mana cost by 2 (min half original cost). Prerequisite: spellcraft
                level must be at least twice the spell&apos;s level. Mastery bonuses do not apply
                under time-altering spells (Haste, Slow).
              </p>
            </div>

            {/* Conjuration Spells */}
            <h3 id="conjuration-spells" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Conjuration Spells
            </h3>
            <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-4 my-3 leading-relaxed">{`100  Flame Bolt                    Lev 1
101  Force Blade                   Lev 3
102  Mystic Armor                  Lev 5
103  Lightning Bolt                Lev 7
105  Globe of Protection           Lev 15
106  Summon Fire Elemental         Lev 12
107  Summon Air Elemental          Lev 12
108  Summon Water Elemental        Lev 12
109  Summon Gargoyle               Lev 16
112  Call Meteor                   Lev 20
113  Light                         Lev 1
114  Mystic Key                    Lev 2
115  Shockwave                     Lev 4
116  Thunder Call                  Lev 21
117  Call Fire                     Lev 8
118  Flaming Sphere                Lev 13
119  Ice Bolt                      Lev 3
120  Frost Ray                     Lev 6
121  Freezing Sphere               Lev 9
122  Summon Familiar               Lev 2
123  Summon Earth Elemental        Lev 12
124  Inferno Glyph                 Lev 20
125  Thunder Glyph                 Lev 10
126  Ice Glyph                     Lev 15
127  Web                           Lev 10
130  Mass Protection               Lev 23
131  Flaming Arrows                Lev 18
132  Chain Lightning               Lev 23
133  Globe of Protection II        Lev 30
134  Siryx's Terrible Tentacles    Lev 25
135  Storm Blade                   Lev 24
136  Inferno Blade                 Lev 19
137  Winter Blade                  Lev 22
138  Energy Maelstrom              Lev 31
139  Sorcerous Summons I           Lev 20
140  Sorcerous Summons II          Lev 35
141  Pyrotechnics                  Lev 17
144  Tindareth's Chaotic Summons   Lev 28`}</pre>

            {/* Enchantment Spells */}
            <h3 id="enchantment-spells" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Enchantment Spells
            </h3>
            <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-4 my-3 leading-relaxed">{`200  Fear                          Lev 1
201  Charm                         Lev 3
202  Enchantment I                 Lev 4
203  Enchantment II                Lev 15
204  Enchantment III               Lev 30
205  Command                       Lev 6
206  Domination I                  Lev 12
207  Strength I                    Lev 4
208  Strength II                   Lev 8
209  Strength III                  Lev 16
210  Haste                         Lev 5
211  Slow                          Lev 5
212  Mass Invisibility             Lev 25
213  Bend Space I                  Lev 17
214  Domination II                 Lev 24
215  Scry                          Lev 10
216  Slumber I                     Lev 2
217  Slumber II                    Lev 6
218  Slumber III                   Lev 18
219  Silence                       Lev 7
220  Dancing Blade                 Lev 1
221  Dancing Sword                 Lev 6
222  Bend Space II                 Lev 23
224  Fly                           Lev 11
225  Invisibility                  Lev 14
226  Paranoia                      Lev 3
227  Imprisonment Rune             Lev 13
228  Identify                      Lev 7
229  Wizard's Armor                Lev 9
230  Disjunction                   Lev 21
231  Imprison                      Lev 19
232  Mist Form                     Lev 20
234  Spell Shield                  Lev 13
235  Cloak Mind                    Lev 22
243  Charge Wand                   Lev 26
244  Enchant an Item               Lev 31
245  Slime Form                    Lev 13
246  Yshtarin's Confounding Trans. Lev 29
248  Phantom Form                  Lev 34`}</pre>

            {/* Necromancy */}
            <h3 id="necromancy" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Necromancy
            </h3>
            <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-4 my-3 leading-relaxed">{`301  Turn Undead I                 Lev 2
302  Turn Undead II                Lev 8
303  Cure Poison                   Lev 11
304  Turn Undead III               Lev 16
305  Breath of Life                Lev 14
306  Animate Skeleton              Lev 6
307  Animate Zombie                Lev 10
308  Control Undead I              Lev 7
309  Control Undead II             Lev 13
310  Control Undead III            Lev 21
311  Speak with Dead               Lev 3
312  Wail of the Banshee           Lev 20
313  Body Destruction I            Lev 1
314  Body Destruction II           Lev 5
315  Body Destruction III          Lev 10
316  Body Restoration I            Lev 1
317  Body Restoration II           Lev 5
318  Body Restoration III          Lev 10
319  Cure Disease                  Lev 12
320  Contagion                     Lev 23
321  Poison                        Lev 17
322  Symbol of Death               Lev 25
323  Spectral Fist                 Lev 3
326  Spectral Shield               Lev 9
334  Invigoration I                Lev 2
335  Invigoration II               Lev 9
336  Wight Animation               Lev 17
337  Reconstruction                Lev 4
338  Unstun                        Lev 9
339  Destroy Undead I              Lev 3
340  Destroy Undead II             Lev 8
341  Destroy Undead III            Lev 13
343  Regeneration                  Lev 27
345  Spectral Sword                Lev 7
347  Divine Blessing               Lev 10
351  Wither Limb                   Lev 24
352  Raise Undead                  Lev 23
353  Summon Spectral Warrior       Lev 32
354  Rorin's Fire                  Lev 17`}</pre>

            {/* General Magic */}
            <h3 id="general-magic" className="text-amber-300 text-base font-bold mt-8 mb-3">
              General Magic
            </h3>
            <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-4 my-3 leading-relaxed">{`400  Detect Magic                  Lev 1
401  Dispel Lesser Magic           Lev 5
403  Mindlink                      Lev 9
404  Aura Sense                    Lev 14
405  See Hidden                    Lev 3
406  Dispel Invisibility           Lev 8
407  Analyze Ore                   Lev 3
408  Truename                      Lev 18
412  Bloodsight                    Lev 9`}</pre>

            {/* Druidic Spells */}
            <h3 id="druidic-spells" className="text-amber-300 text-base font-bold mt-8 mb-3">
              Druidic Spells
            </h3>
            <pre className="text-green-400 text-xs bg-[#111] border border-[#333] rounded p-4 my-3 leading-relaxed">{`500  Plant Snare                   Lev 4
501  Call Storm                    Lev 23
502  Disperse Storm                Lev 19
503  Call Lightning                Lev 17
504  Call Animal                   Lev 1
505  Freedom                       Lev 9
506  Resist Weather                Lev 3
507  Heat Shield                   Lev 7
508  Cold Shield                   Lev 6
509  Repel Plants                  Lev 10
510  Repel Plants and Webs         Lev 18
511  Carapace                      Lev 8
512  True Aim                      Lev 15
513  Agility I                     Lev 4
514  Agility II                    Lev 11
515  Agility III                   Lev 16
516  Wall of Thorns                Lev 14
517  Stick to Snake                Lev 5
518  Claw Growth                   Lev 2
519  Sunray                        Lev 13
520  Night Vision                  Lev 1
521  Camouflage                    Lev 7
522  Insect Swarm                  Lev 25
523  Earth Spike                   Lev 5
524  Earth Wave                    Lev 12
528  Free Action                   Lev 20
531  Tree Door                     Lev 10
532  Ride the Lightning            Lev 34
533  Commune with Nature           Lev 27
534  Claws of the Elder Wolf       Lev 21
535  Form Lock                     Lev 18
536  Wolf Form                     Lev 26`}</pre>

            {/* Art of Roleplaying */}
            <h2 id="art-of-roleplaying" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              The Art of Roleplaying
            </h2>
            <div className="text-gray-300 space-y-3 leading-relaxed text-xs italic">
              <p className="not-italic text-gray-400">by Elriss</p>
            </div>
            <div className="text-gray-300 space-y-3 leading-relaxed">
              <p>
                Legends of Future Past is, first and foremost, a roleplaying game. While
                many people play this game for a variety of reasons, the essence of the game is
                grounded in roleplaying. Roleplaying is like acting. Your ability to roleplay
                will develop in time, and is very important to the &ldquo;suspension of disbelief&rdquo;
                within the game. The better a group of roleplayers, the more believable and
                enjoyable the game will be.
              </p>
              <p>
                The first part of roleplaying a character is to make the character as
                detailed as possible in history, beliefs, motivations, and personal behavior.
                You have to decide who and what your character is. You may want to base your
                character upon a fictional literary character, with some alterations for
                originality. Whatever you decide, be consistent. It is very disconcerting for
                other players if your behavior changes from day to day.
              </p>
              <p>
                When you are in the game, it is important to act consistently within the
                beliefs of your CHARACTER, not necessarily yourself. This is called staying
                IC (In Character). While OOC (Out of Character) conversation goes on amongst
                friends, there is an appropriate time and place for it. In general, when in a
                group you don&apos;t know well, it enhances the atmosphere if you avoid open OOC talk.
              </p>
              <p>
                <span className="text-green-400">What constitutes OOC behavior?</span> Try to put yourself into the world of Andor.
                Would your character have this knowledge? Would they really say these things
                if this world were real? Avoid using game mechanics to cheat or give your
                character knowledge they would not have.
              </p>
              <p>
                Communicate to the game masters via feedback some information about your
                character, as well as what you would like to see happen. Provide a backstory,
                motivations, and goals. The GMs want to entertain you and tell good stories &mdash;
                if you come up with one, they will be more than willing to help.
              </p>
            </div>

            {/* Game Policies */}
            <h2 id="game-policies" className="text-amber-400 text-lg font-bold mt-10 mb-4 border-b border-[#333] pb-2">
              Game Policies and Your Rights as a Player
            </h2>
            <div className="text-gray-300 space-y-4 leading-relaxed">
              <div>
                <h4 className="text-green-400 font-bold">1. Disconnecting During PvP</h4>
                <p>
                  If you find yourself in player vs. player combat, DO NOT disconnect to avoid
                  death. Doing so will result in a warning; repeated offenses may lead to account
                  suspension.
                </p>
              </div>
              <div>
                <h4 className="text-green-400 font-bold">2. Cheating</h4>
                <p>
                  Taking advantage of bugs that directly help your character will result in
                  suspension. Report bugs via ASSIST or REPORT &mdash; you may receive experience
                  points as a reward. Lying to a Game Master is also considered cheating.
                </p>
              </div>
              <div>
                <h4 className="text-green-400 font-bold">3. Common Courtesy</h4>
                <p>
                  Treat everyone with respect. Continuous harassment will result in penalties.
                  We are all here to have fun.
                </p>
              </div>
              <div>
                <h4 className="text-green-400 font-bold">4. New Player Protection</h4>
                <p>
                  Players under tenth level operate under protected status. They are not to be
                  killed, searched, stolen from or otherwise harassed.
                </p>
              </div>
              <div>
                <h4 className="text-green-400 font-bold">5. Sexual Harassment</h4>
                <p>
                  Sexual harassment is seriously frowned upon and will be severely dealt with
                  by the staff. If someone is harassing you, please ASSIST and talk to a
                  Game Master.
                </p>
              </div>
            </div>

            {/* Footer */}
            <div className="mt-16 mb-8 text-center border-t border-[#333] pt-8">
              <p className="text-gray-600 text-xs">
                &copy;1994 Inner Circle Technologies, All Rights Reserved
              </p>
              <button
                onClick={onBack}
                className="mt-4 px-4 py-2 bg-[#333] hover:bg-[#444] text-gray-300 font-mono text-sm rounded border border-[#555]"
              >
                &times; Close Manual
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function SkillEntry({ name, cost, children }: { name: string; cost: string; children: React.ReactNode }) {
  return (
    <div className="mb-2">
      <p>
        <span className="text-green-400 font-bold">{name}</span>
        <span className="text-gray-500"> ({cost})</span>
        : {children}
      </p>
    </div>
  )
}

function CommandEntry({ cmd, desc }: { cmd: string; desc: string }) {
  return (
    <div className="py-1.5 border-b border-[#1a1a1a]">
      <span className="text-amber-300">{cmd}</span>
      <span className="text-gray-500"> &mdash; </span>
      <span className="text-gray-400">{desc}</span>
    </div>
  )
}
