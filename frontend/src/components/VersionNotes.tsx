export default function VersionNotes({ onBack }: { onBack: () => void }) {
  return (
    <div className="flex items-start justify-center h-full p-8 overflow-y-auto">
      <div className="max-w-3xl w-full font-mono">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-amber-500 text-2xl font-bold">Version Notes</h1>
          <button onClick={onBack} className="text-gray-400 hover:text-white text-sm">&larr; Back</button>
        </div>

        <div className="space-y-6 text-sm">
          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.17.0 &mdash; July 22, 2026</h2>
            <p className="text-gray-400 mb-3">Doors and gates now stay in sync from both sides of the doorway, a fully rebuilt ITEMBIT flag system that was silently broken since launch, two revived-from-the-dead spells, and a new GM item-duplication command.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Doors &amp; Locks</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">LOCK</code>/<code className="text-amber-300">UNLOCK</code> now number room items the same way <code className="text-amber-300">OPEN</code>/<code className="text-amber-300">CLOSE</code>/<code className="text-amber-300">TAP</code> already did &mdash; previously they only counted the <em>lockable</em> items in a room when resolving an ordinal, so in a room with two doors, &ldquo;<code className="text-amber-300">lock 2 door</code>&rdquo; could silently miss the door you meant</li>
                  <li>Locking, unlocking, opening, or closing a door or gate now mirrors the same change onto its paired door in the room on the other side, and echoes an ambient message there (&ldquo;You see a door open.&rdquo;, &ldquo;You hear a door lock.&rdquo;, etc.) &mdash; previously the two sides could fall out of sync (unlock from one side, walk through, lock from the other &mdash; the first side stayed unlocked)</li>
                  <li><code className="text-amber-300">LOCK</code>/<code className="text-amber-300">UNLOCK</code>/<code className="text-amber-300">OPEN</code>/<code className="text-amber-300">CLOSE</code> now also announce to your own room (&ldquo;Chandra locks a door.&rdquo;) &mdash; previously these actions were completely silent to anyone standing right there</li>
                  <li><code className="text-amber-300">LOCK</code> now refuses to lock an open door (&ldquo;You must close &lt;door&gt; first.&rdquo;) instead of silently forcing it closed for you</li>
                  <li><code className="text-amber-300">@rdata</code> now shows full per-item detail (adjectives, VAL1&ndash;5, state, flags) for every item in a room instead of just its name &mdash; the only way to actually verify a door and its key share the same lock code (VAL3)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">ITEMBIT System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>ITEMBIT0&ndash;19, the per-item boolean flag system documented in the GM Manual, was never actually wired up &mdash; reading a flag collided with VAL4 (which has its own unrelated meaning), and writing one (<code className="text-amber-300">EQUAL ITEMBIT#</code>) was a silent no-op everywhere. Both now work correctly, backed by a real dedicated field</li>
                  <li>This fixes any script that was quietly relying on an ITEMBIT check and getting nothing &mdash; including the Crimson Band ring&rsquo;s War Room access check at the tapestry in the Hallway of Warriors</li>
                  <li><code className="text-amber-300">@editem</code> can now set <code className="text-amber-300">itembit0</code> through <code className="text-amber-300">itembit19</code> on an item</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Spells</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Pyrotechnics (Conjuration 141) was incorrectly dealing damage; it&rsquo;s now the harmless fireworks display it always should have been &mdash; casting it launches a volley that, over the following minute, treats every outdoor player in your region to one of 12 random firework displays roughly every 15 seconds</li>
                  <li>Mindlink (403) didn&rsquo;t do anything when cast; it now grants <code className="text-amber-300">THINK</code> (telepathic speech) for an hour, the same effect as eating a thesnia leaf or drinking a thesnia potion &mdash; castable on yourself or another player in the room</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New GM Command</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@dupe [#] &lt;item&gt;</code> &mdash; duplicates an item you&rsquo;re wielding, wearing, or carrying and places the copy on the ground, including its adjectives, VAL1&ndash;5, sharpness, hardness modifier, item bits, and (for an open container) its contents</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">IFCARRY</code> now also checks worn items, not just loose inventory &mdash; scripts gating on a worn ring, amulet, etc. were failing even when you had the item on</li>
                  <li>Fixed a data bug across five Fayd script files (the base room plus all four seasonal variants) where a value edit (<code className="text-amber-300">VAL3=1234</code>) glued directly against a trailing comment with no space was silently dropped by the parser &mdash; also removed a stray duplicate, unedited copy of the same door item left over in the spring script</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.16.0 &mdash; July 21, 2026</h2>
            <p className="text-gray-400 mb-3">Unconsciousness replaces instant death at 0 body points, a real monster stun/knockdown system, a game-wide fix for delayed scripted sequences silently dropping, and a new reroll charm for legacy characters.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Death &amp; Unconsciousness</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Dropping to exactly 0 body points from a weapon hit or spell now knocks you unconscious (laying down, out cold, same presentation as being put to sleep) instead of killing you outright &mdash; only a hit that would drive you below 0 is lethal</li>
                  <li>Bleeding, poison, and disease never kill outright on the tick that first drops you to 0 &mdash; you&rsquo;re knocked unconscious instead, giving someone else a chance to <code className="text-amber-300">TEND</code> you or cast Body Restoration/Cure Poison/Cure Disease. If the same condition is still active and you&rsquo;re still unconscious at 0 when it ticks again, that tick is lethal</li>
                  <li>Unconscious players are locked out of every command except <code className="text-amber-300">LOOK</code>/<code className="text-amber-300">WHO</code>/<code className="text-amber-300">QUIT</code>/<code className="text-amber-300">STATUS</code>/<code className="text-amber-300">HEALTH</code>/<code className="text-amber-300">HELP</code> &mdash; no speech, no actions &mdash; until healed or natural regeneration brings them back above 0</li>
                  <li>Room listings and <code className="text-amber-300">LOOK AT</code> now correctly show &ldquo;(unconscious)&rdquo;/&ldquo;is unconscious&rdquo; instead of lumping it in with ordinary &ldquo;lying down&rdquo;</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Monster Combat AI</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Stun from an excellent hit is now a real timed status (3&ndash;6 seconds) that actually blocks a monster from attacking, moving, or fleeing for its whole duration &mdash; previously it only skipped exactly one action before immediately resuming as normal</li>
                  <li>New Knocked Down status &mdash; an alternative outcome to stun on an excellent hit (50/50 split) &mdash; costs a monster one full turn getting back on its feet before it can act again, the same way a monster already had to stand up after being put to sleep</li>
                  <li>A monster that bleeds to death (or dies to Siryx&rsquo;s Terrible Tentacles) now correctly awards kill experience to whoever landed the last hit, split evenly among their group if they&rsquo;re grouped with others who aren&rsquo;t hidden or invisible &mdash; previously a bleed-out death awarded no experience to anyone at all</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed a game-wide bug where a scripted item&rsquo;s delayed continuation (<code className="text-amber-300">PLREVENT</code>/<code className="text-amber-300">CONTPLREVENT</code>) was silently dropped after the first pause in <code className="text-amber-300">RUB</code>/<code className="text-amber-300">TAP</code>/<code className="text-amber-300">TOUCH</code>/<code className="text-amber-300">PULL</code>/<code className="text-amber-300">PUSH</code>/<code className="text-amber-300">TURN</code>/<code className="text-amber-300">SEARCH</code>/<code className="text-amber-300">DIG</code>, <code className="text-amber-300">WORK</code>, <code className="text-amber-300">EAT</code>, <code className="text-amber-300">DRINK</code>, <code className="text-amber-300">FLIP</code>, <code className="text-amber-300">READ</code>, <code className="text-amber-300">GET</code>, <code className="text-amber-300">LOOK</code>/<code className="text-amber-300">EXAMINE</code>, <code className="text-amber-300">GO</code>, <code className="text-amber-300">STEAL</code>, and <code className="text-amber-300">CLIMB</code> &mdash; any multi-stage scripted sequence using a delay (e.g. a fountain&rsquo;s dawn-triggered dance) would just stop after its first line</li>
                  <li>Fixed a related bug where putting an item into a scripted container/device (e.g. a garment-finishing contraption) discarded the item&rsquo;s own adjectives/values before its script ran, breaking material checks, and where the device&rsquo;s success branch never actually ran because it wasn&rsquo;t recognized as having &ldquo;handled&rdquo; the action</li>
                  <li>Weather no longer shows Snow Flurries through Blizzard during Summer while reporting a warm temperature &mdash; the weather-type simulation previously had no concept of season at all</li>
                  <li>Keys no longer falsely open a lock that was never assigned a code &mdash; both sides defaulting to an unset value no longer counts as a match</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Drakin</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Drakin now have natural scale armor that strengthens with level (since they can&rsquo;t wear armor at all), and take 25% additional damage from heat and cold &mdash; both documented racial traits that were never actually implemented</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Item</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Reroll charm &mdash; a GM-distributed item for characters created before stat rerolling existed at character creation. <code className="text-amber-300">RUB</code> it to preview a fresh set of stats as many times as you like, then <code className="text-amber-300">CONCENTRATE</code> on it to lock them in; it crumbles to dust once used</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@set</code> can now edit a player&rsquo;s <code className="text-amber-300">AGE</code>, <code className="text-amber-300">HEIGHT</code>, and <code className="text-amber-300">WEIGHT</code>, and can now write any named script variable (e.g. quest flags) the same way <code className="text-amber-300">@peek</code> could already read them</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Characters</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Starting gold raised from 5 to 20</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.15.0 &mdash; July 20, 2026</h2>
            <p className="text-gray-400 mb-3">Character creation overhaul &mdash; age, eye/hair/skin appearance, and a reroll-until-you&rsquo;re-happy stat system, plus <code className="text-amber-300">STAT</code> and <code className="text-amber-300">EXAMINE</code> rebuilt to match the original game&rsquo;s output, and a scripting bug fix.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Character Creation</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>New characters now roll Age, and choose Eye Color, Skin Color, Hair Style and Hair Color from wide fantasy-appropriate lists (12 hair styles including a Bald option that skips hair color entirely) &mdash; available on the web client, telnet, and SSH</li>
                  <li>Stats (Strength/Agility/Quickness/Constitution/Perception/Willpower/Empathy) plus Height/Weight/Age can now be rerolled as many times as you like before locking them in, matching the original game&rsquo;s &ldquo;reroll until you like what you see&rdquo; character creation &mdash; still respects each race&rsquo;s stat ranges</li>
                  <li>Fixed a telnet/SSH bug where choosing &ldquo;1) Male&rdquo; at character creation actually created a Female character internally, and choosing &ldquo;2) Female&rdquo; was rejected outright</li>
                  <li>Existing characters created before this update are seamlessly backfilled with a rolled Age and appearance the next time they log in</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">STAT &amp; EXAMINE</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">STAT</code> reformatted to match the original layout, verified against a 1996 session capture &mdash; guild rank/title shown first (e.g. &ldquo;You are a high master of the Crimson Band.&rdquo;), followed by Name/Race/Gender, Level, build points, stats grouped Quickness/Constitution/Strength/Agility and Willpower/Perception/Empathy, an Age/Height/Weight/Load line, and (previously missing entirely) Body Points/Mana/Psi/Fatigue</li>
                  <li><code className="text-amber-300">EXAMINE</code> (on yourself or another player) now builds a real physical description from your stats and appearance choices, e.g. &ldquo;You see Shirla Rennay, a young female aelfen. She is tall, light weight and robust. She has blue eyes, fair skin and long, flowing golden blond hair.&rdquo; &mdash; height/weight/build descriptors are derived from your actual stats rather than being random</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed item scripts with a verb-gated block nested inside a value-setting <code className="text-amber-300">IFVAR</code> tree (like the lens of worlds&rsquo; <code className="text-amber-300">SHOWROOM</code> preview) firing on <em>any</em> interaction instead of only the intended verb &mdash; e.g. <code className="text-amber-300">POINT</code>ing at the lens no longer shows the hidden room it&rsquo;s only supposed to reveal on <code className="text-amber-300">EXAMINE</code></li>
                  <li>Other players now see &ldquo;&lt;name&gt; incants a spell.&rdquo; when someone prepares a spell, instead of &ldquo;begins preparing a spell&rdquo;</li>
                  <li>Rorin&rsquo;s Fire now has its own cast flavor text (a wave of red and orange flame that hisses and constricts &ldquo;like a snake&rdquo;) instead of the generic fire-bolt message</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.14.0 &mdash; July 19, 2026</h2>
            <p className="text-gray-400 mb-3">Combat hit messages rebuilt with proper single-word severity descriptors and special elemental death flavor text, plus a batch of crash and quest-blocking fixes and a new weapon hardness modifier for GMs.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat Messages</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Hit messages now show a proper single capitalized severity word (e.g. &ldquo;Minor burn to right leg. [17 Damage]&rdquo;) instead of the multi-word, lowercase persistent-wound vocabulary meant for the <code className="text-amber-300">HEALTH</code> command (e.g. &ldquo;slightly lacerated slash to right arm&rdquo;)</li>
                  <li>Severity is now correctly based on damage as a percentage of the target&rsquo;s max body points, matching original session-log evidence &mdash; the same raw damage number can be &ldquo;Minor&rdquo; against a tough monster and &ldquo;Ghastly&rdquo; against a weak one</li>
                  <li>A killing blow from a cold or heat spell (or a weapon&rsquo;s elemental crit) now shows a special description of the death itself &mdash; e.g. &ldquo;Chilly body barrage solidifies muscle tissue.&rdquo; or &ldquo;Dazzling explosive display carbonizes bones and flesh.&rdquo; &mdash; in place of the normal severity/damage line, matching original wording</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@additem &lt;archetype#&gt; [val1=N] &hellip; [adj1=N] &hellip;</code> now accepts value/adjective modifiers when spawning an item, so quest items that depend on a specific starting value (e.g. the lens of worlds) can actually be set up correctly for testing</li>
                  <li>Weapons now have a GM-editable hardness modifier &mdash; <code className="text-amber-300">@editem &lt;weapon&gt; hardnessmod &lt;N&gt;</code> adjusts Weapon Clash break-resistance on top of the weapon&rsquo;s normal weight/adjective-based value, visible via <code className="text-amber-300">@iexamine</code>, and survives being dropped and picked back up</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed a crash-causing bug where a portal script using <code className="text-amber-300">MOVEGROUP</code> (rather than <code className="text-amber-300">MOVE</code>) &mdash; like the hidden hole on Island &mdash; never actually moved anyone through, and could misroute its departure message to the wrong room</li>
                  <li>Fixed a server crash when stepping through a secret passage revealed by <code className="text-amber-300">KNOCK</code>ing (e.g. the Matriarch Tree basement) &mdash; the passage script removing itself from the room after use could invalidate the room&rsquo;s item list out from under the very next line of code</li>
                  <li>Invisible GMs no longer leak a script&rsquo;s room broadcast (e.g. a portal&rsquo;s custom &ldquo;goes through&rdquo; text) to other players when triggering it &mdash; matching the silence already applied to ordinary movement</li>
                  <li>Fixed <code className="text-amber-300">GET</code> on an item with a non-blocking script (e.g. the lens of worlds&rsquo; first-touch binding ceremony, or a cursed item&rsquo;s magical backlash) never actually completing the pickup, and fixed a related crash when such a script destroys the item itself as a side effect (e.g. &ldquo;crumbles into dust&rdquo;)</li>
                  <li><code className="text-amber-300">FLIP</code> now checks an item&rsquo;s own script before requiring it be physically flippable &mdash; fixes puzzle items like a combination-lock knob that only responds to a room-defined <code className="text-amber-300">FLIP</code> script</li>
                  <li>When a monster dies, every player who was fighting it is now taken out of combat automatically &mdash; previously only the player who landed the killing blow was disengaged, leaving groupmates (or anyone who died to bleed-out rather than a direct hit) stuck needing to manually <code className="text-amber-300">RETREAT</code></li>
                  <li>Wounds on undead players and monsters no longer bleed &mdash; they have no blood to lose</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.13.0 &mdash; July 17, 2026</h2>
            <p className="text-gray-400 mb-3">New weapon specialization system, and a full hit-location damage &amp; wound-tracking overhaul &mdash; real bleeding, per-wound severity descriptions, and TEND reworked to heal wound-by-wound.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">SPECIALIZE &lt;weapon&gt;</code> &mdash; spend build points to specialize in the weapon you&rsquo;re wielding (Crushing, Edged, Drakin, Polearms, or Thrown Weapons only; requires 10+ skill in that weapon&rsquo;s category). Each of the 5 ranks reduces the fatigue cost of attacking with that weapon by 1 (never below 1) and shifts 5% of your hit-location odds toward the head and body. <code className="text-amber-300">SPECIALIZE</code> alone lists your current specializations, and they now appear in the <code className="text-amber-300">SKILLS</code> table</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat &amp; Wounds</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Hits now land on a specific body part with real consequences &mdash; strikes to the head, body, or back deal full damage; arms and legs take 40%; hands, paws, and tails take 20% &mdash; derived from an original session-log analysis</li>
                  <li>Wounds are now individually tracked on both players and monsters, each described with vocabulary matching the weapon that caused it &mdash; slash (nicked &rarr; cut &rarr; lacerated &rarr; gashed), puncture (pricked &rarr; stabbed &rarr; punctured &rarr; gored), crush (scuffed &rarr; bruised &rarr; battered &rarr; crushed &rarr; ruptured), and burn (singed &rarr; scorched &rarr; burned &rarr; charred) for heat/cold/electric damage &mdash; 12 severity levels each</li>
                  <li>Slash and puncture wounds beyond the mildest two levels now cause real bleeding &mdash; body points drain away once a minute until the wound is treated, and can kill if ignored long enough, for both players and monsters</li>
                  <li><code className="text-amber-300">HEALTH</code> and <code className="text-amber-300">EXAMINE</code> now describe accumulated wounds by location, e.g. &ldquo;You have a nicked head, a nicked body, a nicked and slightly punctured back&hellip;&rdquo;</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Healing</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">TEND</code> no longer restores a flat chunk of body points &mdash; it removes one wound at a time, always the least severe first, healing exactly as many body points as that wound&rsquo;s severity level (same-race targets still get the existing +50% bonus)</li>
                  <li>Higher Healing skill is required to treat more severe wounds &mdash; skill 20 can treat any severity, and a healer who isn&rsquo;t skilled enough for the least severe wound present is turned away rather than skipping ahead to an easier one</li>
                  <li>The target being tended must now be sitting or lying down, but no longer needs to be alive &mdash; <code className="text-amber-300">TEND</code> now also works on a fresh corpse, player or monster</li>
                  <li>Tending a wound now awards the healer experience scaled to its severity</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>New <code className="text-amber-300">@specialize &lt;player&gt; [&lt;weapon&gt; &lt;level&gt;]</code> &mdash; list or directly set a player&rsquo;s weapon specialization rank (0&ndash;5), spending or refunding build points as the level changes, the same way <code className="text-amber-300">@mastery</code> already works for spells</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Psionic Teleportation (<code className="text-amber-300">PROJECT &lt;mark&gt;</code>) always sent you to mark 1 no matter which mark number you specified &mdash; it now correctly honors marks 1&ndash;10, matching Bend Space I</li>
                  <li>Thrown weapons (javelins, spears, throwing daggers, etc.) were training your Missile Weapons skill instead of Thrown Weapons for to-hit purposes &mdash; fixed to use the correct skill, matching the original game&rsquo;s distinct bow vs. thrown-weapon skills</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.12.0 &mdash; July 16, 2026</h2>
            <p className="text-gray-400 mb-3">New WEATHER command and live temperature system, matching GM weather override, a weapon-value economy fix, and original-style spell messages for the defense buffs.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">WEATHER</code> shows the current conditions and temperature for your region &mdash; the condition name plus a descriptive line (e.g. &ldquo;A light rain falls steadily, pattering softly on the ground.&rdquo;) when outdoors, or just an ambient temperature reading when indoors</li>
                  <li>Temperature is now a live value derived from season, time of day, and the region&rsquo;s current weather state, rather than a fixed or absent reading</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>New <code className="text-amber-300">@weather [value]</code> &mdash; shows the numeric weather state and temperature for your region with no argument, or sets the region&rsquo;s weather (0&ndash;14) and broadcasts the transition to outdoor players there, same as a natural weather change</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed a long-standing bug where the <code className="text-amber-300">WEA</code> script variable (used by <code className="text-amber-300">IFVAR WEA</code> checks in world scripts) always read region 0&rsquo;s weather no matter where a script was actually running &mdash; weather-gated content in every other region (e.g. rain-triggered foraging on Island) could never correctly detect rain</li>
                  <li>The non-magical sharpness bonus a forged weapon gets from Weaponsmithing was being stored in the same field read as the item&rsquo;s copper sell value &mdash; freshly forged weapons were selling for only a few coppers instead of a real price; sharpness now has its own dedicated field, and existing forged weapons in the database were corrected with a one-time migration</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">SKILLS</code> (abbr. <code className="text-amber-300">SKILL</code>) now renders as a fixed-width table (<code className="text-amber-300">#</code>, Skill, Level columns) matching the original game&rsquo;s output, instead of a simple bulleted list</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Spellcasting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">PREPARE</code> now shows &ldquo;You prepare the &lt;spell&gt; spell.&rdquo;, matching original wording</li>
                  <li>Mystic Armor, Globe of Protection, Mass Protection, and Spectral Shield now show original-style self-cast messages &mdash; &ldquo;You gesture.&rdquo; &rarr; the success roll &rarr; a spell-specific flavor line (e.g. &ldquo;A prismatic globe encircles you.&rdquo;) &mdash; instead of one generic combined line</li>
                  <li>Examining another player now shows a spell-specific line for these four buffs (e.g. &ldquo;She is outlined in glowing armor.&rdquo;) instead of the same generic &ldquo;shimmering magical aura&rdquo; line used for every defense spell</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.11.0 &mdash; July 15, 2026</h2>
            <p className="text-gray-400 mb-3">Script engine execution-order rework, monster dialogue scripting, and the full &ldquo;Enter the Fold&rdquo; lens-of-worlds quest chain fixed end-to-end.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Script Engine</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Rebuilt how conditional script blocks execute &mdash; actions and nested <code className="text-amber-300">IFVAR</code> checks now run in their original source order instead of every flat action always running before any nested block, fixing scripts that pick a new value and then reference it later in the same block</li>
                  <li><code className="text-amber-300">ELSE</code> branches now correctly support <code className="text-amber-300">PLREVENT</code>/<code className="text-amber-300">SETEVENT</code> delayed continuations &mdash; previously anything after a delay inside an ELSE branch ran immediately instead of waiting</li>
                  <li><code className="text-amber-300">IFSAY</code> blocks nested inside a room&rsquo;s <code className="text-amber-300">IFVAR</code> tree (multi-stage conversations gated on quest flags) are now reachable &mdash; previously only top-level <code className="text-amber-300">IFSAY</code> blocks could ever match</li>
                  <li><code className="text-amber-300">KNEEL</code>/<code className="text-amber-300">SIT</code>/<code className="text-amber-300">STAND</code>/<code className="text-amber-300">LAY</code> and spoken commands (<code className="text-amber-300">SAY</code>/<code className="text-amber-300">&apos;</code>) now correctly schedule delayed script continuations (<code className="text-amber-300">PLREVENT</code>/<code className="text-amber-300">CONTPLREVENT</code>) instead of silently dropping everything after the delay</li>
                  <li>Fixed a parser bug where a <code className="text-amber-300">CEVENT</code> (cyclic world event) immediately followed by a <code className="text-amber-300">MACRO</code> definition would swallow the entire macro into its own body and fire it on a timer, unconditionally, with no acting player</li>
                  <li>Added support for <code className="text-amber-300">CALL N</code> as an inline, mid-script subroutine call (e.g. resolving a quest item&rsquo;s name for display on demand) &mdash; previously only the static room/item-level <code className="text-amber-300">CALL</code> attachment worked</li>
                  <li>Monsters can now have their own <code className="text-amber-300">EXAMINE</code>/<code className="text-amber-300">GIVE</code>/etc. scripts attached via <code className="text-amber-300">SCRIPTMACRO</code> &mdash; this directive was silently ignored before, so no monster dialogue script ever ran</li>
                  <li><code className="text-amber-300">GIVE &lt;item&gt; TO &lt;monster&gt;</code> now actually works &mdash; previously GIVE only ever considered player targets</li>
                  <li><code className="text-amber-300">IFCARRY</code> now resolves variable arguments (not just literal numbers) and checks all three adjective slots instead of only the first &mdash; fixes quest checks for store-bought items, whose adjective is stored in the third slot</li>
                  <li>Text set via <code className="text-amber-300">STRCPY</code> now converts underscores to spaces (e.g. &ldquo;some_meteoric_dust&rdquo; &rarr; &ldquo;some meteoric dust&rdquo;), matching how <code className="text-amber-300">IFSAY</code> patterns already worked</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Ming-K&rsquo;Tuk &amp; The Fold</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>The riddle trial in the Phosphorescent Catacombs (speaking the pass phrases, then kneeling at the altar) now actually teleports you onward instead of just running a normal kneel</li>
                  <li>Ming-K&rsquo;Tuk will now accept tribute, ask his questions, and grant passage through his lair when properly flattered &mdash; the entire conversation and tribute sequence was previously unreachable</li>
                  <li>The &ldquo;Enter the Fold&rdquo; lens-of-worlds scavenger hunt (Cellar &rarr; Beyond the Breach &rarr; The Void) now works start to finish &mdash; turning in a quest component is correctly detected, the hint for the next component names it correctly, and the finished lens is awarded after the full trial</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>New <code className="text-amber-300">@give [#] &lt;item&gt; to &lt;player&gt;</code> and <code className="text-amber-300">@take [#] &lt;item&gt; from &lt;player&gt;</code> &mdash; silently move an item between a GM&rsquo;s inventory and a player&rsquo;s, with ordinal and adjective matching</li>
                  <li>Fixed <code className="text-amber-300">@whisper</code> &mdash; it reported the message as sent but never actually delivered it to the target</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Weapons and jewelry crafted from oiled/oily metal now correctly show both the iridescent and oiled adjectives (needed to avoid round-time in certain caves) instead of losing the oiled adjective entirely</li>
                  <li>Mortar (used in alchemy) was incorrectly classified as an instantly-created Weaponsmithing item &mdash; it&rsquo;s now properly a Jeweler craft using the same <code className="text-amber-300">CRAFT</code> &rarr; <code className="text-amber-300">WORK</code> sequence as its pestle counterpart</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">SNEAK</code> could never actually keep you hidden while moving, win or lose &mdash; fixed</li>
                  <li>Enchantment spells no longer overwrite an item&rsquo;s existing adjective when all three adjective slots are already full &mdash; the magical bonus is still applied, the item&rsquo;s look is just left alone</li>
                  <li><code className="text-amber-300">OPEN</code>/<code className="text-amber-300">CLOSE</code> on an already-open or already-closed item now says so instead of silently repeating the action</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">World</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Sharkhor spawn rate along the Inner Sea shore increased slightly (10% &rarr; 15% per check)</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.10.0 &mdash; July 14, 2026</h2>
            <p className="text-gray-400 mb-3">Alchemy &amp; potion brewing rebuilt from the ground up, plus fixes to spell targeting, item script dispatch, and Wood Lore crafting.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Alchemy &amp; Brewing</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Reagent recognition rebuilt &mdash; <code className="text-amber-300">BREW</code> now identifies every catalyst and reagent by its actual name (e.g. mandrake root, babich root, muur crystal), matching the original &ldquo;Art of Alchemy&rdquo; reference notes, instead of a per-instance field almost no real ingredient in the world ever had set</li>
                  <li>Brewing order no longer matters &mdash; the catalyst and two reagents can be added in any order, as documented</li>
                  <li>Fixed reagent/container name matching so <code className="text-amber-300">BREW MANDRAKE ROOT IN FLASK</code> and similar actually find the named items</li>
                  <li>Potion sip count now matches the container&rsquo;s real capacity (Vial 2, Flask 5, Flagon 6, Bottle 10, Ewer 6) instead of a random 2&ndash;5 regardless of vessel</li>
                  <li>Brewed potions now get a random liquid-appearance adjective like any other potion (e.g. &ldquo;a fuming potion&rdquo;) instead of showing as plain &ldquo;some liquid&rdquo; when examined</li>
                  <li>Removed the reference &ldquo;Bottle Color&rdquo; column from the recipe list and completion messages &mdash; that was only the original compiling player&rsquo;s personal notes, not a game mechanic</li>
                  <li>Each <code className="text-amber-300">BREW</code> step now takes a 15-second round (7 sec under Haste), matching other crafting skills</li>
                  <li>Successfully brewing a potion now awards experience scaled to its recipe level (level &times; 20), the same formula used for jewelry/wood/weaving crafts</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Spellcasting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">CAST</code> now supports ordinal targeting for item-targeted spells like Enchantment &mdash; <code className="text-amber-300">CAST 2 RAPIER</code> correctly enchants the second rapier instead of failing</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Item Scripts</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">LIGHT</code> and <code className="text-amber-300">EXTINGUISH</code> now check a named item&rsquo;s own script before falling back to standard lighting logic</li>
                  <li><code className="text-amber-300">PLAY</code> now runs a named instrument&rsquo;s own script instead of always printing generic flavor text &mdash; fixes instruments like the teak flute that have distinct wielded/not-wielded text</li>
                  <li>Script engine now supports <code className="text-amber-300">IFITEM -1 WIELDED</code> checks (previously only WORN was recognized), and wielded/off-hand items are now correctly tagged when their scripts run</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Wood Lore &amp; Crafting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed a parser bug where FLETCHER-tagged rooms (bowyer/fletcher shops) were never recognized as valid crafting workshops, blocking Wood Lore <code className="text-amber-300">CRAFT</code> entirely</li>
                  <li>Fixed several Wood Lore items (musical instruments, staves, and other non-launcher items) awarding zero experience on completion due to missing source data &mdash; they now fall back to a sensible difficulty-based reward</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.9.0 &mdash; July 13, 2026</h2>
            <p className="text-gray-400 mb-3">New potions system &mdash; random potion drops, containers, alchemy analysis, and pouring &mdash; plus several spell-duration fixes and a new APPEARANCE command.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Potions</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Potion containers (bottle, flask, vial) now drop as loot from monster kills or turn up inside chests, filled with 2&ndash;10 sips of a random spell scaled to the monster&rsquo;s treasure level</li>
                  <li>Flask and vial get a random material adjective (glass, jade, obsidian, crystal, etc.); every container reveals a random liquid appearance (crimson, cloudy, fizzing, reeking, etc.) once opened</li>
                  <li><code className="text-amber-300">SIP</code>/<code className="text-amber-300">DRINK</code> now actually casts the potion&rsquo;s spell instead of printing &ldquo;[Spell effect coming soon.]&rdquo;</li>
                  <li><code className="text-amber-300">LOOK IN</code> and plain <code className="text-amber-300">EXAMINE</code> on a potion container report fullness (full, 3/4 full, half full, 1/4 full, almost empty) and describe the liquid inside</li>
                  <li>New <code className="text-amber-300">POUR &lt;container&gt; INTO &lt;container&gt;</code> command to transfer liquid between containers</li>
                  <li><code className="text-amber-300">ANALYZE</code> can now identify a potion&rsquo;s magical properties for players with Alchemy skill</li>
                  <li>Potions (and items generally) sitting inside an open container can now be targeted directly by <code className="text-amber-300">SIP</code>, <code className="text-amber-300">EXAMINE</code>, <code className="text-amber-300">ANALYZE</code>, <code className="text-amber-300">@iexamine</code>, and <code className="text-amber-300">@editem</code> &mdash; including by the potion&rsquo;s own color rather than the vessel holding it (e.g. <code className="text-amber-300">SIP CRIMSON POTION</code>)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Spell Duration Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fly (224) and the new Heat Shield/Cold Shield (507/508) are now proper 20-minute timed buffs like other spells, extending on recast instead of lasting forever or doing nothing at all</li>
                  <li>Heat Shield and Cold Shield now actually reduce heat and cold damage taken by 50% while active</li>
                  <li>Potion/scroll/wand-triggered Strength, Agility, Mystic Armor, and defense-spell buffs are now temporary (20 minutes, extending on recast) instead of permanent stacking increases</li>
                  <li>Psionic Flight (Mind over Matter 10) no longer persists across logout, and players no longer get stuck &ldquo;hovering in the air&rdquo; after landing</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">APPEARANCE</code> (abbr. <code className="text-amber-300">APPEAR</code>) lets you set a custom line shown when others examine you, appended after your worn equipment</li>
                  <li><code className="text-amber-300">COMMAND LOOK</code> lets a summoned creature look around its room and report back</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.8.0 &mdash; July 11, 2026</h2>
            <p className="text-gray-400 mb-3">New TARGET command for multi-target spellcasting; Chain Lightning, Flaming Arrows, and Siryx&rsquo;s Terrible Tentacles rebuilt to hit every creature you&rsquo;ve targeted.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">TARGET Command</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">TARGET &lt;creature&gt;</code> (or <code className="text-amber-300">TAR</code>) builds a list of up to 6 creatures in the room for multi-target spells &mdash; supports ordinal disambiguation (<code className="text-amber-300">TARGET 2 werewolf</code>) just like ATTACK</li>
                  <li>Targeting the same creature twice is rejected (&ldquo;That is already being targeted.&rdquo;), and the list reports when it&rsquo;s full (6/6)</li>
                  <li>Targets are automatically dropped from the list if they die, flee, or otherwise leave the room, freeing a slot for a new target</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Chain Lightning (Conjuration 132)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>With an active TARGET list, the bolt now arcs from you to the named target and then chains through every other targeted creature in turn, each taking its own independently-rolled damage</li>
                  <li>Falls back to a single bolt at one target if no TARGET list has been built, same as before</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Flaming Arrows (Conjuration 131)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Now fires one flaming arrow at every creature in your TARGET list, each independently rolled for damage</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Siryx&rsquo;s Terrible Tentacles (Conjuration 134)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>New immobilize spell &mdash; black tentacles burst from the ground and grab hold of every creature in your TARGET list (or a single named target), with no body-point size limit on what it can restrain</li>
                  <li>Entangled creatures can&rsquo;t attack or flee, and take crushing damage once every minute until they die, break free, or the spell expires</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.7.0 &mdash; July 10, 2026</h2>
            <p className="text-gray-400 mb-3">Necromancy overhaul &mdash; undead-only Turn/Destroy Undead, Control/Animate Undead, Reconstruction, Regeneration, Speak with Dead, Summon Spectral Warrior; SAY command and &ldquo;.&rdquo; repeat-last-command.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Undead-Only Spells</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Turn Undead I/II (301/302) and Destroy Undead I/II/III (339&ndash;341) now only affect actual undead creatures (RACE 22, e.g. skeletons and zombies) &mdash; casting on a living creature now fizzles with no effect instead of dealing damage</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Control Undead I/II (Necromancy 308/309)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Dominate an existing undead creature in the room &mdash; <code className="text-amber-300">CAST control undead i &lt;creature&gt;</code> &mdash; and command it with the same <code className="text-amber-300">COMMAND FOLLOW/GUARD/ATTACK/BEGONE</code> verbs used on summoned elementals</li>
                  <li>Control Undead I only works on undead with 100 or fewer body points; Control Undead II raises the cap to 200</li>
                  <li>Control lasts 40 minutes &mdash; afterward the undead breaks free of its bonds and turns hostile again</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Animate Skeleton / Animate Zombie (Necromancy 306/307)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Summon a fresh skeleton or zombie fully under your control, permanent until dismissed &mdash; same command set as a summoned elemental</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Summon Spectral Warrior (Necromancy 353)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Summons a spectral warrior under your command &mdash; requires some ghoul dust as a reagent, consumed at <code className="text-amber-300">PREPARE</code></li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Undead Healing &amp; Harm</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Body Restoration I/II/III (316&ndash;318) now sear an undead target with holy energy as damage instead of healing them</li>
                  <li>Reconstruction (337) now only heals undead targets &mdash; casting it on a living creature fizzles with no effect</li>
                  <li>Regeneration (343) is now a heal-over-time spell: heals once immediately on cast, then heals the same amount again once per minute for 5 more minutes</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Speak with Dead (Necromancy 311)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Cast on a dead player&rsquo;s body to grant them the power of speech again, even though they remain otherwise incapacitated until DEPART or resurrection</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">SAY &lt;message&gt;</code> now works as an alias for <code className="text-amber-300">'&lt;message&gt;</code> / <code className="text-amber-300">"&lt;message&gt;</code>, including automatic ask/exclaim detection and speech-manner overrides</li>
                  <li>Entering <code className="text-amber-300">.</code> by itself now repeats your last entered command (e.g. <code className="text-amber-300">attack fire giant</code> then <code className="text-amber-300">.</code> attacks again)</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.6.8 &mdash; July 8, 2026</h2>
            <p className="text-gray-400 mb-3">INLAY and INSET jeweler commands, spell reagent fixes, and a new Breath of Life resurrection spell.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Jeweler Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Added <code className="text-amber-300">INLAY &lt;item&gt; WITH &lt;gem&gt;</code> and <code className="text-amber-300">INSET &lt;item&gt; WITH &lt;gem&gt;</code> alongside <code className="text-amber-300">ENCRUST</code> &mdash; same requirements (Jeweler 3, forge/workshop, 2 free adjective slots), different resulting adjective (e.g. an emerald inlaid ring, a sapphire inset amulet)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Spell Reagents</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed Enchantment II and Enchantment III incorrectly demanding a spider eye/imp toe at cast time even when the spell was chanted from a scroll instead of self-prepared &mdash; reagents are now only required when a player prepares and casts the spell themselves</li>
                  <li>Closed a loophole where casting a reagent-requiring spell in one step (<code className="text-amber-300">CAST &lt;spell&gt; &lt;target&gt;</code> without a prior <code className="text-amber-300">PREPARE</code>) skipped the reagent check entirely &mdash; it now directs you to <code className="text-amber-300">PREPARE ... WITH ...</code> instead</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Breath of Life (Necromancy 305)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>New resurrection spell &mdash; requires some mandrake root as a reagent (consumed at <code className="text-amber-300">PREPARE</code>), then <code className="text-amber-300">CAST &lt;dead player&gt;</code> to raise them where they fell, restoring 1-10 body points</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.6.7 &mdash; July 5, 2026</h2>
            <p className="text-gray-400 mb-3">Guard-redirected combat message routing, Crescent muldragun lair over-spawning, GM ordinal targeting for lock commands.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed guard-redirected monster attacks not sending the defender their own <code className="text-amber-300">[ToHit: X, Roll: Y]</code> detail &mdash; when a player guards another and an attack is redirected to them, the guard now sees their private combat roll instead of only the room&rsquo;s simplified broadcast line</li>
                  <li>Guard-redirected combat now saves the correct player&rsquo;s health and status afterward &mdash; was previously saving the original target instead of the guard who actually took the hit</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Monster Spawning</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed the Crescent muldragun/bhulondag lair spawning far too many monsters at once &mdash; duplicate <code className="text-amber-300">MLIST</code> spawn entries defined in both MONSTERS.SCR and HAVEN.SCR were being loaded twice, doubling every spawn roll for that group</li>
                  <li>Monster groups no longer fill to their full population cap in a single check &mdash; each check now adds at most one monster per spawn entry, so numbers build up gradually instead of bursting in all at once</li>
                  <li>Periodic respawn checks slowed roughly 3.5&times; (about every 30 seconds &rarr; about every 105 seconds)</li>
                  <li>Added a 20-second per-room spawn cooldown &mdash; a group of players arriving together, or rapidly leaving and re-entering, no longer triggers a separate spawn roll for every single arrival</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@unlock</code>/<code className="text-amber-300">@lock</code>/<code className="text-amber-300">@open</code>/<code className="text-amber-300">@close</code> now support ordinal targeting (e.g., <code className="text-amber-300">@unlock 2 door</code>) to disambiguate when multiple matching items are in the room</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.6.6 &mdash; July 3, 2026</h2>
            <p className="text-gray-400 mb-3">Spell cast message overhaul &mdash; original per-spell flavor text, caster/onlooker perspective split, and shared damage lines.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Spell Flavor Text</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Damage spells with no custom text now correctly fall back to generic per-damage-type flavor (bolt of energy, ball of flame, freezing sphere, bolt of lightning, force blast) &mdash; and the caster now sees a second-person line (&ldquo;You form a freezing sphere&hellip;&rdquo;) while onlookers see the third-person version (&ldquo;Chandra forms a freezing sphere&hellip;&rdquo;)</li>
                  <li>Call Meteor (112) restored to its original two-part effect &mdash; hammers the target with independently-rolled heat and crushing damage, each shown as its own damage line (&ldquo;&hellip;burn to &hellip;&rdquo; / &ldquo;&hellip;blow to &hellip;&rdquo;)</li>
                  <li>Frost Ray (120), Lightning Bolt (103), Spectral Sword (345), and Earth Spike (523) now use their original spell-specific cast text instead of the generic elemental fallback</li>
                  <li>Web spell now shows &ldquo;&lt;target&gt; is covered with strands of sticky webbing!&rdquo; to both caster and room, replacing the old generic entangle text</li>
                  <li>Onlookers in the room now see the same damage line as the caster (e.g. &ldquo;Minor blast to body. [20 Damage]&rdquo;) for every damage spell, instead of only the caster seeing the result</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.6.5 &mdash; July 2, 2026</h2>
            <p className="text-gray-400 mb-3">Foraging overhaul (terrain tables, Wood Lore gating, skill-scaled rarity), crafting material search fix, free player-taught training.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Foraging</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed a script-loading bug that silently dropped almost every real <code className="text-amber-300">FORAGEDEF</code> entry &mdash; seasonal terrain tables (forest/mountain/plain/swamp) and the jungle table were never being loaded, so <code className="text-amber-300">FORAGE</code> fell back to generic bare-noun items with no adjectives</li>
                  <li>Foraged items now correctly carry their proper adjective (e.g. &ldquo;yulman leaf&rdquo; instead of just &ldquo;leaf&rdquo;)</li>
                  <li><code className="text-amber-300">FORAGE</code> now requires Wood Lore skill (18) &mdash; &ldquo;You have no training in Wood Lore.&rdquo; if untrained</li>
                  <li>Success chance now scales with Wood Lore skill and Perception (base 30%, capped 90%), with a 10-second round time</li>
                  <li>Higher Wood Lore skill biases the odds toward rarer finds (e.g. mandrake root) instead of only ever turning up common plants</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">WORK &lt;material&gt;</code> now keeps searching your inventory when it finds a matching-but-unsuitable item instead of giving up immediately &mdash; e.g. a finished &ldquo;cotton jacket&rdquo; no longer blocks raw cotton later in your pack from being found</li>
                  <li>Material matching now recognizes fully-qualified names (e.g. <code className="text-amber-300">WORK BROWN SNAKE SKIN</code>) to disambiguate between multiple similarly-named materials, in addition to a bare noun like <code className="text-amber-300">WORK SKIN</code></li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Training</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">TRAIN</code> no longer charges gold for ranks made available by a fellow player&rsquo;s <code className="text-amber-300">TEACH</code> &mdash; only ranks within an organization&rsquo;s own posted training cap are charged</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.6.4 &mdash; June 30, 2026</h2>
            <p className="text-gray-400 mb-3">Jewelry, weaving, and wood lore multi-step crafting; ENCRUST and ENGRAVE commands for Jewelers.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Jewelry, dyeing/weaving, and wood lore now use the same multi-step <code className="text-amber-300">CRAFT &rarr; WORK</code> cycle as weaponsmithing &mdash; <code className="text-amber-300">CRAFT &lt;item&gt;</code> begins planning, then <code className="text-amber-300">WORK &lt;material&gt;</code> each step to complete it</li>
                  <li>Each craft type has its own step flavor: jewelry (shape &rarr; refine &rarr; polish), weaving (mount &rarr; cut &amp; stitch &rarr; hem), wood lore (carve &rarr; sand &rarr; oil)</li>
                  <li>Material is matched by noun name &mdash; if an item matches the noun but isn&rsquo;t a valid crafting material, you receive a &ldquo;not suitable&rdquo; message instead of a confusing error</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">ENCRUST Command (Jeweler 3)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">ENCRUST &lt;item&gt; WITH &lt;gem&gt;</code> &mdash; set a gem stone into an encrustable item</li>
                  <li>Item must have the ENCRUSTABLE flag and at least 2 free adjective slots; gem is consumed</li>
                  <li>Result: gem name and &ldquo;encrusted&rdquo; are applied as adjectives (e.g. a gold ring becomes &ldquo;an emerald encrusted gold ring&rdquo;)</li>
                  <li>Requires a forge or jeweler&rsquo;s workshop; 30-second round time (15 sec under Haste)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">ENGRAVE Command (Jeweler 3)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">ENGRAVE &lt;item&gt; WITH &lt;inscription&gt;</code> &mdash; engrave a custom inscription onto an item</li>
                  <li>Works on any ENCRUSTABLE item or any item made of hard metal</li>
                  <li>Inscription is set verbatim as the item tail &mdash; <code className="text-amber-300">ENGRAVE mug WITH etched with a howling wolf</code> produces &ldquo;a mug etched with a howling wolf&rdquo;</li>
                  <li>Requires a forge or jeweler&rsquo;s workshop; 30-second round time (15 sec under Haste)</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.6.3 &mdash; June 29, 2026</h2>
            <p className="text-gray-400 mb-3">Summoning system overhaul: elemental guard combat, immunities, reagent timing, and creature lifetime management.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Summoned Creature Guard</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">COMMAND GUARD ME</code> now actually intercepts melee attacks &mdash; monsters targeting you hit the elemental instead</li>
                  <li>Ranged attacks (bows, guns, thrown weapons) bypass the guard and land on you normally</li>
                  <li>Guard combat uses the same <code className="text-amber-300">[ToHit: X, Roll: Y] Hit! / Miss!</code> format seen in normal combat</li>
                  <li>Elementals with <code className="text-amber-300">MAGICWEAPON</code> requirements are immune to non-magical attacks &mdash; the TEXI flavor text is shown on hits that do no damage (e.g. &ldquo;The attack serves only to scatter a little water about.&rdquo;), while misses show a normal miss line</li>
                  <li>When the elemental is destroyed while guarding, its TEXD death text is shown and the summoner receives a psychic shock stun (2&ndash;5 seconds)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">COMMAND Improvements</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">COMMAND FOLLOW ME</code> is now a toggle &mdash; issuing it again stops the creature from following (&ldquo;The water elemental stops following you.&rdquo;)</li>
                  <li>All COMMAND actions (FOLLOW, GUARD, ATTACK, BEGONE) now broadcast a confirmation message to the room</li>
                  <li>Summoned creatures follow their summoner through named portals and GO exits, not just cardinal directions</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Reagent Timing</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Elemental summoning reagents (garnet, opal, aquamarine, tourmaline, diamond) are now consumed at PREPARE time, not on CAST</li>
                  <li>Message shown on prepare: &ldquo;The [gem] turns to dust as it is consumed by the spell.&rdquo;</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Creature Lifetime</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Summoned creatures are automatically dismissed if the summoner dies or disconnects</li>
                  <li>Summoned elementals no longer auto-aggro nearby players or wander between rooms</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.6.2 &mdash; June 28, 2026</h2>
            <p className="text-gray-400 mb-3">Physicians&rsquo; Guild healer rooms and full Bank of Fayd banking services.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Healer Rooms</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Rooms tagged as <code className="text-amber-300">HEALER</code> (e.g. Physicians&rsquo; Guild, room 312 in Fayd) now provide medical treatment when you SIT or LAY down</li>
                  <li>Physicians heal all wounds instantly &mdash; charged 1 copper penny per body point restored</li>
                  <li>If you have no money at all you are turned away; if you have some but not enough you are charged everything you have and still fully healed</li>
                  <li>Cure poison: 10 gold crowns &mdash; removes all poison and poison severity</li>
                  <li>Cure disease: 10 gold crowns &mdash; removes all disease and disease severity</li>
                  <li>Poison and disease are treated independently before wounds; each checks your current purse after any prior charge</li>
                  <li>If you are already in perfect health and have no conditions, the physician tells you so</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Banking</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Rooms tagged as <code className="text-amber-300">BANK</code> (e.g. Bank of Fayd, room 355) now offer full banking services</li>
                  <li><code className="text-amber-300">DEPOSIT &lt;amount&gt;</code> &mdash; deposit copper pennies; suffix with GOLD or SILVER to use those denominations (e.g. <code className="text-amber-300">DEPOSIT 5 GOLD</code>)</li>
                  <li><code className="text-amber-300">WITHDRAW &lt;amount&gt;</code> &mdash; withdraw from your account, same denomination suffix supported; coins returned as carried currency</li>
                  <li>No fee to deposit or withdraw money</li>
                  <li><code className="text-amber-300">DEPOSIT &lt;item name&gt;</code> &mdash; place an item into your safety deposit box for 1 gold crown; item matched by name and adjectives (e.g. <code className="text-amber-300">DEPOSIT steel longsword</code>)</li>
                  <li><code className="text-amber-300">WITHDRAW &lt;item name&gt;</code> &mdash; retrieve an item from your safety deposit box, no charge</li>
                  <li>Safety deposit box holds up to 20 items; you must afford the 1 gold fee to deposit an item</li>
                  <li><code className="text-amber-300">BALANCE</code> &mdash; shows your cash balance and lists all items stored in your safety deposit box (X/20 slots used)</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.6.1 &mdash; June 26, 2026</h2>
            <p className="text-gray-400 mb-3">Spell mastery system, 46 missing spells, poison/disease severity levels, herb fixes, Cure Poison and Cure Disease spells, wave emote.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Spell Mastery</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Spell mastery system implemented &mdash; when your school skill reaches double a spell&rsquo;s level, you achieve first mastery rank; each additional rank requires one more spell level of skill</li>
                  <li>Each mastery rank reduces mana cost by 2, floored at half the base cost (minimum 1)</li>
                  <li>SPELLS command now shows a School column and mastery stars: <code className="text-amber-300">(*)</code>, <code className="text-amber-300">(**)</code>, <code className="text-amber-300">(***)</code>, etc.</li>
                  <li>SPELLS list is sorted by school, then level, then spell ID</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Spells</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>46 spells added to the registry across all five schools (Conjuration, Enchantment, Necromancy, General, Druidic) that were present in the original spell list but missing from the engine &mdash; they now appear when learned and are castable</li>
                  <li>Cure Poison (spell 303, Necromancy level 11) &mdash; removes poison and all poison levels from self or a target in the room</li>
                  <li>Cure Disease (spell 319, Necromancy level 12) &mdash; removes disease and all disease levels from self or a target in the room</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Poison &amp; Disease Severity</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Poison and disease now have severity levels (1&ndash;5) &mdash; each level deals 1 BP damage per regen tick</li>
                  <li>Trap severity: minor traps (types 1, 2) = level 1; moderate (types 5, 12) = level 2; major (type 7) = level 3; lethal black needle (type 13) = level 5</li>
                  <li>Monster hits that poison or disease default to level 1 and never downgrade an existing higher level</li>
                  <li>Regen tick message shows damage amount and current BP: <code className="text-amber-300">The poison courses through your veins! [-2 BP, 45/80]</code></li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Herbs &amp; Consumables</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Rowik berry now reliably cures poison and clears poison level, even when the item script has no ITEMADJ3 block</li>
                  <li>Babich root and piece of babich root now correctly heal 5&ndash;15 BP (Body Restoration I)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Emotes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>WAVE with no target now emotes &ldquo;You wave.&rdquo; / &ldquo;[Name] waves.&rdquo; instead of returning &ldquo;Wave what?&rdquo;</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.18 &mdash; June 25, 2026</h2>
            <p className="text-gray-400 mb-3">Vertical passage movement fix, magical item charge detection, timed defense spells target other players.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Movement &amp; Script Engine</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed infinite wait loop in vertical passages (Devil&rsquo;s Chimney, deep crevices) &mdash; <code className="text-amber-300">EQUAL ROUNDTIME</code> in an <code className="text-amber-300">IFPREVERB</code> block without <code className="text-amber-300">CLEARVERB</code> now applies roundtime as a post-move penalty, not a pre-move block</li>
                  <li>Roundtime check now happens before room scripts fire on directional movement, matching original engine behavior</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Magical Items &amp; Detect Magic</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Detect Magic (spell 400) now correctly reads charges from Val2 (was incorrectly checking Val4)</li>
                  <li>Detect Magic now reports remaining charges in its output: e.g. <code className="text-amber-300">glows a soft blue (3 charges remaining)</code></li>
                  <li>Items with a stored spell but no charges remaining now show &ldquo;a faint magical residue &mdash; completely drained of power&rdquo; instead of glowing</li>
                  <li>Magical item treasure drops now correctly store charges in Val2 (was Val4)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Defense Spells</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>All timed defense spells (Globe of Protection, Globe of Protection II, Spectral Shield, and others) can now be cast on yourself or on another player in the room, matching Mystic Armor behavior</li>
                  <li>Casting on another player shows distinct messages to caster, target, and room</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.17 &mdash; June 24, 2026</h2>
            <p className="text-gray-400 mb-3">Flying movement overhaul, script MOVEGROUP fixes, geyser ROOMCOPY, round time display.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Flying &amp; Movement</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>ASCEND and DESCEND now work correctly for players who used the FLY command &mdash; previously only worked for Drakin or players with psionic flight</li>
                  <li>EXIT ABOVE and EXIT BELOW no longer appear in the exits list &mdash; they are fly-only passages, accessible only via ASCEND/DESCEND</li>
                  <li>IFPREVERB ASCEND/DESCEND scripts now fire before movement &mdash; rooms can block or redirect ascent/descent with a message</li>
                  <li>ASCEND/DESCEND abbreviations (e.g. <code className="text-amber-300">asc</code>) now resolve correctly</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Script Engine</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>MOVEGROUP in IFVERB blocks now fires correctly &mdash; was only being checked in IFPREVERB results</li>
                  <li>PLREVENT and CONTPLREVENT (player-timed events) are now implemented &mdash; enables deferred room entry sequences such as river currents pulling players downstream</li>
                  <li>ROOMCOPY script command implemented &mdash; copies exits and description from a template room into the current room; powers the geyser system that periodically opens new exits</li>
                  <li>ECHO OTHGROUP now implemented &mdash; sends a message to room occupants who are not in the player&rsquo;s group</li>
                  <li>EQUAL ROUNDTIME now shows a <code className="text-amber-300">[Round: X sec]</code> message to the player, matching original engine behavior</li>
                  <li>MOVEGROUP in CLIMB scripts now fires correctly and shows round time; success and failure echoes no longer both display after a successful group move</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.16 &mdash; June 23, 2026</h2>
            <p className="text-gray-400 mb-3">Weaponsmith XP scaling, cyclic world events, smelting round time and XP, gold and silver ore.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Weaponsmith XP now scales correctly: 25 XP per skill level required by the weapon, plus a metal quality bonus (up to +150 for truesteel) and a blade sharpness bonus (up to +100)</li>
                  <li>Smelting now applies a 5-second round time (2 seconds if Hasted) before the smith can act again</li>
                  <li>Smelting now awards XP on success: 5 for copper/tin, scaling up to 50 for truesteel and 100 for exotic metals</li>
                  <li>Gold and silver ore can now be found when mining &mdash; grade A mines yield silver (8%) and gold (4%), grade B mines yield silver (5%) and gold (1%); grade C mines do not produce precious metals</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">World Events</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed a parser bug where all 51 cyclic world events (CEVENT) were silently discarded at server startup &mdash; timed events such as Menelian&rsquo;s wandering cottage now fire correctly</li>
                  <li>Menelian&rsquo;s cottage cycles through rooms 128, 156, and 944 every ~12.5 minutes, appearing as a door that players can enter</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.15 &mdash; June 22, 2026</h2>
            <p className="text-gray-400 mb-3">Herb effects, elevator mechanics, script engine AFFECT fixes.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Herbs &amp; Consumables</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Stat-boosting herbs (yarrow lichen, quillim fruit, malatrin leaf, shay-ahm blossoms, zarus stem, coriam seed, kurkan pollen) now correctly limited to one dose each</li>
                  <li>Stat boost range corrected to 1&ndash;4 (was 2&ndash;4)</li>
                  <li>Rowik berry now cures poison</li>
                  <li>Disease-curing herb now implemented</li>
                  <li>Thesnia leaf (mindlink) now refreshes on every bite, not just the first</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Script Engine</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>FLIP switch now correctly fires IFVERB scripts after the state change &mdash; fixes power-on scripts in Technologists Hall</li>
                  <li>Room-level IFTOUCH blocks now execute during IFPREVERB handling &mdash; fixes the Technologists Hall identity scanner</li>
                  <li>IFSAY pattern matching now strips trailing punctuation &mdash; &ldquo;computer, identify&rdquo; matches whether or not the player adds a period</li>
                  <li>PLRSINROOM and MONINROOM now count players/monsters in the AFFECTed room rather than always the player&rsquo;s room &mdash; fixes the Technologists Hall elevator auto-return</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.14 &mdash; June 21, 2026</h2>
            <p className="text-gray-400 mb-3">Group disconnect handling, GUARD command overhaul, improved guard messages, STATUS build point fix.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Group System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>When a group leader disconnects or logs off, the entire group is disbanded and all members are notified</li>
                  <li>When a group member disconnects or logs off, they are removed from the leader&rsquo;s group and the leader is notified</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GUARD Command Overhaul</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>GUARD now requires Combat Maneuvering skill at level 2 or higher to guard a player</li>
                  <li>Combat Maneuvering level 3 unlocks portal guard &mdash; anyone trying to pass a guarded gate or doorway must make a skill check</li>
                  <li>Any Combat Maneuvering level can guard an item on the ground &mdash; others must make a skill check to pick it up</li>
                  <li>You can now guard multiple targets simultaneously &mdash; GUARD &lt;target&gt; toggles each guard on or off independently</li>
                  <li>Cannot guard a player who is already themselves on guard duty</li>
                  <li>Hidden or invisible bypassers are announced as &ldquo;something&rdquo; rather than by name in all guard messages</li>
                  <li>Skill check formula: roll d100 + CM&times;5 + (AGI&minus;50)/10 + (QUI&minus;50)/10 vs 50 + guard&rsquo;s CM&times;5</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Improved Guard Messages</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Skill roll results now shown on both success and failure: <code className="text-amber-300">[Roll: 73 vs 60]</code></li>
                  <li>Successful bypass now shows &ldquo;You slip past X&rsquo;s guard&rdquo; before the pickup or movement confirmation</li>
                  <li>Guard sees their own success/failure with the roll result</li>
                  <li>Room bystanders see appropriate success/failure echoes for portal and item guard attempts</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>STATUS command now shows correct &ldquo;Build Points to date&rdquo; and &ldquo;Unspent Build Points&rdquo; values &mdash; was previously double-subtracting spent points</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.13 &mdash; June 20, 2026</h2>
            <p className="text-gray-400 mb-3">Weapon repair fixes, damaged adjective, oily metal crafting correction.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Weapons damaged in a clash now show a &ldquo;damaged&rdquo; adjective &mdash; a damaged longsword appears as &ldquo;a damaged longsword&rdquo;</li>
                  <li>Successfully repairing a weapon removes the damaged adjective</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Oily and oiled metals no longer produce weapons with icy elemental crit damage &mdash; they were incorrectly inheriting cold-damage properties</li>
                  <li>Weapons and jewelry crafted from oily or oiled metal now show as iridescent instead</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>REPAIR now matches weapons by adjective &mdash; &ldquo;repair damaged longsword&rdquo; correctly finds the item</li>
                  <li>REPAIR no longer says &ldquo;doesn&rsquo;t need repair&rdquo; when the weapon isn&rsquo;t in your inventory &mdash; now says &ldquo;you aren&rsquo;t carrying that&rdquo;</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.12 &mdash; June 19, 2026</h2>
            <p className="text-gray-400 mb-3">Weapon clash metal bonuses, off-hand inventory fix, crafted weapon sharpness system.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Weapon clashes now account for metal type &mdash; harder metals resist damage: steel +25, truesteel +40, elkyri +55, albescent +65, randar +75</li>
                  <li>Soft metals (copper, silver, gold, tin) provide no clash bonus and are more likely to be damaged or broken</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Crafted weapons now receive a non-magical to-hit bonus (Val1) based on the ore&rsquo;s hidden color quality</li>
                  <li>Ore color (purple / indigo / blue) determines the sharpness ceiling &mdash; blue is the best currently mineable</li>
                  <li>Smith&rsquo;s Weaponsmithing skill level shifts the bonus range &mdash; higher skill yields sharper weapons</li>
                  <li>Weapon completion message now reports blade quality and bonus (e.g., &ldquo;The blade is very sharp. (+7 non-magical bonus)&rdquo;)</li>
                  <li>ANALYZE ore at Mining 5+ now also reveals the metal&rsquo;s color quality alongside purity</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Off-hand weapon (Two Weapons skill) now appears at the top of the INVENTORY list alongside wielded and worn items, labeled &ldquo;(off-hand weapon)&rdquo;</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.11 &mdash; April 29, 2026</h2>
            <p className="text-gray-400 mb-3">Crafting pipeline fix, LOOK IN containers, CRAFT recipe list, manuscript protection.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Smelting now preserves ore type &mdash; iron ore smelts to iron metal, not generic metal</li>
                  <li>WORK command matches instance adjectives &mdash; &ldquo;work steel&rdquo; finds steel metal in inventory</li>
                  <li>CRAFT with no args lists all available recipes</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>LOOK IN now works for containers in rooms and inventory (chest, sack, coffer, etc.)</li>
                  <li>Manuscripts and FIXED items can no longer be picked up</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.10 &mdash; April 28, 2026</h2>
            <p className="text-gray-400 mb-3">Players no longer look dead, date/time on login, mining tools from inventory, ambient text reduced.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Critical Fix</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Players no longer appear dead when examined &mdash; fixed divide-by-zero when MaxBodyPoints was 0</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Features</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Date, time, and weather shown when entering the game world</li>
                  <li>Mining tools work from inventory (not just wielded)</li>
                  <li>Mining accepts &ldquo;pick-axe&rdquo; (hyphenated) in addition to &ldquo;pickaxe&rdquo;</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Balance</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Ambient NPC text frequency reduced (15% &rarr; 8%) for less screen clutter</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.9 &mdash; April 28, 2026</h2>
            <p className="text-gray-400 mb-3">Build point fix, level-up mana/psi, open-ended crits, typed mining ore, GM idle exempt.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Critical Fix</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Build Points no longer reset when viewing EXP/STATUS &mdash; spent BP are now properly tracked</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Open-ended critical hits (96-100): bonus damage roll, with double-open-ended for devastating crits</li>
                  <li>Level-up now increases MaxMana and MaxPsi (based on Willpower/Empathy)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Mining now produces typed ore (iron, copper, bronze, tin, steel) based on mine grade</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>GMs exempt from 30-minute idle timeout</li>
                  <li>SET accepts TRUE/FALSE/YES/NO in addition to ON/OFF</li>
                  <li>LOOK IN container shows contents (or &ldquo;empty&rdquo;)</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.8 &mdash; April 28, 2026</h2>
            <p className="text-gray-400 mb-3">Ethereal Projection persistence, Flight cancel, Manipulate Lock, LOOK direction items.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Psionics</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Ethereal Projection now persists through movement (psi-maintained hiding)</li>
                  <li>SEARCH no longer reveals players using Ethereal Projection</li>
                  <li>Canceling Flight/Levitate properly resets position and CanFly</li>
                  <li>Manipulate Lock (PSI 8) now works &mdash; psionically unlocks locked items</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>LOOK direction now shows room items in addition to players and monsters</li>
                  <li>Bot login/logout messages suppressed</li>
                  <li>@vis/@invis now reports if already in that state</li>
                  <li>@set FATIGUE always sets both current and max</li>
                  <li>TEND self says &ldquo;You don&rsquo;t need healing&rdquo; instead of &ldquo;yourself doesn&rsquo;t need healing&rdquo;</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.7 &mdash; April 25, 2026</h2>
            <p className="text-gray-400 mb-3">PSI Flight fix, GM invisibility, Ethereal Projection, emotes on dead monsters.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Psionics</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>PSI Flight (10) now properly sets CanFly &mdash; flying players can go UP/ASCEND</li>
                  <li>Ethereal Projection (PSI 15) is now a maintained buff that makes you hidden</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Invisibility</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Invisible/hidden GMs no longer broadcast login/logout messages</li>
                  <li>Prompt &lsquo;H&rsquo; indicator now shows when GM is invisible (persists across sessions)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Emotes now work on dead monsters (kick that dead rat!)</li>
                  <li>Prompt &lsquo;P&rsquo; indicator now uppercase for Poisoned (per original manual)</li>
                  <li>&lsquo;C&rsquo; indicator used for combat-joined (was conflicting with P for Poisoned)</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.6 &mdash; April 23, 2026</h2>
            <p className="text-gray-400 mb-3">Flight movement, PSI persistence, INITIATE, per-IP limit, NPC pacing, ordinals.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Psionics</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Prepared psionic power now persists after projection &mdash; no need to re-prepare each time</li>
                  <li>Flying players can now move normally (PSI Flight no longer blocks movement)</li>
                  <li>Added Ethereal Projection (PSI 15) discipline</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>INITIATE &lt;player&gt; &lt;org#&gt; &mdash; GM command to set player organization</li>
                  <li>&ldquo;GO COUNTER 2&rdquo; now works (trailing numbers parsed as ordinals)</li>
                  <li>&ldquo;WORK METAL&rdquo; now prompts for specific metal type instead of confusing error</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>@set BP/FAT/MANA/PSI now sets both current and max values</li>
                  <li>Use BODYPOINTS/MAXBODYPOINTS for fine-grained control</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Balance &amp; Polish</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>NPC ambient wander rate halved (less screen clutter)</li>
                  <li>Per-IP connection limit raised from 5 to 8</li>
                  <li>Chrome desktop: additional focus retention fix</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.5 &mdash; April 23, 2026</h2>
            <p className="text-gray-400 mb-3">PSI teleportation, partial item names, @echoplr fix, emote articles, custom descriptions.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Psionics</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>PSI Teleportation (ID 12) now works &mdash; teleports to your marked location</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@echoplr</code> now works with any capitalization of player name</li>
                  <li>DROP/GET now match partial item names (e.g., &ldquo;drop tooth&rdquo; matches &ldquo;rat tooth&rdquo;)</li>
                  <li>Emotes with monsters now include articles (&ldquo;You sniff a lost mutt&rdquo; not &ldquo;You sniff lost mutt&rdquo;)</li>
                  <li>Custom @line descriptions now replace the auto-generated race/gender line in LOOK</li>
                  <li>VER command now shows correct version number</li>
                  <li>Character names must be at least 3 characters (prevents single-letter names)</li>
                  <li>Chrome desktop: improved input focus retention after command output</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.4 &mdash; April 23, 2026</h2>
            <p className="text-gray-400 mb-3">GM edit target, shields, skinning, position checks, CANT, @answer, STOMP.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@edpl &lt;player&gt;</code> now sets an edit target &mdash; subsequent <code className="text-amber-300">@set</code> commands modify that player</li>
                  <li><code className="text-amber-300">@edpl</code> (no args) clears the edit target</li>
                  <li><code className="text-amber-300">@answer</code> &mdash; teleports GM to the last player who used ASSIST</li>
                  <li>@yank offline message clarified</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Shields now have a default worn slot &mdash; WIELD/WEAR shield works correctly</li>
                  <li>Can no longer skin the same corpse multiple times</li>
                  <li>Can no longer SIT/STAND/KNEEL/LAY if already in that position</li>
                  <li>Mark list no longer shows room numbers to non-GM players</li>
                  <li>THUMP and script-interactive verbs now fire IFVERB scripts with emote fallback</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Features</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>CANT (thieves&rsquo; cant) now requires Stealth or Legerdemain skill &mdash; only skilled players can hear it</li>
                  <li>STOMP emote added</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.3 &mdash; April 23, 2026</h2>
            <p className="text-gray-400 mb-3">Stun, immobilize, healing cooldown, multi-attacker defense, NPC flight restrictions.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Stun now works: stunned monsters skip their next attack and are easier to hit (+20 to-hit)</li>
                  <li>Multi-attacker defense penalty: -5 per 2 additional monsters attacking you</li>
                  <li>Polearm fatigue reduced (capped at 3 per swing instead of scaling with weight)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Psionics</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>PSI Immobilize now implemented &mdash; freezes target in place (monsters and players)</li>
                  <li>PSI damage disciplines now give a clear message when targeting players</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Skills</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>TEND/HEAL now has a 5-second round timer and cannot be used in combat</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">World</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>NPCs can no longer wander through ABOVE/UP exits into sky rooms</li>
                  <li>THUMP and similar verbs now trigger room scripts before falling back to emotes</li>
                  <li>Immobilized players cannot move</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">UI</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Mobile keyboard now auto-capitalizes after periods</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.2 &mdash; April 23, 2026</h2>
            <p className="text-gray-400 mb-3">Seasonal rooms, martial arts, healing, whisper, idle timeout, and more fixes.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Seasonal World</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Seasonal room overrides now load at startup and on season change &mdash; fixes missing rooms like Grymwood</li>
                  <li>Rooms defined in spring/summer/autumn/winter scripts now apply their descriptions, exits, and items</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat &amp; Skills</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Martial Arts now boosts unarmed damage: +1 per rank + expanded damage range</li>
                  <li>Monsters lose their target when the player hides (no more attacking hidden players)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Whisper target no longer sees both the whisper content and 3rd-person room message</li>
                  <li>HEAL another player now saves the target&rsquo;s updated health</li>
                  <li>WIELD shield now properly routes to WEAR for any shield or wearable item</li>
                  <li>Fixed &ldquo;You look looks sickly&rdquo; double-verb in self-examine when poisoned/diseased</li>
                  <li>Fixed double space in skin command output (&ldquo;skin a  giant rat&rdquo;)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@setp &lt;player&gt; &lt;var&gt; &lt;value&gt;</code> &mdash; set variables on another player (fixes @set after @edpl)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Server</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>30-minute idle timeout &mdash; inactive players are now disconnected automatically</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5.1 &mdash; April 23, 2026</h2>
            <p className="text-gray-400 mb-3">Second round of player-reported fixes: scripts, groups, combat, GM tools.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Scripts &amp; World</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Scripts can now charge gold &mdash; <code className="text-amber-300">SUB WEALTH</code> / <code className="text-amber-300">SET WEALTH</code> in scripts now works (fixes herb shops, vendors)</li>
                  <li><code className="text-amber-300">MOVEGROUP</code> script command implemented &mdash; moves all players in a room to a destination</li>
                  <li>Monsters no longer spawn in sky/ABOVE rooms</li>
                  <li>Death telepathy &mdash; psionic characters now sense when another player dies</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Group &amp; Follow</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Group members now follow through portals, gates, and doorways</li>
                  <li>Stale &ldquo;J&rdquo; prompt indicator cleared when leader goes offline</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat &amp; Abilities</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>BERSERK stance now available to all races (not just Murg)</li>
                  <li>Psionic Levitate now grants flight (fixes FLY command for non-Drakin psionics)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Dyeing now preserves material adjective &mdash; color goes to second adjective slot instead of overwriting the material name</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@gop</code> no longer reveals invisible GMs (ExitEcho/EntryEcho suppressed when @invis)</li>
                  <li><code className="text-amber-300">@yank</code> now shows the yanked player their new location</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.5 &mdash; April 23, 2026</h2>
            <p className="text-gray-400 mb-3">Feedback pipeline, 16 bug fixes from player reports, email deliverability.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Feedback Pipeline</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>REPORT command now forwards player feedback to VibeCtl for triage and issue tracking</li>
                  <li>Reports include player name, room location, and full message text</li>
                  <li>Backfilled 205 existing reports from game logs</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Features</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">APPRAISE &lt;item&gt;</code> &mdash; see what a merchant will pay before selling</li>
                  <li><code className="text-amber-300">SEARCH</code> (no target) &mdash; scan the area to reveal hidden players (perception vs stealth check)</li>
                  <li><code className="text-amber-300">HEAL &lt;player&gt;</code> &mdash; now routes to TEND for healing instead of showing health stats</li>
                  <li><code className="text-amber-300">TEACH</code> now accepts skill names in addition to skill numbers</li>
                  <li>WHO list is now sorted alphabetically</li>
                  <li>Merchant payments now show proper coin names: gold crowns, silver shillings, copper pennies</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat &amp; Movement</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Wolfling wolf form now uses claw attacks with higher damage (3-10 + STR/10) instead of fist damage</li>
                  <li>Wolf form shows &ldquo;claws&rdquo; instead of &ldquo;fists&rdquo; in combat messages</li>
                  <li>Can no longer climb ladders, go through portals, or use GO while lying down, sitting, or kneeling</li>
                  <li>CONTACT now requires psionic abilities (Psionics or psionic school skill)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Group &amp; Follow</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Following now breaks when you manually move away from your leader</li>
                  <li>Two players can no longer follow each other (circular follow prevented)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Items &amp; Containers</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Shields can now be equipped via WIELD (auto-routes to WEAR)</li>
                  <li>Containers in your inventory can now be opened and closed</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@echoplr</code> now sends the message to the target player (was GM-only echo)</li>
                  <li><code className="text-amber-300">@set ORG &lt;value&gt;</code> &mdash; set a player&rsquo;s organization for script conditionals</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Other Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed invisible player bug &mdash; reconnecting no longer creates duplicate sessions</li>
                  <li>Capture recording button now resets properly on reconnect</li>
                  <li>Fixed input losing focus after pressing Enter on Windows and Android Chrome</li>
                  <li>SPF, DKIM, and DMARC DNS records configured for email deliverability</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.4 &mdash; April 12, 2026</h2>
            <p className="text-gray-400 mb-3">GM Script Editor &mdash; hot-load scripts without server restart.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Script Editor</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>New <strong>GM</strong> nav button (visible when playing a GM character)</li>
                  <li>Upload, edit, and manage <code className="text-amber-300">.scr</code> scripts via a web-based text editor</li>
                  <li>Scripts are parsed, validated, and hot-loaded into the running engine &mdash; no restart needed</li>
                  <li>Scripts stored in MongoDB with priority ordering (higher priority loads first)</li>
                  <li>Version history: last 10 versions preserved with one-click restore</li>
                  <li>Shared GM filespace &mdash; all GMs see and can edit all uploaded scripts</li>
                  <li>Audit trail: who uploaded what, when</li>
                  <li>Upload <code className="text-amber-300">.scr</code> files from disk or type directly in the editor</li>
                  <li>Script size limit: 262 KB</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.3 &mdash; April 11, 2026</h2>
            <p className="text-gray-400 mb-3">Mobile improvements, GM server banner, announcement fixes, weather descriptions.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Mobile UI</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Responsive layout throughout — status bar, nav, admin panel, character creation all adapt to small screens</li>
                  <li>Send button (↵) visible on touch devices</li>
                  <li>Keyboard dismissed after each command so the screen zooms back out on iOS</li>
                  <li>Predictive text / QuickType bar suppressed — no more autocorrect mangling game commands</li>
                  <li>Input font-size set to 16px on mobile, preventing iOS auto-zoom on focus</li>
                  <li>Mobile browser chrome accounted for with <code className="text-amber-300">dvh</code> viewport units</li>
                  <li>Login/logout messages now off by default for new characters</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Server Banner</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@banner &lt;text&gt;</code> — broadcast a notice to all online players and set a login banner</li>
                  <li><code className="text-amber-300">@banner</code> (no args) — clear the banner</li>
                  <li>Banner displays on the web login screen and in the telnet/SSH login menu</li>
                  <li>Stored in-memory with MongoDB backup across restarts</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@announce</code> now broadcasts to all online players (was only echoing to the sender)</li>
                  <li>Weather room descriptions now show prose instead of "The weather is Heavy Snow"</li>
                  <li>Rare Weapons Exchange parlor door now works with GO as well as PUSH</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.2.2 &mdash; April 9, 2026</h2>
            <p className="text-gray-400 mb-3">Game-time calendar, passive regeneration, seasonal world.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Game Calendar &amp; Seasons</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>In-game time now runs at 6:1 ratio (4 real hours = 1 game day)</li>
                  <li>Full calendar: 12 months, 28 days each, with named months</li>
                  <li>Seasons follow the game calendar: Spring, Summer, Autumn, Winter</li>
                  <li>Season changes trigger world-wide broadcasts and hot-swap monster spawns</li>
                  <li>Game time persists to MongoDB &mdash; calendar survives server restarts</li>
                  <li>TIME command shows date, season, and moon phases</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Passive Regeneration</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fatigue, Mana, PSI, and Body Points now regenerate passively over time</li>
                  <li>Regeneration rate based on character stats (Constitution, Willpower, Empathy)</li>
                  <li>Position affects regen speed: laying (3x) &gt; sitting (2x) &gt; kneeling (1.5x) &gt; standing (1x)</li>
                  <li>Ticks every real minute for all online players</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.2.1 &mdash; April 8, 2026</h2>
            <p className="text-gray-400 mb-3">Player manual, SET command, weather, monster spawning fix, combat fixes.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Player Manual</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Full original Player Manual (V3.1, 1994) available as in-game reference</li>
                  <li>Opens as a modal overlay &mdash; read it while playing or creating a character</li>
                  <li>Sticky table of contents with 29 sections</li>
                  <li>Accessible from top navigation bar and footer</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">SET Command</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>SET &mdash; view and toggle all display settings</li>
                  <li>Full/Brief room descriptions, Prompt mode</li>
                  <li>Filter logon, logoff, disconnect messages</li>
                  <li>RPbrief, Battlebrief, Actionbrief, Actbrief filters</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">World</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Dynamic weather system with regional transitions</li>
                  <li>Sunrise and sunset broadcasts to outdoor rooms</li>
                  <li>PSI command: list disciplines, activate by number, toggle maintained powers</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Critical: Monster spawning fixed (MLIST group IDs were not matching rooms)</li>
                  <li>Fixed multi-machine desync (scaled to single machine)</li>
                  <li>Height/weight now set on character creation with race-specific ranges</li>
                  <li>Existing characters backfilled with height/weight on login</li>
                  <li>Attack command now strips articles (&ldquo;attack a skeleton&rdquo; works)</li>
                  <li>Player state saved after monster combat (death, poison persists)</li>
                  <li>Clearer message when attempting PvP (&ldquo;Player combat is not allowed here&rdquo;)</li>
                  <li>Monster room listing condensed to single line</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.2.0 &mdash; April 8, 2026</h2>
            <p className="text-gray-400 mb-3">Authentic gameplay restoration from original 1990s session captures. Major new systems.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Output Authenticity</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>3rd-person combat now shows simplified format (Hit! Awesome damage) matching original</li>
                  <li>Spell casting: Spectacular success (roll 1), Extreme failure (roll 100)</li>
                  <li>Weapon elemental procs show severity + body part damage lines</li>
                  <li>Merchant flavor text: &ldquo;The merchant inspects...&rdquo; for sell, &ldquo;You hand over your money...&rdquo; for buy</li>
                  <li>Search output corrected with round timer</li>
                  <li>Prompt flags: P for combat, J for group, moved before &gt;</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>PRAY &mdash; temple interaction, triggers deity scripts</li>
                  <li>CONTACT &mdash; targeted psionic telepathy</li>
                  <li>GUARD &mdash; protect another player, redirects attacks</li>
                  <li>CHANT &mdash; activate scrolls</li>
                  <li>TEACH &mdash; share skills with other players</li>
                  <li>FILL &mdash; fill glasses from kegs, barrels, fountains</li>
                  <li>DISARM &mdash; disarm traps with Trap &amp; Poison Lore skill</li>
                  <li>SING &mdash; dedicated song verb with message text</li>
                  <li>PLAY &mdash; instrument-specific music when wielding an instrument</li>
                  <li>TURN PAGE &mdash; multi-page book support</li>
                  <li>Whisper to those close &mdash; proximity whisper</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Group System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>FOLLOW/JOIN &mdash; follow another player</li>
                  <li>HOLD &mdash; leader adds member to group</li>
                  <li>DISBAND/LEAVE &mdash; dissolve or leave groups</li>
                  <li>Group movement: followers travel with leader automatically</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting &amp; Weaponsmithing</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Full CRAFT &rarr; WORK forging cycle: heat, hammer, quench, buff, sharpen</li>
                  <li>Material difficulty system: copper (easiest) through exotic metals</li>
                  <li>Enchantment I spell (#202): +10 edge on non-magical weapons</li>
                  <li>REPAIR command for damaged weapons at forges</li>
                  <li>Crafting awards XP based on material difficulty</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Roleplay Features</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Wolfling TRANSFORM: assume wolf form and back</li>
                  <li>Player titles (Lord, Baroness, etc.) shown in LOOK, set via @title</li>
                  <li>TAP staff for light in dark rooms</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Security</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Per-IP rate limiting on all auth endpoints</li>
                  <li>Fixed IP spoofing via X-Forwarded-For (uses Fly-Client-IP)</li>
                  <li>WebSocket origin check hardened to exact match</li>
                  <li>JWT secret validation at startup</li>
                  <li>Case-insensitive character name uniqueness</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.1.0 &mdash; April 8, 2026</h2>
            <p className="text-gray-400 mb-3">MUD client protocol support, mobile fixes, rich prompts.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">MUD Client Protocols</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>GMCP support: Char.Vitals, Char.Status, Char.Stats, Room.Info (powers Mudlet automapper)</li>
                  <li>MCCP2 compression for reduced bandwidth</li>
                  <li>MSSP game metadata for MUD directory listings</li>
                  <li>MSDP variable reporting for TinTin++ compatibility</li>
                  <li>MXP clickable exits and items in supported clients</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Interface</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Rich status prompt in telnet/SSH: color-coded BP, Mana, Psi, Fatigue</li>
                  <li>Simple prompt when GMCP is active (client renders gauges)</li>
                  <li>Fixed character creation screen on mobile/small screens</li>
                  <li>Added Privacy Policy and Terms of Service links</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.0.0 &mdash; April 8, 2026</h2>
            <p className="text-gray-400 mb-3">Telnet &amp; SSH access, email/password authentication, account management.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">MUD Client Access</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Connect via telnet: <span className="text-green-400">lofp.metavert.io</span> port <span className="text-green-400">4000</span></li>
                  <li>Connect via SSH: <span className="text-green-400">ssh -p 4022 lofp.metavert.io</span></li>
                  <li>Works with Mudlet, TinTin++, and any standard MUD client</li>
                  <li>Full ANSI color support, character creation and selection via text menus</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Email/Password Authentication</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Create an account with email and password (in addition to Google login)</li>
                  <li>Link Google login to an existing email/password account (and vice versa)</li>
                  <li>Email verification for new accounts</li>
                  <li>Forgot password / password reset via email</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Account Management</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Account settings modal (click your name in the top-right corner)</li>
                  <li>Change display name, change password, resend verification email</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v10.0.5 &mdash; April 6, 2026</h2>
            <p className="text-gray-400 mb-3">GM tools, combat polish, and multiplayer fixes.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>@zap actually destroys monsters now (was a stub)</li>
                  <li>@trace toggles script execution debug output</li>
                  <li>@verb lists every game command with parameters</li>
                  <li>@go/@goplr broadcast entry/exit echoes (uses custom @entry/@exit text)</li>
                  <li>@invis GMs now move completely silently (no echoes at all)</li>
                  <li>GM flags (@invis, @hide, @gm) persist across reconnections</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat &amp; Movement</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>ADVANCE &lt;target&gt; engages a monster or player in combat</li>
                  <li>RETREAT disengages without fleeing the room</li>
                  <li>YELL is now heard in adjacent rooms through exits</li>
                  <li>DEPART sends to City Gate (safe area) instead of tutorial room</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>USE is now an item interaction verb (not an alias for WIELD)</li>
                  <li>ACT preserves original text case</li>
                  <li>THINK preserves full text including punctuation</li>
                  <li>GIVE no longer shows duplicate messages to recipient</li>
                  <li>QUIT no longer broadcasts departure twice</li>
                  <li>Directional LOOK shows players and monsters in adjacent rooms</li>
                  <li>Fixed double-space in SEARCH output</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v10.0.4 &mdash; April 6, 2026</h2>
            <p className="text-gray-400 mb-3">Multiplayer polish, THINK fix, RECITE poetry, and room presence.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Multiplayer</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Players in a room see when someone logs in (&ldquo;X arrives.&rdquo;) or out (&ldquo;X fades from the Realms.&rdquo;)</li>
                  <li>GIVE items and money no longer shows duplicate messages to recipient</li>
                  <li>QUIT no longer broadcasts &ldquo;left the Realms&rdquo; twice</li>
                  <li>@heal and other GM commands now affect the live session player immediately</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>THINK preserves original text case and punctuation (was dropping first word)</li>
                  <li>RECITE supports backslash (\) for line breaks in poetry and songs</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v10.0.3 &mdash; April 4, 2026</h2>
            <p className="text-gray-400 mb-3">Bot API, money giving, and currency system.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Bot API</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Control characters programmatically via WebSocket API keys</li>
                  <li>Generate API keys from the character menu (shown once, SHA-256 hashed)</li>
                  <li>Bots can&rsquo;t do anything a human player can&rsquo;t &mdash; same rules, same rate limits</li>
                  <li>Bots appear as [Bot] on the WHO list</li>
                  <li>GM bots can be scoped to prevent GM command use</li>
                  <li>Python, Node.js, and TypeScript SDK examples in /bots</li>
                  <li><em>Bots are a new feature &mdash; not part of the original game &mdash; added to help repopulate the world!</em></li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Currency &amp; Trading</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>GIVE money to other players: gold crowns, silver shillings, copper pennies</li>
                  <li>Regional currencies: kragenmark, danir, shard, darktar, dollar</li>
                  <li>Accepts plural forms and full names (e.g., &ldquo;give 5 gold crowns to Taliesin&rdquo;)</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v10.0.2 &mdash; April 4, 2026</h2>
            <p className="text-gray-400 mb-3">Script engine improvements, portal fixes, and quality of life.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Script Engine</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>ELSE branches now work in script conditionals (IFVAR, IFITEM, etc.)</li>
                  <li>Case-insensitive file loading for DOS-era script filenames</li>
                  <li>PORTAL_CLIMBUP and PORTAL_CLIMBDOWN types now recognized by parser</li>
                  <li>New script variables: WARRANT, GFLAG1-4, NUMPLRS, ARENADEATH, position states</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>@line1/2/3 &mdash; set persistent description lines on any character</li>
                  <li>@entry/@exit &mdash; custom room entry and exit messages</li>
                  <li>@speech &mdash; set custom speech patterns (e.g., &ldquo;says grimly&rdquo;, &ldquo;squawks&rdquo;)</li>
                  <li>REPORT command &mdash; players can file reports (broadcast to GMs, logged)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Room 225 stairway (GO STAIR) now works correctly</li>
                  <li>Temple of Amilor (rooms 591-592) doorway and water blessing work</li>
                  <li>AMILOR.SCR now loads correctly on Linux (was lowercase in git)</li>
                  <li>SKILLS command now shows your actual trained skills</li>
                  <li>Fixed grammar in weapon drops and combat messages</li>
                  <li>Natural weapons (claws, teeth, fists) no longer drop as loot</li>
                  <li>Dead players restricted to essential commands (DEPART, LOOK, etc.)</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v10.0.1 &mdash; April 4, 2026</h2>
            <p className="text-gray-400 mb-3">Character management, script variables, admin tools, and bug fixes.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Character Management</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Delete characters from main menu (soft-delete with confirmation modal)</li>
                  <li>Unique first names enforced on character creation</li>
                  <li>Admin: browse deleted characters, recover with optional rename</li>
                  <li>Name validation split: exact match for monsters, substring for slurs only</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Script Engine</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>New variables: WARRANT, GFLAG1-4, NUMPLRS, ARENADEATH</li>
                  <li>Position variables: SITTING, LAYING, STANDING, KNEELING</li>
                  <li>WIELDED, WEALTH, REGION variables added</li>
                  <li>Warrant field on player for law/crime system</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed login blocked by reserved-word check (&ldquo;Pendragon&rdquo; contains &ldquo;dragon&rdquo;)</li>
                  <li>Name validation moved to character creation only, not login</li>
                  <li>Version notes accessible via /version-notes deeplink</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v10.0.0 &mdash; April 4, 2026</h2>
            <p className="text-gray-400 mb-3">Major milestone: complete combat, magic, psionics, crafting, alchemy, and full skill system.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>MINE ore from MINEA/B/C rooms (grade determines purity)</li>
                  <li>SMELT ore into refined metal at FORGE rooms</li>
                  <li>CRAFT weapons/armor at FORGE, clothing at LOOM, wood items at FLETCHER</li>
                  <li>FORAGE terrain-based materials (wood, plants, reagents, dyes)</li>
                  <li>DYE materials at LOOM rooms with natural and crafted dyes</li>
                  <li>ANALYZE ore purity and reagent properties</li>
                  <li>BREW potions via alchemy &mdash; 32 recipes from original game data</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Skill System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>36 skills with build point costs from original documentation</li>
                  <li>Skill prerequisites enforced (magic needs Spellcraft, etc.)</li>
                  <li>Weapon skills: +5 attack per rank &middot; Dodge: +5 defense per rank</li>
                  <li>Martial Arts: +5 attack/+2 defense unarmed, 10+ hits magic monsters</li>
                  <li>Combat Maneuvering: -1s roundtime, 2%/rank dodge special attacks</li>
                  <li>Endurance: +4 BP per rank, 1%/rank elemental damage reduction</li>
                  <li>ANOINT weapons with poison (Trap &amp; Poison Lore)</li>
                  <li>TEND wounds with Healing skill (+50% same-race bonus)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Treasure &amp; Loot</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Treasure table system based on monster TREASURE level</li>
                  <li>Coin drops, weapon/armor drops, spell scrolls, locked chests</li>
                  <li>Magic weapon bonuses and premium materials (elkyri, adamantine)</li>
                  <li>Trapped chests with 13 trap types and spell glyphs</li>
                  <li>Monster weapon drops on death &middot; SEARCH corpses for loot</li>
                  <li>MONEY items auto-convert to currency on pickup</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat Fidelity</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fatigue drain on melee attacks, fatigue ToHit penalties</li>
                  <li>Weapon clash on roll &lt; 3 &mdash; weapons can be damaged or broken</li>
                  <li>Backstab requires puncture weapon (daggers, rapiers)</li>
                  <li>Death = 90% XP penalty toward current build point</li>
                  <li>Spellcraft formula: 25% + EMP/10 + skill*5, fumble on 98+</li>
                  <li>Mana cost = spell level &middot; NOCK/LOAD ranged weapons with ammo</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Monster System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Demand-based spawning: monsters appear when players enter rooms</li>
                  <li>Monsters unload after 3 minutes with no players (ETERNAL exempt)</li>
                  <li>Psi defense auto-activation on spawn (Wall of Force, Psychic Shield, etc.)</li>
                  <li>Hidden/Invisible distinction &mdash; Invisibility spell not broken by movement</li>
                  <li>Corpse decay after 60 seconds &middot; Dead monsters show as &ldquo;(dead)&rdquo;</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v0.97 &mdash; April 4, 2026</h2>
            <p className="text-gray-400 mb-3">Combat, spells, psionics, and GM Manual fidelity &mdash; fight monsters, cast spells, project disciplines.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>ATTACK/KILL &lt;monster&gt; &mdash; original format: [ToHit: X, Roll: Y] Hit!/Miss./Excellent Hit!</li>
                  <li>Damage severity tiers: Puny, Grazing, Minor, Passable, Good, Masterful, Grisly, Severe, Ghastly</li>
                  <li>Attack verbs by weapon type: swings (slash), thrusts (pierce/pole), slashes (claw)</li>
                  <li>Body part targeting: head, body, arms, legs, back, tail</li>
                  <li>Weapon elemental crits (VAL3): 10-50% chance heat/cold/electric bonus damage</li>
                  <li>Racial slayer weapons (VAL3 21-32): bonus damage vs specific monster races</li>
                  <li>Weapon poison (VAL4): delivers poison on hit</li>
                  <li>MAGICWEAPON gating: some monsters require enchanted weapons to hit</li>
                  <li>Monster guard behavior: guards intercept attacks on their charge</li>
                  <li>Cry for law: attacking lawful NPCs alerts nearby guards</li>
                  <li>Monster poison, disease, and fatigue attacks on hit</li>
                  <li>EXTRABODY: monsters have extra HP not counted toward XP</li>
                  <li>Weather combat modifiers: rain/snow/storms reduce attack accuracy</li>
                  <li>Arena rooms prevent lethal damage</li>
                  <li>Alignment shifts on monster kills</li>
                  <li>Hostile monsters (strategy 301+) auto-attack on room entry</li>
                  <li>Monster flee AI based on strategy type and HP percentage</li>
                  <li>Combat stances: OFFENSIVE, DEFENSIVE, BERSERK (Murg), WARY, NORMAL</li>
                  <li>FLEE to escape combat, [Round: X sec] roundtime</li>
                  <li>Death &rarr; Eternity, Inc. &rarr; DEPART to respawn</li>
                  <li>Real XP/build-point table from original GM Manual (100 levels)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Magic System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>PREPARE &lt;spell&gt; then CAST [target] two-step casting</li>
                  <li>60+ spells across 5 schools: Conjuration, Enchantment, Necromancy, General, Druidic</li>
                  <li>Offensive spells: Flame Bolt, Lightning Bolt, Freezing Sphere, Call Meteor, and more</li>
                  <li>Healing: Body Restoration I/II/III, Invigoration, Reconstruction, Regeneration</li>
                  <li>Defense: Mystic Armor (+20), Globe of Protection (+50/+100), Spectral Shield</li>
                  <li>Buffs: Strength I/II/III, Agility I/II/III, Fly, Invisibility, Haste</li>
                  <li>Mana costs, spellcraft skill checks, magic resistance</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Psionic System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>PSI &lt;discipline&gt; then PROJECT [target] two-step projection</li>
                  <li>Mind over Matter: Kinetic Thrust, Pyrokinetics, Cryokinetics, Electrify, Wall of Force, Flight</li>
                  <li>Mind over Mind: Psychic Blast, Psychic Crush, Terror, Pain, Psychic Screen/Shield/Barrier/Fortress</li>
                  <li>Psi point costs, psionic skill checks, psi resistance</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">World Systems</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>SKIN dead monsters for components (weighted random drops from SkinItem definitions)</li>
                  <li>Container traps: 13 types (needles, gas, acid, blades, explosives, glyph spells)</li>
                  <li>Highlander BLEND in mountain/cave terrain</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Race-Specific Emotes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Drakin: flick tongue, bare teeth, spread/fold wings, swish tail</li>
                  <li>Aelfen: rub ears &middot; Highlander: pull beard &middot; Wolfling: bare fangs, chase tail, scent air</li>
                  <li>23 new self-emotes: fume, squint, hum, sneeze, crack knuckles, bat eyelashes, and more</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v0.96 &mdash; April 4, 2026</h2>
            <p className="text-gray-400 mb-3">Living world &mdash; monster spawning, ambient text, wandering, 30+ new emotes, and submit system.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Monster System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Monsters spawn from original script data and appear in room descriptions</li>
                  <li>TEX1-4 random ambient text &mdash; monsters emit flavor text on a timer</li>
                  <li>Non-hostile monsters wander between rooms via exits</li>
                  <li>TEXG/TEXE/TEXM text overrides for spawn, entry, and movement</li>
                  <li>Examine monsters to see their descriptions</li>
                  <li>Target monsters with emotes (point skeleton, kick rat, etc.)</li>
                  <li>GM @spawn and @genmon commands actually create monsters</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Emotes (30+)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>lick, nibble, bark, claw, curse, duck, hiss, hold, hula, jig, moan, massage, and more</li>
                  <li>Self-targeting overrides: spit me, lick me, laugh me, kick me, thump me</li>
                  <li>KISS with body part qualifiers (head, nose, lips, etc.)</li>
                  <li>Submit-gated interactions (kiss lips/navel/feet require target to submit)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Submit System</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>SUBMIT/UNSUBMIT &mdash; accept intimate emotes from other players</li>
                  <li>LICK behavior changes based on submit state</li>
                  <li>Moving to a new room automatically clears submit</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>CEVENT script ECHO messages now delivered to players in the room</li>
                  <li>Monster article handling: &ldquo;an orc&rdquo; vs &ldquo;a skeleton&rdquo;</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v0.95 &mdash; April 3, 2026</h2>
            <p className="text-gray-400 mb-3">LEGENDS.DOC fidelity pass &mdash; lock/unlock, ordinal targeting, Mechanoid emote, and verb aliases.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Lock &amp; Unlock</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>LOCK/UNLOCK commands match KEY items via Val3</li>
                  <li>Proper messages for missing keys, wrong keys, already locked/unlocked</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Ordinal Targeting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>&ldquo;2 gate&rdquo;, &ldquo;other gate&rdquo;, &ldquo;second gate&rdquo; target the Nth matching item</li>
                  <li>Works across all 19 item-matching functions (get, drop, look, open, close, etc.)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Mechanoid Emote</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>EMOTE/UNEMOTE is now a Mechanoid racial ability (toggle emotional state)</li>
                  <li>ACT remains the general-purpose roleplaying command</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Verb Aliases &amp; Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>ORDER as BUY synonym, UNLIGHT/IGNITE/QUAFF/SHOUT/PLACE aliases</li>
                  <li>RECALL with no args runs room-level IFVERB RECALL scripts</li>
                  <li>ACTBRIEF/RPBRIEF toggle commands</li>
                  <li>POUR verb stub</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v0.94 &mdash; April 3, 2026</h2>
            <p className="text-gray-400 mb-3">Deep script engine, named variables, CEVENT system, food mechanics, and original fidelity.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Script Engine</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>226 named global variables synchronized across servers</li>
                  <li>CEVENT cyclic event system &mdash; timed world events every 3 seconds</li>
                  <li>Arithmetic (MUL/DIV/MOD), monster spawning (GENMON/ZAPMON), persistent PVALs</li>
                  <li>Implicit ENDIF handling matches original engine behavior</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Food, Drink &amp; Spells</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Food tracks bites, drinks track sips &mdash; items consumed over multiple uses</li>
                  <li>Mindlink spell (#403) &mdash; eat a thesnia leaf to gain telepathy for one hour</li>
                  <li>THINK command broadcasts to telepathy-enabled players</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Original Fidelity</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Messages match the original: &ldquo;has just entered the Realms&rdquo;, WHO grid format</li>
                  <li>Body Points (not HP), proper articles (&ldquo;an axe&rdquo;)</li>
                  <li>SIT/LAY/KNEEL trigger room scripts, direction abbreviations resolve for IFPREVERB</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v0.93 &mdash; April 3, 2026</h2>
            <p className="text-gray-400 mb-3">Stealth, flight, combat stubs, session capture, and 70+ new verbs.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Stealth &amp; Flight</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>HIDE to conceal yourself, SNEAK to move while hidden</li>
                  <li>FLY/ASCEND/DESCEND/LAND &mdash; Drakin can always fly, others need spells</li>
                  <li>MARK to set teleport anchors for future spell use</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Session Capture</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Record your gameplay sessions from the Capture button</li>
                  <li>View and download previous captures as .txt files</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Admin Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Real-time Event Monitor for script execution, time cycles, and world state</li>
                  <li>Backend health monitoring with /healthz endpoint</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Proper articles: &ldquo;an axe&rdquo; instead of &ldquo;a axe&rdquo;</li>
                  <li>GO command works for non-portal items with scripts (stairways, etc.)</li>
                  <li>Text is now selectable/copyable in the terminal</li>
                  <li>HP renamed to BP (Body Points) throughout</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v0.91 &mdash; April 3, 2026</h2>
            <p className="text-gray-400 mb-3">Major systems expansion &mdash; script engine, world systems, and player features.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Script Engine</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>80+ script variables for conditions (stats, resources, room info, time, weather, flags)</li>
                  <li>IFSAY blocks &mdash; NPCs and objects respond to what you say</li>
                  <li>AFFECT for multi-room script effects</li>
                  <li>Environmental damage, random events, forced positioning</li>
                  <li>Full string substitution: pronouns, item names, newlines, converted numbers</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">World Systems</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>In-game clock and calendar with day/night cycle</li>
                  <li>Monsters spawn in rooms and appear in descriptions</li>
                  <li>Weather system with 15 states shown in outdoor rooms</li>
                  <li>Dark rooms require light sources to see</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Drink, light/extinguish, flip, latch/unlatch</li>
                  <li>Banking (deposit/withdraw in bank rooms)</li>
                  <li>Skill training with 36 named skills</li>
                  <li>150+ spells registered across 5 schools (casting coming soon)</li>
                  <li>Mining, foraging, and crafting commands (stubs for now)</li>
                </ul>
              </div>
            </div>

            <h2 className="text-amber-400 text-lg font-bold mb-1">v0.9 &mdash; April 3, 2026</h2>
            <p className="text-gray-400 mb-3">First public release of Legends of Future Past, resurrected from the original 1990s script files.</p>

            <div className="space-y-4">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Explore the World</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Over 2,200 rooms to explore across the Shattered Realms</li>
                  <li>Nearly 2,000 items and 300 monsters parsed from the original game scripts</li>
                  <li>Move in all compass directions, climb portals, go through gates and doors</li>
                  <li>Look directionally to see what lies ahead before moving</li>
                  <li>Rich room descriptions with original formatted text preserved (poems, maps, ASCII art)</li>
                  <li>Examine items in rooms with scripted descriptions</li>
                  <li>Read signs, plaques, manuscripts, and scrolls</li>
                </ul>
              </div>

              <div>
                <h3 className="text-green-400 font-bold mb-1">Interact with Everything</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Pull, push, turn, rub, tap, touch, search, and dig</li>
                  <li>Buy weapons, armor, and supplies from shops (1,400+ items for sale)</li>
                  <li>Sell items at appropriate merchants</li>
                  <li>Open, close, lock and unlock doors and containers</li>
                  <li>Look inside, on top of, under, and behind objects</li>
                  <li>Recall lore about items based on your knowledge skill</li>
                  <li>Script-driven puzzles and interactions throughout the world</li>
                </ul>
              </div>

              <div>
                <h3 className="text-green-400 font-bold mb-1">Roleplaying & Communication</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>60+ social emotes: smile, bow, kick, hug, dance, and many more</li>
                  <li>Targeted emotes show second-person messages to the target</li>
                  <li>Say, whisper, yell, recite, and custom emote commands</li>
                  <li>See other players in rooms with position descriptions</li>
                  <li>WHO list shows all online players</li>
                </ul>
              </div>

              <div>
                <h3 className="text-green-400 font-bold mb-1">Characters</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>8 races: Human, Aelfen, Highlander, Wolfling, Murg, Drakin, Mechanoid, Ephemeral</li>
                  <li>Stat rolling based on racial ranges</li>
                  <li>Starting gear and newbie guidance at the City Gate</li>
                  <li>Persistent inventory, equipment, skills, and internal variables</li>
                  <li>Multiple characters per account</li>
                </ul>
              </div>

              <div>
                <h3 className="text-green-400 font-bold mb-1">Multiplayer</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Real-time WebSocket gameplay</li>
                  <li>Cross-server coordination &mdash; all players share the same world</li>
                  <li>Automatic reconnection if connection is lost</li>
                  <li>Google sign-in with 30-day session persistence</li>
                </ul>
              </div>

              <div>
                <h3 className="text-yellow-400 font-bold mb-1">Coming Soon</h3>
                <ul className="text-gray-400 space-y-1 ml-4 list-disc">
                  <li>Combat system</li>
                  <li>Magic and psionics (spells, casting, concentration)</li>
                  <li>NPC/monster AI and spawning</li>
                  <li>Crafting, mining, and foraging</li>
                  <li>Tutorial room sequence</li>
                  <li>Seasonal world variations</li>
                  <li>Level progression and experience</li>
                </ul>
              </div>
            </div>
          </section>
        </div>

        <div className="mt-8 pt-4 border-t border-[#333] text-gray-600 text-xs text-center">
          Legends of Future Past &mdash; Originally created in the 1990s, resurrected from original script files.
        </div>
      </div>
    </div>
  )
}
