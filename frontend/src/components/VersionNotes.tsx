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
