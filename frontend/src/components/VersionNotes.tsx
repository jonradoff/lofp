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
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.36.0 &mdash; August 28, 2026</h2>
            <p className="text-gray-400 mb-3">A new invasive spell, Truename, is implemented for the first time. Also two real fixes: a melee critical hit that&rsquo;s supposed to stun or knock down a monster wasn&rsquo;t actually disabling it, and group-following now handles separation and incapacitated followers consistently across every way the group can travel.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Spell: Truename</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Every player now has a hidden, made-up truename &mdash; never their real first/last name &mdash; randomly rolled and guaranteed unique the first time anything actually needs it: casting <code className="text-amber-300">CAST TRUENAME</code> on them, or them summoning/controlling a creature or inscribing a sigil/glyph</li>
                  <li>Cast on a summoned or controlled creature, Truename reveals its summoner&rsquo;s truename; cast on an item you inscribed with a sigil/glyph spell (Imprison Rune, the three elemental Glyphs, Death Scythe), it reveals whoever inscribed it &mdash; neither can put up any resistance</li>
                  <li>Cast on another player it&rsquo;s an aggressive act: their normal mental defenses apply and can make the spell fail, unless they&rsquo;ve <code className="text-amber-300">SUBMIT</code>ted first, which drops those defenses to almost nothing</li>
                  <li>Casting with no target given (or explicitly on <code className="text-amber-300">me</code>/<code className="text-amber-300">self</code>) reveals your own truename, the same default-to-self behavior every other targeted spell already has</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed: Stun/Knockdown From a Critical Hit Not Actually Applying</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>An &ldquo;Excellent Hit&rdquo; that rolls to stun or knock a monster down showed &ldquo;It is stunned!&rdquo; correctly, but wrote the flag onto a disposable copy of the monster instead of the real one tracked by the world &mdash; so the monster was free to attack again on its very next turn as if nothing had happened, sometimes less than a second later</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed: Group Following on Separation and Incapacitated Members</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Walking away from your leader through a door, gate, or other portal now breaks your membership in the group the same way walking through a plain compass exit already did &mdash; previously only directional movement cleared the link, so a portal could leave you still &ldquo;following&rdquo; someone you&rsquo;d actually left behind</li>
                  <li>When a group leader moves, a follower who&rsquo;s in round time, sitting, laying down, or stunned is no longer silently dragged along with the rest of the group &mdash; they&rsquo;re left behind and dropped from the group instead. If that was the leader&rsquo;s last follower, the leader stops being a group leader entirely</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.35.1 &mdash; August 27, 2026</h2>
            <p className="text-gray-400 mb-3">Three bug fixes: repeatable lockpicking on the second+ matching lock in a room, the Teeth of Shartan stream-jump only working once per direction, and several precious gems selling for a single copper instead of their real value.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed: Lockpicking a Second Matching Lock</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">PICK &lt;ordinal&gt; &lt;item&gt; WITH LOCKPICK</code> (e.g. <code className="text-amber-300">pick 2 door</code> or <code className="text-amber-300">pick second door</code>) now actually disambiguates when a room has more than one item with the same name &mdash; it previously ignored the ordinal entirely and reported &ldquo;You don&rsquo;t see anything locked here,&rdquo; matching how <code className="text-amber-300">OPEN</code>/<code className="text-amber-300">CLOSE</code>/<code className="text-amber-300">UNLOCK</code> already number matching items in the room</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed: &ldquo;Jump Boulder&rdquo; Only Working Once Per Direction</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>The stream-crossing boulders at the Teeth of Shartan (all four seasonal variants) let you jump across once each way, then refused with &ldquo;You can&rsquo;t do that&rdquo; on a repeat attempt &mdash; a one-line flag reset in the room&rsquo;s own script was being silently dropped by the script loader instead of running before every jump attempt, so the flag set by a successful jump never cleared. Bare reset lines like this one, sitting outside any <code className="text-amber-300">IF</code> block in a room&rsquo;s script, are now loaded and re-run on every interaction the way the original scripts intended</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed: Several Gems Selling for 1 Copper</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Sapphire, diamond, garnet, sardonyx, agate, coral, and topaz previously sold to shops for a single copper regardless of quality, since the code used to price gems only recognized ones flagged as spell reagents &mdash; and several genuine gem types were never flagged that way in the original data. Gem pricing is now keyed to the same item-number range (99&ndash;121) the Wrecked Ship&rsquo;s offering bowl already uses to recognize a valid gem, so every gem type &mdash; including quartz/crystal stones, which were also underpriced &mdash; now sells for its proper value</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.35.0 &mdash; August 25, 2026</h2>
            <p className="text-gray-400 mb-3">A pass focused on summoned and controlled creatures: buff spells like Strength, Haste, and Globe of Protection can now actually be cast on a familiar, elemental, or dominated undead instead of failing outright, and two real bugs affecting controlled undead are fixed along the way.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Feature: Buff Spells on Summoned &amp; Controlled Creatures</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Strength I/II/III, Agility I/II/III, Haste, Mystic Armor, and the whole Globe of Protection family (Globe of Protection I/II, Mass Protection, Spectral Shield, Ride the Lightning) previously only ever looked for a named player as a target, so casting one on a familiar, elemental, animated skeleton, or dominated undead failed with &ldquo;You don&rsquo;t see &lsquo;X&rsquo; here.&rdquo; Any caster &mdash; not just the pet&rsquo;s own summoner &mdash; can now target it by name</li>
                  <li>These aren&rsquo;t cosmetic: Strength and Agility scale a creature&rsquo;s effective attack and defense rating the same way they scale a player&rsquo;s, Mystic Armor and the Globe of Protection spells add real stacking defense, and Haste actually halves the creature&rsquo;s action-tick interval so a hasted pet acts and attacks twice as often &mdash; mirroring how Haste halves a player&rsquo;s round time</li>
                  <li>Buffing someone else&rsquo;s pet now sends them a heads-up (&ldquo;X casts Strength on your skeleton&rdquo;), the same way <code className="text-amber-300">COMMAND GUARD</code> already notifies a guarded player</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed: Controlled Undead Defending Hostile Monsters Instead of Fighting Them</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Undead dominated with Control Undead I/II kept their species&rsquo; original wild instinct to guard other monster types it&rsquo;s scripted to protect &mdash; a phantom caretaker, for example, guards zombies and zovembies in the wild. Bound to a player, it kept doing that instead of obeying its new master: bringing a controlled caretaker near a hostile zovembie made it &ldquo;guard&rdquo; the zovembie and intercept attacks meant for it, instead of fighting alongside its controller. A summoned or controlled creature no longer acts as an independent guardian of its own kind</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed: Item-Cast Reconstruction Healing the Living</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Reconstruction (337) correctly fizzles when <code className="text-amber-300">CAST</code> on a living target &mdash; it&rsquo;s an undead-only heal &mdash; but an item imprinted with it (like a rub-activated idol) used a separate item-triggered spell path that never had the same check, so the idol healed any player who rubbed it. Body Restoration I/II/III had the matching gap in the other direction: cast from an item on an undead target, it healed them instead of searing their undead flesh as damage. Both now match how the spells behave when cast normally</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.34.0 &mdash; August 14, 2026</h2>
            <p className="text-gray-400 mb-3">A security hardening pass to protect from bots &mdash; account registration is now guarded by a CAPTCHA challenge and a much tighter rate limit. Also, a mistyped or missing spell target no longer wastes the spell.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New: Account Registration Security</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Creating an account now requires solving a CAPTCHA challenge first &mdash; invisible to almost everyone, only escalating to an interactive checkbox or puzzle when the traffic pattern looks automated, so real players notice nothing while scripted mass-account creation gets blocked outright</li>
                  <li>The registration rate limit dropped from 5 attempts/minute to 3/hour per IP address &mdash; the old limit still allowed hundreds of accounts (and hundreds of verification emails) to be created in well under an hour from a single source</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed: Spells No Longer Wasted on a Bad Target</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Casting a spell at a target that doesn&rsquo;t exist &mdash; a typo, or nothing specified at all &mdash; used to permanently consume the spell&rsquo;s mana and clear it from being <code className="text-amber-300">PREPARE</code>d, forcing a full re-prepare even though nothing was ever actually cast. Every spell that resolves a named target (damage spells, buffs, item-enchantment spells, summon/command spells, and more) now leaves the prepared spell and its mana untouched when the target can&rsquo;t be found &mdash; a spell that genuinely gets cast but fails for some other reason (resisted, already in effect, wrong creature type) still costs the attempt as before</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.33.0 &mdash; August 12, 2026</h2>
            <p className="text-gray-400 mb-3">The big one: room containers like a player&rsquo;s home chest can now be marked persistent, so their contents actually survive a server restart instead of resetting every time. Also a new GM registry command, and a wide sweep of fixes to summoned creatures, item commands, and combat/social text.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Feature: Persistent Containers (Home Chests)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Room containers are normally rebuilt from scratch every restart, along with the rest of the world &mdash; anything a player <code className="text-amber-300">PUT</code> in a chest, bin, or other container was always gone by the next reboot. A specific room-item instance can now be flagged durable with <code className="text-amber-300">ITEMBIT19=1</code> on its <code className="text-amber-300">ITEM</code> line (or <code className="text-amber-300">EQUAL ITEMBIT19 1</code> from a script with that item in context), and its contents are written through to the database on every <code className="text-amber-300">PUT</code>/<code className="text-amber-300">GET</code> and restored on boot</li>
                  <li><code className="text-amber-300">@rdata</code> now shows a room item&rsquo;s <code className="text-amber-300">ItemBits</code> and, for containers, their current contents &mdash; neither was visible anywhere before, unlike <code className="text-amber-300">@iexamine</code> for inventory items</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New GM Command: @intnum3</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>INTNUM3 is a per-player identifier scripts lean on constantly &mdash; theft prevention, guildmaster/packleader checks in WOLFHOME.SCR, item ownership stamps &mdash; that GM staff used to track by hand on an external document. <code className="text-amber-300">@intnum3</code> alone lists every assignment, <code className="text-amber-300">@intnum3 &lt;plr&gt;</code> checks one player&rsquo;s value, and <code className="text-amber-300">@intnum3 &lt;plr&gt; &lt;val&gt;</code> assigns one, refusing if another player already has it (the shared GM sentinel value of 1 is exempt)</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed: Summoned Creatures</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">COMMAND GUARD ME</code> was silently cancelling the creature&rsquo;s follow order the moment guard mode turned on, so it got left behind the next time you moved. Guarding is now purely additive &mdash; a creature can guard several people at once &mdash; and never touches who it&rsquo;s following; only <code className="text-amber-300">COMMAND FOLLOW</code> changes that</li>
                  <li>Bend Space I and II teleported the caster (and group) but left any summoned or guarding creature stranded in the room they left, since neither spell went through the normal movement code that brings summons along</li>
                  <li>Bend Space I and II also broke Hidden and Invisibility on cast &mdash; per the original session logs, a hidden or invisible caster stays unseen straight through the teleport</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed: Item Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">PUT</code> could never find a wielded weapon, so any scripted interaction that requires one to be actively held (like a weaponsmith&rsquo;s whetstone) always failed with &ldquo;You aren&rsquo;t carrying that,&rdquo; even right after wielding it</li>
                  <li><code className="text-amber-300">LEARN</code> counted scrolls differently than <code className="text-amber-300">EXAMINE</code> does, so <code className="text-amber-300">learn 12 scroll</code> could target a different scroll than <code className="text-amber-300">exam 12 scroll</code> just showed, or fail outright if anything non-scroll shared the name</li>
                  <li><code className="text-amber-300">@give</code> and <code className="text-amber-300">@take</code> silently deleted the item entirely when the target player wasn&rsquo;t currently online, instead of failing &mdash; both now refuse with a clear &ldquo;not online&rdquo; error</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">WHISTLE</code> always said &ldquo;You whistle innocently,&rdquo; whether you meant it that way or not, and had no way to whistle at someone &mdash; it&rsquo;s now a plain &ldquo;You whistle,&rdquo; with target support (<code className="text-amber-300">whistle mortif</code>)</li>
                  <li><code className="text-amber-300">PULL</code> now supports targeting another player (<code className="text-amber-300">pull mortif</code>) the same way other physical emotes do, respecting <code className="text-amber-300">AVOID</code></li>
                  <li>Invisibility told the caster &ldquo;You fade from sight&rdquo; &mdash; which they obviously can&rsquo;t see happen to themselves &mdash; instead of &ldquo;You feel a tingling sensation,&rdquo; per the original session logs</li>
                  <li><code className="text-amber-300">LOOK</code>/<code className="text-amber-300">EXAMINE</code> on a player listed health and injuries before worn equipment; the original always lists equipment (and any active magical effects) first</li>
                  <li>The <code className="text-amber-300">H&gt;</code> prompt indicator only ever showed for a manual <code className="text-amber-300">HIDE</code>, not for Invisibility or Phantom Form, even though both make you just as hard to see</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.32.0 &mdash; August 11, 2026</h2>
            <p className="text-gray-400 mb-3">Wizard&rsquo;s Armor, Spell Shield, and Cloak Mind were all secretly the same generic physical defense buff instead of what MAGIC.TXT actually describes them as &mdash; fixing Cloak Mind meant building an entirely new monster psionic attack system from scratch, since nothing existed for &ldquo;psi resistance&rdquo; to defend against. Also: a working Charge Wand, five new &ldquo;sigil&rdquo; item-warding trap spells, and two small player-command fixes.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Feature: Monster Psionic Attacks</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Monster psi disciplines (<code className="text-amber-300">DISCIPLINE</code>/<code className="text-amber-300">PSIUSE</code>/<code className="text-amber-300">PSISKILL</code>/<code className="text-amber-300">PSI</code> in the script data) were parsed and stored but only ever consumed for a monster&rsquo;s own passive defense bonus &mdash; no monster had ever actually attacked a player with one. Monsters with an offensive discipline (Kinetic Thrust, Pyrokinetics, Cryokinetics, Electrify, Psychic Blast/Crush/Terror/Pain) now windup and strike with it, mirroring the existing monster-spellcasting system (windup announcement, then a hit-flavor line and damage, dodgeable, and disruptable by taking a hit mid-charge, same as an interrupted spell)</li>
                  <li>Per GMSCRIPT.DOC (&ldquo;powerful psionic creatures should have well over 100 PSISKILL so as to defeat a player&rsquo;s natural resistance to psionics&rdquo;), every player now has a baseline psi-resistance rating even without Cloak Mind &mdash; a weak-to-moderate psionic monster can be shrugged off some of the time; a strong one usually still connects</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixed Spells: Wizard&rsquo;s Armor, Spell Shield, Cloak Mind</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Wizard&rsquo;s Armor (229), Spell Shield (234), and Cloak Mind (235) were all routed through the same generic timed-defense-buff spell handler as Mystic Armor or Globe of Protection, quietly granting flat physical <code className="text-amber-300">+N defense</code> &mdash; none of which matches what MAGIC.TXT documents for any of the three</li>
                  <li>Wizard&rsquo;s Armor now does what it&rsquo;s actually supposed to: &ldquo;no spell disruption&rdquo; &mdash; while active, taking a hit no longer breaks a prepared spell. Casting it surrounds the target in &ldquo;a yellow curtain of light&rdquo; instead of granting any defense bonus</li>
                  <li>Spell Shield now grants &ldquo;+25 magic resistance&rdquo; &mdash; a real, live mechanic: a 25% chance to deflect a spell a monster casts at the target outright, on top of the monster&rsquo;s own cast-chance and the target&rsquo;s dodge. Casting it wraps the target in &ldquo;an antimagical field&rdquo;</li>
                  <li>Cloak Mind now grants &ldquo;+25 psi resistance&rdquo; against the new monster psionic attacks above, and deliberately shows no line at all in a <code className="text-amber-300">LOOK</code>&rsquo;s active-effects list &mdash; the point of this one is that the target doesn&rsquo;t look warded. Casting it on someone else just says &ldquo;X seems changed.&rdquo;</li>
                  <li>All three still stack duration the same way every other timed defense spell does &mdash; 20 minutes per cast, recasting before it expires extends it, capped at 4 hours</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Spell: Charge Wand</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Charge Wand (243, &ldquo;Recharge Wand&rdquo; per MAGIC.TXT) requires mandrake root to prepare, then <code className="text-amber-300">CAST &lt;item&gt;</code> tops up the charges on a wand, rod, or trinket you&rsquo;re holding or wearing that already has a spell imprinted on it &mdash; it can&rsquo;t create a new magic item from scratch. Drains every point of your remaining mana and converts it into charges (remaining mana &divide; the imprinted spell&rsquo;s own mana cost, rounded up), added on top of whatever charges are already there</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Spells: Sigil Traps</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Imprison Rune (227), Thunder Glyph (125), Inferno Glyph (124), Ice Glyph (126), and Death Scythe (322) all share the same mechanic per MAGIC.TXT: &ldquo;cast upon an item, and attuned to the next person who touches them.&rdquo; Cast on an item you&rsquo;re holding, wearing, or one lying on the ground &mdash; including a door or gate. The first person to <code className="text-amber-300">TOUCH</code> it (or try to <code className="text-amber-300">OPEN</code> it while closed or locked) silently claims it and can go on handling it freely; anyone else springs the trap once and it disarms</li>
                  <li>Imprison Rune traps the intruder in a 5-minute force bubble (spell 231). Thunder/Inferno/Ice Glyph hit them with their own damage type at double the roll. Death Scythe conjures a Spectral Sword (345) at 5&times; damage</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">SIT</code> from a laying position said &ldquo;You sit down.&rdquo; like sitting from standing; it now says &ldquo;You sit up.&rdquo;</li>
                  <li><code className="text-amber-300">ACT</code> always used your real name even while in Slime Form, Mist Form, or a Wolfling&rsquo;s wolf form &mdash; <code className="text-amber-300">ACT sloshes around</code> in Slime Form now shows &ldquo;A slime sloshes around&rdquo; instead of your name, matching how every other identity-masking system in the game already works</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.31.0 &mdash; August 10, 2026</h2>
            <p className="text-gray-400 mb-3">Two new spells &mdash; a stealthy Paranoia curse and a wild-card Dispel Lesser Magic &mdash; a spellcraft rebalance to help new casters actually land their first spells, and fixes to potion drinking, herb-based spell effects, and a room-routing bug that sent departure messages to the wrong room when climbing or slipping through a portal.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Spells: Paranoia and Dispel Lesser Magic</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Paranoia (226) settles a 20-minute creeping unease on its target: each real minute there&rsquo;s a 50% chance of one of thirteen unsettling echoes (&ldquo;Something taps you on the shoulder.&rdquo;, &ldquo;You hear a scream behind you!&rdquo;, and eleven others). Unlike every other spell, casting it at someone else produces no message at all to the caster, the target, or the room &mdash; the target has no way of knowing they&rsquo;ve been cursed &mdash; and it doesn&rsquo;t reveal a Hidden or Invisible caster the way casting normally does</li>
                  <li>Dispel Lesser Magic (401) strips one random active timed magical effect from its target &mdash; a buff, debuff, entangle, or defense spell &mdash; regardless of how much duration it had left. &ldquo;A deep red light twinkles around&rdquo; the target is visible to everyone in the room, but which effect actually faded, if any, is only reported privately to the target</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Balance: Easier Early Spellcasting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>The base spellcraft success chance for casting any spell rose from 25% to 50% (still Empathy/10 + spellcraft skill&times;5% on top, capped at 95%) &mdash; a fresh level-1 caster with one rank of Spellcraft was landing spells far less often than a level-1 weapon user landed hits, since casters needed a favorable roll just to get the spell off at all, on top of a second resist roll most offensive spells still require against the target</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Drinking a bound potion showed backwards, over-verbose messages (&ldquo;You drink the potion from a reeking bottle engraved with the writing, &lsquo;Night Vision&rsquo;.&rdquo;) &mdash; now reads &ldquo;You drink the reeking potion from a bottle.&rdquo;, naming the potion itself rather than misapplying its color to the container, and no longer spoils which spell is bound to it</li>
                  <li><code className="text-amber-300">EAT</code> on a spell-bearing herb or mushroom only worked for a couple of hardcoded cases (Cure Poison, Body Restoration I) &mdash; anything else printed &ldquo;[Spell #318 effect coming soon.]&rdquo; even though the spell was fully implemented, e.g. a riyong mushroom&rsquo;s Body Restoration III. <code className="text-amber-300">EAT</code> now falls back to the same general spell-effect handling <code className="text-amber-300">DRINK</code> already used for potions</li>
                  <li><code className="text-amber-300">CLIMB</code> (e.g. the ladder in room 615) and slipping through a scripted portal via <code className="text-amber-300">STEAL</code> were sending the &ldquo;X leaves.&rdquo; departure message to the destination room instead of the room being left, because the room being left was read after the room-changing script had already moved the player. Both now capture the departing room first, matching how ordinary <code className="text-amber-300">GO</code> movement already handled it</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.30.0 &mdash; August 7, 2026</h2>
            <p className="text-gray-400 mb-3">Guild rank now advances automatically as you train, matching the original game&rsquo;s formula, and Menelian&rsquo;s traveling cottage in Grymwood no longer leaves a stray duplicate of itself behind as it moves.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Feature: Automatic Guild Rank Advancement</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Training a skill at your guild&rsquo;s own trainer now raises your rank in that guild by the build points spent &mdash; reconstructed from the original documented formula (&ldquo;ORGRANK = build points spent on training within guild&rdquo;), so rank reflects how much you&rsquo;ve actually trained there since joining, not your overall character level or experience</li>
                  <li>Automatic advancement caps at rank 199 (just below High Master/High Priest) &mdash; crossing into the 200+ tier still requires a GM&rsquo;s <code className="text-amber-300">@rank</code></li>
                  <li>Training now prints &ldquo;You are now rank X in the guild.&rdquo; whenever your displayed rank ticks up, matching the original flavor text</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Menelian&rsquo;s Cottage (Grymwood, room 128) periodically appeared with an extra duplicate of itself and a phantom door, and could leave the cottage stranded behind with its door missing after moving on &mdash; caused by the cottage&rsquo;s placement script reusing an item-ref slot the room already used for its own ground-snow decoration. Placing an item at a ref that&rsquo;s already occupied now replaces what was there instead of stacking a duplicate, which also protects any other script that reuses a room&rsquo;s item refs the same way</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.29.0 &mdash; August 6, 2026</h2>
            <p className="text-gray-400 mb-3">A fully working Scry spell &mdash; remote room visions and a two-step eye-of-scrying ritual &mdash; a brand new guild area, the Order of the Skull, and a wide sweep of commands that were leaking a disguised player&rsquo;s real name instead of their persona.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Spell: Scry</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Scry (215) has two uses. <code className="text-amber-300">CAST &lt;mark#&gt;</code> shows a brief vision of a marked room &mdash; full occupants, items, and exits, respecting <code className="text-amber-300">BRIEF</code> &mdash; without moving you there or revealing who&rsquo;s watching. Anyone actually standing in the scried room gets &ldquo;You have a brief yet distinct feeling that you are being watched.&rdquo;</li>
                  <li>The second use targets a carried eye (termite eye, sharkhor eye, newt eye, werewolf eye, etc.): prepare and cast Rite of Preparation (412, a newly added spell &mdash; previously mislabeled &ldquo;Bloodsight&rdquo; in the spell list and otherwise unimplemented) at the eye to turn it cloudy, then Scry at the cloudy eye to make it translucent. <code className="text-amber-300">LOOK IN</code> a translucent eye reveals the room where the last player death occurred, with a 1-in-10 chance per look that the eye crumbles to dust and is destroyed</li>
                  <li>Fixed a message-ordering bug shared with Bend Space (213/222): the success-roll line was being inserted before &ldquo;You gesture into the air.&rdquo; instead of after it</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Guild: Order of the Skull</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>A new necromancy-focused organization (28) with a six-room tower reachable via a climbable path from the cemetery trail near Fayd. Entry to the tower is restricted to Order members; inside, a Great Hall connects to a reagent shop (Anatomicatory), a skill-training room (Chamber of Necrology), and a hidden passage behind the throne &mdash; concentrate, pull, tap, or push the onyx skull revealed by looking behind the throne &mdash; leading up to the Skull Lord&rsquo;s private chamber</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Disguise Command</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">DISGUISE</code> with no field value now always lists the basic personas (trader, merchant, lawkeeper, priest, commoner, beggar) available regardless of skill rank, instead of only mentioning them below rank 10</li>
                  <li><code className="text-amber-300">DISGUISE LIST &lt;#&gt;</code> now shows Hairstyle and Haircolor as separate lines instead of blending them into one &ldquo;Hair&rdquo; entry</li>
                  <li>New <code className="text-amber-300">DISGUISE CLEAR &lt;#&gt;</code> command resets a save slot back to empty</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes: Disguise Identity Leaks</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>A disguised player&rsquo;s real name was leaking through instead of their persona in a large number of places: tending wounds (self, another player, or a monster/corpse); wielding or unwielding a weapon or shield, wearing or removing an item; eating, drinking, picking up, and dropping items (including coins and <code className="text-amber-300">GET ALL</code> / <code className="text-amber-300">GET ALL FROM</code> a container); following, holding, leaving, and disbanding a group, including the equivalent messages when a group leader or member disconnects; picking a lock; opening, closing, locking, and unlocking a door, gate, or chest; giving an item or money to another player; every step of crafting (mining, smelting/forging, weaving, dyeing, foraging, alchemy, jewelcraft, engraving); buying from a shop; dumping a container; and logging out (&ldquo;X fades from the Realms&rdquo;)</li>
                  <li>Leveling up no longer announces a disguised player&rsquo;s advancement to the room at all, rather than exposing their real name</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.28.0 &mdash; August 5, 2026</h2>
            <p className="text-gray-400 mb-3">Nine new Enchantment spells &mdash; temporary creature domination and a full suite of concealment/transformation magic &mdash; plus a rebuilt <code className="text-amber-300">HEALTH</code> command, a new <code className="text-amber-300">FATIGUE</code> command, an optional reagent bonus reconstructed from a 1990s session log, and a handful of smaller fixes.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Spells: Temporary Domination</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Command (205), Domination I (206), and Domination II (214) let a caster temporarily seize control of a living (non-undead) creature &mdash; the same <code className="text-amber-300">COMMAND</code>-verb control (<code className="text-amber-300">FOLLOW</code>, <code className="text-amber-300">STAY</code>, <code className="text-amber-300">GUARD</code>, <code className="text-amber-300">ATTACK</code>, <code className="text-amber-300">LOOK</code>, <code className="text-amber-300">SAY</code>, <code className="text-amber-300">BEGONE</code>) already granted by a summoned elemental, but only for the duration, and only against a creature at or below a body-point ceiling based on its <em>current</em> body points, not its max: Command 50 BP for 2 minutes, Domination I 100 BP for 20 minutes, Domination II 200 BP for 20 minutes. Recasting on the same creature resets the duration instead of stacking it</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Spells: Concealment &amp; Transformation</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Invisibility (225), Mass Invisibility (212), and Phantom Form (248) have no duration &mdash; the target stays concealed until they smile, cast a spell (preparing doesn&rsquo;t count, only the actual cast), or attack. All three can be cast on yourself or a named player in the room</li>
                  <li>See Hidden (405) reveals every hidden, invisible, or phantom-formed player in the room: a hidden player shows as &ldquo;You see something.&rdquo;, an invisible one shows their real effective name (accounting for wolf form, disguise, mist, or slime), and Phantom Form always shows as &ldquo;You see a shimmering grey form.&rdquo; no matter who it actually is</li>
                  <li>Mist Form (232) and Slime Form (245) &mdash; self-only transforms lasting until <code className="text-amber-300">TRANSFORM</code> reverts them (15-second roundtime, &ldquo;You congeal back into your body.&rdquo;). Both block attacking, casting, wearing/removing items, all speech, and every emote (even a smile), and lock your inventory entirely (<code className="text-amber-300">INVENTORY</code>, <code className="text-amber-300">GET</code>, <code className="text-amber-300">DROP</code>, <code className="text-amber-300">PUT</code>, <code className="text-amber-300">WIELD</code>, <code className="text-amber-300">UNWIELD</code>, <code className="text-amber-300">GIVE</code>) so there&rsquo;s no way to fall back on a magic item instead. Mist Form grants full immunity to physical and magical damage &mdash; including bleeding, poison, and disease ticks &mdash; plus flight/ascend/descend and passing through closed doors and gates unless they&rsquo;re flagged <code className="text-amber-300">SEALED</code>; Slime Form only reduces damage taken to 10% (also skipping DOT ticks entirely), with none of the mobility perks. Successfully casting either immediately breaks Plant Snare and Imprison. Both appear in room listings and <code className="text-amber-300">EXAMINE</code> as &ldquo;some mist&rdquo; / &ldquo;a slime&rdquo; instead of your real name</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">HEALTH</code> rewritten to match the original session-capture format (&ldquo;You have 256/276 body points.&rdquo;) and now shows your current bleeding/poison/disease drain rate per minute when any are active, in place of the old vague &ldquo;You are moderately wounded.&rdquo; descriptor</li>
                  <li>New <code className="text-amber-300">FATIGUE</code> command (abbreviates to <code className="text-amber-300">FAT</code>) &mdash; shows current/max Mana, Psi, and Fatigue, split out of <code className="text-amber-300">HEALTH</code> into its own command</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Items &amp; Reagents</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>A moonstone can now be used as an optional <code className="text-amber-300">PREPARE</code> reagent for any spell that doesn&rsquo;t already require one of its own &mdash; <code className="text-amber-300">PREPARE &lt;spell&gt; WITH &lt;moonstone&gt;</code> consumes it for a +25% bonus to that cast&rsquo;s success chance, reconstructed from a 1990s player session log where a moonstone was used exactly this way alongside Freezing Sphere</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>A room&rsquo;s bare-verb <code className="text-amber-300">SMELL</code>/<code className="text-amber-300">LISTEN</code>/<code className="text-amber-300">SNIFF</code> script (no item targeted) was silently dropping everything after a <code className="text-amber-300">PLREVENT</code>/<code className="text-amber-300">CONTPLREVENT</code> pause &mdash; e.g. the wolfling cave&rsquo;s cleansing-smoke effect on <code className="text-amber-300">SMELL</code> never actually applied its second half</li>
                  <li><code className="text-amber-300">OPEN</code> on a container with something inside now says &ldquo;You open the chest, revealing a rusty key and 12 gold.&rdquo; instead of stopping at &ldquo;You open the chest.&rdquo;</li>
                  <li>Call Storm (501) now has its own flavor text &mdash; &ldquo;Energy crackles between &lt;caster&gt;&rsquo;s fingertips and then lances skyward.&rdquo; &mdash; instead of a generic &ldquo;gestures and casts&rdquo; line</li>
                  <li>New kill-flavor line for heat/fire deaths: &ldquo;Internal organs stew in their own juices. Throw another on the barbe&rsquo;, mate.&rdquo;</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.27.1 &mdash; August 4, 2026</h2>
            <p className="text-gray-400 mb-3">Three long-dormant Enchantment spells brought to life &mdash; Silence, Imprison, and Identify previously did nothing when cast &mdash; plus a couple of message-ordering fixes.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Enchantment Spells</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Silence (219) now silences its target for a full minute &mdash; unable to <code className="text-amber-300">SAY</code>/<code className="text-amber-300">'</code>/<code className="text-amber-300">YELL</code>/<code className="text-amber-300">SING</code>/<code className="text-amber-300">RECITE</code>, and since casting requires speech, unable to cast any spell either. Recasting resets the duration back to a full minute rather than stacking. A monster flagged <code className="text-amber-300">SILENCEIGNORE</code> (one that casts via hand movements or symbols instead of speech, per GMSCRIPT.DOC) keeps casting right through it</li>
                  <li>Imprison (231) now traps its target in a blue force bubble for 5 minutes &mdash; they can&rsquo;t attack anyone or cast any spell, including on themselves, so they can&rsquo;t even attempt to dispel it. The bubble works both ways: nobody else can land a physical or spell attack on the trapped target either. An imprisoned monster now shows &ldquo;(imprisoned)&rdquo; in room listings</li>
                  <li>Identify (228) now names the exact spell bound to an item and how many charges remain, the precise counterpart to Detect Magic&rsquo;s vaguer &ldquo;glows a soft blue&rdquo; hint</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed a handful of spells whose cast messages named the target (Call Meteor, Chain Lightning, Siryx&rsquo;s Terrible Tentacles) showing the success roll before the opening gesture line instead of after it, unlike every other targeted spell</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.27.0 &mdash; August 3, 2026</h2>
            <p className="text-gray-400 mb-3">A full Disguise skill &mdash; compose and wear a persona that blends seamlessly into the world&rsquo;s common NPCs or a custom identity of your own, with matching movement, speech, and appearance across every system that shows a player&rsquo;s identity &mdash; a merged room-look format matching the original game&rsquo;s style, and a sweep of related bugs including a Rakes guild password that echoed but never actually let anyone in.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Feature: Disguise Skill</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">DISGUISE &lt;slot&gt; &lt;field&gt; &lt;value&gt;</code> composes a saved persona &mdash; each Disguise rank unlocks a new field to change (1 name, 2 gender, 3 hair color/style, 4 skin/eye color, 5 age, 6 apparent strength, 7 height, 8 weight, 9 race, 10 a custom name of your own instead of a generic one), and grants a save slot every 2 ranks, up to 5</li>
                  <li><code className="text-amber-300">DISGUISE APPLY &lt;slot&gt;</code> puts one on (<code className="text-amber-300">[Round: 30 sec]</code>), <code className="text-amber-300">DISGUISE REMOVE</code> takes it off (<code className="text-amber-300">[Round: 15 sec]</code>), and <code className="text-amber-300">DISGUISE LIST</code> / <code className="text-amber-300">DISGUISE LIST &lt;slot&gt;</code> review what you&rsquo;ve composed. Bare <code className="text-amber-300">DISGUISE</code> shows instructions tailored to your current rank</li>
                  <li>Below rank 10, your name is restricted to one of the game&rsquo;s generic town NPC types &mdash; commoner, trader, merchant, lawkeeper, beggar &mdash; and disguised as one of those, you&rsquo;re indistinguishable from the real thing: your name gets the same article (&ldquo;a commoner&rdquo;), you wander and arrive the way those NPCs do (&ldquo;A commoner wanders south.&rdquo; / &ldquo;has arrived.&rdquo; instead of &ldquo;goes/arrives&rdquo;), and <code className="text-amber-300">EXAMINE</code> skips straight to the description with no &ldquo;You look at X.&rdquo; opener, exactly like looking at a genuine NPC of that type</li>
                  <li>At rank 10+, a custom persona can have a full first and last name; room lists and broadcasts still show only the first name (matching how real players are always shown), with the full name reserved for <code className="text-amber-300">EXAMINE</code></li>
                  <li>Your disguised identity now carries through everywhere another player would see or address you &mdash; <code className="text-amber-300">LOOK</code>, <code className="text-amber-300">WHISPER</code>, <code className="text-amber-300">CONTACT</code>, a summoned creature&rsquo;s <code className="text-amber-300">COMMAND FOLLOW/GUARD/ATTACK</code>, and the broadcast text for combat, movement, spellcasting, and emotes all resolve and display your disguise instead of your real name while one is worn</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Room Look Overhaul</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Players and monsters/NPCs are now listed together in one sentence (&ldquo;You see Rion, a lawkeeper and a commoner.&rdquo;) instead of two separate ones, and players are shown by first name only &mdash; matching the original 1990s format &mdash; instead of full name plus race</li>
                  <li>Items now get their own sentence (&ldquo;A table is here.&rdquo;, &ldquo;A glass and some wine are here.&rdquo;), matching the original session-capture wording, instead of being folded into a &ldquo;You see&hellip;&rdquo; sentence</li>
                  <li>The five generic town NPC types (commoner, trader, merchant, lawkeeper, beggar) now get a full randomly-rolled appearance on <code className="text-amber-300">LOOK</code> &mdash; race, gender, age, build, eyes, skin, hair &mdash; generated once per spawn and cached so looking twice shows the same person, instead of a flat &ldquo;You see a trader.&rdquo;</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>A script using <code className="text-amber-300">MOVEGROUP</code> from an <code className="text-amber-300">IFSAY</code> block &mdash; like the Rakes guild hall&rsquo;s &ldquo;admit&rdquo; password at Geoffrey the wine cellar doorman &mdash; played out its flavor text but never actually relocated anyone, since <code className="text-amber-300">IFSAY</code> scripts only ever handled the single-player <code className="text-amber-300">MOVE</code> action, not <code className="text-amber-300">MOVEGROUP</code></li>
                  <li>Items placed inside a container via <code className="text-amber-300">PUT</code>/<code className="text-amber-300">NEWPUT</code> (e.g. wine poured into a glass) were showing up as loose items lying on the floor in addition to being in the container &mdash; room <code className="text-amber-300">LOOK</code> now correctly hides anything marked as being inside another item</li>
                  <li>An <code className="text-amber-300">@invis</code> GM moving through a portal (door, arch, etc.) leaked their real name in the &ldquo;X goes through Y.&rdquo;/&ldquo;X arrives.&rdquo; messages, even though speech already correctly anonymized them as &ldquo;Something says&hellip;&rdquo; &mdash; portal movement is now silent for concealed players, matching ordinary directional movement</li>
                  <li><code className="text-amber-300">@editem</code> couldn&rsquo;t set any multi-word field except <code className="text-amber-300">tail</code> &mdash; added <code className="text-amber-300">examinedesc</code> (sets an item archetype&rsquo;s <code className="text-amber-300">EXAMINE</code> text) and generalized the multi-word-value parsing instead of hardcoding it to one field</li>
                  <li><code className="text-amber-300">EXAMINE</code> on an item with a custom examine description showed a redundant &ldquo;You look at your X.&rdquo; line before the description; it now prints just the description, matching the original game</li>
                  <li><code className="text-amber-300">APPEARANCE</code> with no argument now clears your custom appearance line instead of just redisplaying it &mdash; there was previously no other way to reset it</li>
                  <li><code className="text-amber-300">@speech &lt;name&gt;</code> with no verb phrase now resets that player&rsquo;s speech pattern to default, instead of failing with a usage error</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.26.0 &mdash; August 2, 2026</h2>
            <p className="text-gray-400 mb-3">A new Highlander gem-molding ability, casting a spell now actually breaks stealth and invisibility the way preparing one doesn&rsquo;t, monsters that flee combat properly release everyone who was fighting them, and a wide sweep of command-abbreviation and script-hijack bugs uncovered while chasing down a single quirky item.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Highlander Ability</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">MOLD &lt;gem&gt;</code> &mdash; Highlanders can now work a chipped or cracked gem into a polished one (e.g. <code className="text-amber-300">MOLD chipped diamond</code> or <code className="text-amber-300">MOLD 3 diamond</code>), raising its resale value. There&rsquo;s always some risk of botching it: a fresh Highlander has a 20% chance to ruin the gem instead (leaving it permanently damaged and un-moldable), dropping 1% per level to a permanent floor of 1%. A successful mold raises the gem&rsquo;s value by 25% at level 1, up to a 5%-per-level bonus capped at a 100% increase. A gem already polished, or already ruined from a prior botch, can&rsquo;t be molded again</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Spellcasting &amp; Stealth</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Casting a spell (not preparing one) now reveals a hidden or invisible caster &mdash; the spoken words and hand gestures give away your position the moment you actually cast, even on a failed or fumbled attempt. <code className="text-amber-300">PREPARE</code> stays silent by comparison: onlookers just see &ldquo;Something prepares a spell.&rdquo; while you remain concealed right up until the cast itself</li>
                  <li>Invisibility (225) now actually breaks when you cast any other spell, on top of already breaking when you attack</li>
                  <li>An invisible player who <code className="text-amber-300">SMILE</code>s now fades back into view &mdash; the expression gives away your position even when the rest of you shouldn&rsquo;t be seen. Psionic powers still don&rsquo;t reveal you, since they&rsquo;re worked by thought alone</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>When a monster flees combat, everyone who was fighting it is now automatically taken out of combat, the same way it already worked on a kill &mdash; previously only the monster&rsquo;s own target reference was cleared, leaving players stuck thinking they were still fighting a monster that had already left the room</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Command &amp; Script Engine Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed a latent bug in command-abbreviation resolution: nine verbs (<code className="text-amber-300">TOUCH</code>, <code className="text-amber-300">THINK</code>, <code className="text-amber-300">TAP</code>, <code className="text-amber-300">RUB</code>, <code className="text-amber-300">RECITE</code>, <code className="text-amber-300">PROMPT</code>, <code className="text-amber-300">FULL</code>, <code className="text-amber-300">DEPART</code>, <code className="text-amber-300">BRIEF</code>) were accidentally listed twice in the verb registry, which made the ambiguity checker think a verb collided with itself and refuse to resolve any abbreviation of it at all</li>
                  <li><code className="text-amber-300">APPRAISE</code> was missing from the verb registry entirely, so no abbreviation of it could ever resolve (only the fully-typed word worked, by coincidence)</li>
                  <li>Several genuinely ambiguous short abbreviations now resolve to their more commonly used meaning instead of failing outright: <code className="text-amber-300">mas&rarr;MASSAGE</code>, <code className="text-amber-300">pro&rarr;PROJECT</code>, <code className="text-amber-300">sel&rarr;SELL</code>, <code className="text-amber-300">fli&rarr;FLIP</code>, <code className="text-amber-300">poi&rarr;POINT</code>, <code className="text-amber-300">sla&rarr;SLAP</code>, <code className="text-amber-300">hea&rarr;HEADSHAKE</code>, <code className="text-amber-300">lea&rarr;LEAN</code>, <code className="text-amber-300">sni&rarr;SNIFF</code>, <code className="text-amber-300">cur&rarr;CURSE</code>, and <code className="text-amber-300">dro&rarr;DROP</code> &mdash; each runner-up verb (e.g. MASTER, POISON, SLAY, HEALTH) stays reachable with one more letter or the full word</li>
                  <li><code className="text-amber-300">WEAR</code>, <code className="text-amber-300">OPEN</code>, <code className="text-amber-300">CLOSE</code>, <code className="text-amber-300">LATCH</code>, <code className="text-amber-300">SELL</code>, <code className="text-amber-300">APPRAISE</code>, and <code className="text-amber-300">PROJECT</code> all shared the same bug: each rejected a target as mechanically ineligible (not wearable, not a container, no shop nearby, no psi discipline prepared, etc.) before ever giving that item&rsquo;s own scripted reaction a chance to run &mdash; so an item designed to hijack one of these verbs unconditionally (regardless of its own type or the caster&rsquo;s state) could never actually do so. All seven now check for that scripted reaction first</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.25.0 &mdash; July 31, 2026</h2>
            <p className="text-gray-400 mb-3">Wolfling wolf-form transformation is now a real disguise &mdash; a generated animal description, growled/snarled speech, wolf-only emotes, and an anonymized &ldquo;a wolf&rdquo; everywhere a transformed wolf acts &mdash; plus prepared spells can now actually be interrupted by taking a hit (with Wizard&rsquo;s Armor as the ward against it), and Nyraine&rsquo;s long-isolated coastal farmlands are finally reachable.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Wolf Form (Wolfling)</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">LOOK</code> at a transformed Wolfling now generates a real wolf description (age, gender, eye color, and fur color mirroring hair color, or black if bald in human form), followed by health and any wounds &mdash; described with paw/foreleg/hind-leg wording &mdash; instead of showing their real name, race, build, or any GM-set <code className="text-amber-300">@line1-3</code> text. Worn gear and a custom <code className="text-amber-300">APPEARANCE</code> line are hidden too, all reverting to the normal human description (custom or generated) the moment they transform back</li>
                  <li>Speech becomes growls (<code className="text-amber-300">SAY</code>) and snarls (<code className="text-amber-300">EXCLAIM</code>) while transformed &mdash; asking a question still reads as &ldquo;asks&rdquo;. A custom speech adverb is ignored while in wolf form</li>
                  <li>A transformed wolf&rsquo;s identity is now hidden as &ldquo;a wolf&rdquo; everywhere they act, not just when looked at or spoken to &mdash; <code className="text-amber-300">SAY</code>/<code className="text-amber-300">WHISPER</code>/<code className="text-amber-300">YELL</code>, every emote, combat round messages, arriving/leaving a room (including logging in already transformed), <code className="text-amber-300">SIT</code>/<code className="text-amber-300">STAND</code>/<code className="text-amber-300">KNEEL</code>/<code className="text-amber-300">LAY</code>, and <code className="text-amber-300">%N</code> in room/item scripts. <code className="text-amber-300">THINK</code> and <code className="text-amber-300">CONTACT</code> still use the real name, since telepathy isn&rsquo;t something anyone could physically observe</li>
                  <li>New wolf-only emotes: <code className="text-amber-300">WAG</code> (wags tail, overriding the human finger-wag), <code className="text-amber-300">SNIFF</code>, <code className="text-amber-300">LICK</code> (a sloppy lick, replacing the human kiss/lick-all-over-body variants), <code className="text-amber-300">SHAKE</code>, and <code className="text-amber-300">POUNCE</code> (around, or on a target)</li>
                  <li><code className="text-amber-300">LOOK WOLF</code>, and ordinals like <code className="text-amber-300">LOOK 2 WOLF</code>/<code className="text-amber-300">LOOK 3 WOLF</code>, now correctly disambiguate multiple wolves sharing a room &mdash; which also surfaced and fixed a pre-existing bug where the online-player list was built by ranging over a Go map (randomized order on every call), making any &ldquo;2nd/3rd match&rdquo; targeting unstable from one command to the next</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Spellcasting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>A fumbled spell cast (backfire) now lands on a random body part with a real recorded wound, same as any other hit, instead of just subtracting body points with nothing to show for it</li>
                  <li>Prepared spells can now actually be interrupted &mdash; taking a hit while mid-cast (from a monster&rsquo;s attack, special attack, spell, or an item trap) breaks the prepared spell, mirroring the <code className="text-amber-300">NONDISRUPTABLE</code> flag already documented for disrupting a monster&rsquo;s own casting. Wizard&rsquo;s Armor (229) is the ward against it &mdash; while active, taking a hit no longer breaks a prepared spell</li>
                  <li>Added a missing electrical kill-flavor line: &ldquo;Spectacular charge electrolyzes water in body.&rdquo;</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">World Connections</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed a duplicate room-numbering bug in Nyraine (all four seasonal script variants) where three separate content passes had silently overwritten each other&rsquo;s room definitions, leaving the Farmlands area &mdash; and a Rocky Trail/Outcropping/Troll Caves connection &mdash; completely unreachable, with several nearby exits pointing at the wrong rooms. Renumbered the orphaned Farmlands rooms onto free room numbers and reconnected them via City Gates and the coastal Lower Rocky Trail, so the route from the Inner Sea dock through Nyraine to its farmlands finally works end to end</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">GM Tools</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@set</code> can now edit a player&rsquo;s <code className="text-amber-300">EYECOLOR</code>, <code className="text-amber-300">HAIRCOLOR</code>, <code className="text-amber-300">HAIRSTYLE</code>, and <code className="text-amber-300">SKINCOLOR</code> &mdash; previously any string value was rejected before the command even checked which field you&rsquo;d named, so only numeric fields could be set at all. Values are validated against the same fixed choice lists used at character creation, and <code className="text-amber-300">@edpl</code>&rsquo;s summary now also shows age/height/weight/eyes/skin/hair</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.24.0 &mdash; July 30, 2026</h2>
            <p className="text-gray-400 mb-3">WEAR and REMOVE now actually run item scripts, three previously-inert Conjuration spells imbue weapons with a temporary elemental crit, and Thunder Call gets its own flavor text plus a missing electrical kill message.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Script Engine</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">WEAR</code> and <code className="text-amber-300">REMOVE</code> never ran any item scripts at all &mdash; unlike <code className="text-amber-300">GET</code>, <code className="text-amber-300">EAT</code>, <code className="text-amber-300">GO</code>, and <code className="text-amber-300">TOUCH</code>, which already did. Every <code className="text-amber-300">IFPREVERB WEAR/REMOVE</code> and <code className="text-amber-300">IFVERB WEAR/REMOVE</code> block in the world was silently dead code &mdash; cursed items that should resist removal, a heartstone that should graft onto your chest, and a stairway in Idemmu Ag that checks whether you&rsquo;re wearing boots all did nothing. Both commands now run the same script hooks the rest of the verb set already does</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New Spells</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Storm Blade, Inferno Blade, and Winter Blade (135/136/137) previously did nothing at all when cast. They now imbue your wielded weapon with a temporary elemental crit &mdash; 20% chance per hit for 1&ndash;20 bonus damage, lasting 20 minutes (recasting extends the timer, 4-hour cap) &mdash; the same crit mechanic already used by ore-forged elemental weapons. A weapon that already carries a crit of its own (forged-in element or a slayer bonus) can&rsquo;t accept the spell</li>
                  <li>If the weapon has a free adjective slot, it&rsquo;s marked fiery/icy/electric for the duration and reverts automatically when the spell fades; the imbue lives on the weapon itself, so wielding, unwielding, or giving it to someone else carries the effect along for whatever time is left</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat &amp; Spell Messages</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Thunder Call (116) had no unique flavor text of its own; it now beckons a bolt of lightning down from the storm clouds and deals burn damage, matching the original session-capture wording</li>
                  <li>Added a missing kill-flavor line for deaths by electrical damage (both a direct spell hit and a weapon elemental crit), and fixed the same gap for fire-crit kills</li>
                  <li>Damage spells were broadcasting the exact damage number, wound severity, and body part to everyone in the room, not just the caster &mdash; bystanders now see the same vague damage tier (e.g. &ldquo;Awesome damage.&rdquo;) melee combat already shows them</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.23.0 &mdash; July 29, 2026</h2>
            <p className="text-gray-400 mb-3">Karhad&rsquo;s mountain trail now connects both directions, and the long-isolated coastal city of Nyraine is reachable for the first time &mdash; a new sailing ship ferries passengers across the Inner Sea between the two.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">World Connections</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>The Teeth of Shartan trail up to Karhad previously only worked in one direction (down to the Inner Sea region, at the dam) &mdash; a matching climbable trail back up now exists on the Inner Sea side, so the route works both ways</li>
                  <li>Nyraine, a fully-built coastal city that had no connection at all to the rest of the world, is now reachable by sea &mdash; Captain Aldous Kestrel and the crew of the <em>Windrunner</em> run passage between the Inner Sea docks and Nyraine&rsquo;s own dock for two silver each way. <code className="text-amber-300">PAY</code> the captain, then board via the gangplank (<code className="text-amber-300">GO</code>) &mdash; the ship makes way, cuts across open water, and arrives on the far shore over the course of the crossing</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.22.0 &mdash; July 28, 2026</h2>
            <p className="text-gray-400 mb-3">A wide sweep of container bugs (carried and worn alike, coins included), a crafting-economy fix so forged items are actually worth something, several scripted verbs that were silently ignoring an item&rsquo;s own response, and two revived Druidic spells.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Containers</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">GET &lt;coins&gt; FROM &lt;container&gt;</code> never recognized a coin pile sitting inside a container (it&rsquo;s stored differently than a normal item) and reported &ldquo;You don&rsquo;t see that in there.&rdquo; even when <code className="text-amber-300">LOOK IN</code> showed coins present; <code className="text-amber-300">GET ALL FROM</code> was worse &mdash; it silently destroyed any coins in the container instead of adding them to your purse. Both fixed</li>
                  <li>Worn containers (backpack, waistpack, etc.) were invisible to <code className="text-amber-300">OPEN</code>, <code className="text-amber-300">CLOSE</code>, <code className="text-amber-300">PUT &hellip; IN</code>, <code className="text-amber-300">GET &hellip; FROM</code>, and <code className="text-amber-300">GET ALL FROM</code> &mdash; only a container held in your hands or lying on the ground worked. All five now check worn items too, with the same priority a carried container already had</li>
                  <li>Fixed container capacity to match the documented meaning of the two fields: <code className="text-amber-300">INTERIOR</code> caps the total <code className="text-amber-300">VOLUME</code> of everything held inside (it was being used as a raw item-count limit), and a container&rsquo;s own <code className="text-amber-300">VOLUME</code> governs whether a single item is too bulky to fit through the opening at all (it was being used as the total capacity) &mdash; a backpack meant to hold 40 volume of gear was actually being capped by its own small carried-bulk value</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Crafting &amp; Treasure Economy</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>A weapon or piece of armor forged via Weaponsmithing never had a resale value set at all &mdash; it sold for about a copper no matter what metal it was made from. Now scales with the skill level required to craft it, capped at half of whatever the raw metal actually cost to buy, so cheap material can&rsquo;t be laundered into profit by forging and reselling it</li>
                  <li>Monster-dropped jewelry, spell scrolls, and potions found in treasure containers had the identical problem &mdash; no value was ever set on them at all (only gems were handled correctly). All three now scale with treasure level and, for jewelry and potions, the power of any bound spell</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">REPAIR</code> always checked the first inventory item matching the name and gave up immediately if that particular one wasn&rsquo;t damaged &mdash; carrying two items sharing a name (one damaged, one not) meant the damaged one could never be reached and repaired. It now keeps looking until it finds an actually damaged match</li>
                  <li>Weapon Clash now factors in Agility (+1 resistance per 5 points) alongside the weapon&rsquo;s own weight/adjective-based hardness, giving more agile characters better odds of keeping their own weapon undamaged</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Spells</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Sunray (Druidic 519) was dealing heat damage; it&rsquo;s supposed to stun instead &mdash; it now blinds and stuns the target for 4&ndash;6 seconds (two 1&ndash;2 rolls plus 2)</li>
                  <li>Claw Growth (Druidic 518) previously had no effect at all when cast. It now grows natural claws usable whenever you aren&rsquo;t wielding a weapon &mdash; damage and Natural Weapons skill bonus matching a real claw weapon &mdash; lasting 20 minutes (re-casting while active adds another 20 minutes rather than resetting the timer), castable on yourself only, and immune to weapon-clash damage since there&rsquo;s no physical weapon to break</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Script Engine &amp; Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">%A</code> (the capitalized item name/article, e.g. &ldquo;The happy fun ball&rdquo;) was never expanded in scripts and printed literally &mdash; only lowercase <code className="text-amber-300">%a</code> worked</li>
                  <li><code className="text-amber-300">DRINK</code> and <code className="text-amber-300">EAT</code> only ever considered items typed FOOD/LIQUID/LIQCONTAINER, so an item with its own scripted response to one of these verbs but no matching type could never trigger it. Both now fall back to the same generic script dispatch <code className="text-amber-300">PUSH</code>/<code className="text-amber-300">PUNCH</code>/<code className="text-amber-300">TURN</code> already use</li>
                  <li><code className="text-amber-300">FLIP</code> only ever checked items lying in the room, never anything you were carrying</li>
                  <li><code className="text-amber-300">ACTBRIEF</code> now reflects each viewer&rsquo;s own preference &mdash; previously the <em>actor&rsquo;s</em> own ACTBRIEF setting controlled whether everyone else in the room saw parentheses around their <code className="text-amber-300">ACT</code> message, instead of each player&rsquo;s own toggle</li>
                  <li>Fixed the &ldquo;Experience Points until next Build Point&rdquo; counter reading roughly 1000 XP higher than it should for most of a level, due to a mismatched starting-build-point baseline in the calculation</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">New GM Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">@stat &lt;name&gt;</code> and <code className="text-amber-300">@skill &lt;name&gt;</code> &mdash; show a GM the exact same <code className="text-amber-300">STATUS</code> page or <code className="text-amber-300">SKILLS</code> list a player would see themselves, for any player by name (online or offline)</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.21.0 &mdash; July 27, 2026</h2>
            <p className="text-gray-400 mb-3">Two core script-interpreter bugs fixed that silently no-op&rsquo;d across dozens of scripts game-wide, a magic/psi resistance overhaul affecting roughly half the bestiary, several dyeing and weaving fixes, and spell fumbles now actually do something instead of just fizzling.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Script Interpreter</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">IFVAR X == N</code> (double equals) always evaluated false &mdash; only single <code className="text-amber-300">=</code> was recognized as equality. Silently broke every script using the typo&rsquo;d double-equals form, including a Keep gate password puzzle and an orb-tap stairway reveal that could never trigger no matter how correctly you performed the sequence</li>
                  <li><code className="text-amber-300">EQUAL</code>/<code className="text-amber-300">ADD</code>/<code className="text-amber-300">SUB ITEMADJ1-3</code> were a total no-op &mdash; the interpreter had no write case for that variable prefix at all (reads worked, writes silently did nothing). Affects roughly 300 script lines across the world; discovered via a sap reagent that was supposed to change state when lit but never did</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Magic &amp; Psionics</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Monster <code className="text-amber-300">RESIST</code>/<code className="text-amber-300">PSIRESIST</code> values (a rating stat, same family as Attack/Defense, ranging into the thousands) were being compared directly against a 0&ndash;99 roll &mdash; any monster with a resist rating of 100 or higher (roughly 48% of the bestiary) auto-resisted every spell and psi attack, unconditionally, regardless of caster skill</li>
                  <li>Resistance now scales against the caster&rsquo;s own rating (Spellcraft skill + Empathy for spells, Psionics/school skill + Willpower for psi), using the same rating-vs-rating formula melee ToHit already uses &mdash; no monster is unhittable, no resist is a sure thing</li>
                  <li>A fumbled spell cast (roll of 100) previously just printed &ldquo;the spell backfires&rdquo; and did nothing at all. It now actually backfires: damage spells hurt the caster instead of the intended monster, and heal/defense/buff spells land on the caster instead of whoever else they were aimed at</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Dyeing &amp; Weaving</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">DYE</code> only ever searched your own inventory for the material &mdash; cloth left soaking in a dye cauldron (the normal, intended way to dye raw material) couldn&rsquo;t be found at all. Now checks inventory, then the floor, then any container in the room</li>
                  <li><code className="text-amber-300">DYE</code> now requires an actual cauldron present in the room, not just a room flagged as a loom &mdash; most loom rooms (player housing, guild halls) never had one, so dyeing was usable in places it originally shouldn&rsquo;t have been</li>
                  <li>Fixed the dye-color field mapping &mdash; a two-word color like &ldquo;inky black&rdquo; was showing the wrong first word (e.g. &ldquo;amazing black&rdquo; instead of &ldquo;inky black&rdquo;) because the modifier adjective was being read from the wrong item field</li>
                  <li>Crafting a garment from cloth dyed with a two-word color (e.g. &ldquo;olive green cotton&rdquo;) dropped the second word &mdash; the crafted item now keeps both</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Combat &amp; Items</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>A monster knocked down while simultaneously fighting a summoned creature or acting as a guardian never announced getting back up &mdash; it silently recovered with no message, unlike a knockdown during ordinary player combat</li>
                  <li><code className="text-amber-300">GET ALL</code> could scoop up items that are scripted to be un-gettable (e.g. a shop&rsquo;s price-list manuscript that&rsquo;s meant to always refuse <code className="text-amber-300">GET</code>) &mdash; it never ran an item&rsquo;s <code className="text-amber-300">GET</code> script at all, only the single-item <code className="text-amber-300">GET</code> command did. <code className="text-amber-300">GET ALL</code> now respects the same scripted protections</li>
                  <li>Doubled the coin yield from the random lootable containers (chests, strongboxes) monsters drop</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">World Time</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>&ldquo;It is dawn&rdquo; used to cover a 3-hour window, but scripted dawn events (like a fountain&rsquo;s once-a-day effect) trigger on one exact hour in the middle of it &mdash; so two-thirds of the time you saw &ldquo;dawn&rdquo; displayed, the actual triggering hour hadn&rsquo;t arrived yet. Narrowed &ldquo;dawn&rdquo; to the exact hour scripts check</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.20.0 &mdash; July 26, 2026</h2>
            <p className="text-gray-400 mb-3">New room-aware <code className="text-amber-300">ADVICE</code> subcommands for newcomers and crafters, HELP/ADVICE pointed out right at character creation, and two real bugs fixed &mdash; a parser bug mangling every flush-left price sign in the game, and an enchantment spell that was erasing a store-bought item&rsquo;s material adjective.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">ADVICE Command</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">ADVICE HINTS</code> &mdash; 21 newcomer tips covering guild locations, the Test Tunnels, banking, healing-by-resting, death penalties, and where to find player-made maps (Facebook group and PDF)</li>
                  <li><code className="text-amber-300">ADVICE CRAFTING</code> is now room-aware &mdash; away from a workshop it lists the Foundry, Crafter&rsquo;s Guild, Bowyer &amp; Fletcher, and New Havarth Mining Company (room 394); standing in one of them, a master crafter steps up with trade-specific tips (smelting/forging/repair at the forge, pelt/hide prep and dyeing at the loom, Wood Lore and carving at the fletcher&rsquo;s bench, tool/purity tips at the mining shop), including a reminder that a forge or loom doubles as a jeweler&rsquo;s bench for <code className="text-amber-300">ENCRUST</code>, <code className="text-amber-300">INLAY</code>, <code className="text-amber-300">INSET</code>, and <code className="text-amber-300">ENGRAVE</code></li>
                  <li>New characters are now told &ldquo;Type HELP for a full list of commands, or ADVICE to get some tips for getting started.&rdquo; right after character creation, on the web client, telnet, and SSH alike</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Fixes</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Fixed a script-parser bug where a flush-left price list or sign (no leading whitespace or blank lines to mark it as a table) was joined into a single unreadable run-on line instead of keeping its columns &mdash; affects every sign of this style across the world, discovered via the hat shop&rsquo;s price sign in Fayd</li>
                  <li>Enchantment I/II/III (202/203/204) were destroying a store-bought item&rsquo;s material adjective (e.g. casting Enchantment I on leather armor produced &ldquo;enchanted armor&rdquo;, losing &ldquo;leather&rdquo;) &mdash; the spell now fills the first empty adjective slot instead of shifting all three, which was clobbering the variety adjective that purchased items carry in the third slot</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.19.0 &mdash; July 24, 2026</h2>
            <p className="text-gray-400 mb-3">Five long-dormant Druidic spells brought to life &mdash; Camouflage, Call Storm, Disperse Storm, Plant Snare, and Freedom either did nothing or worked incorrectly before today &mdash; plus a full <code className="text-amber-300">SNEAK</code> rework and a new hurricane knockdown mechanic.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Stealth</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Camouflage (Druidic 521) now works &mdash; grants +10 effective Stealth skill for 20 minutes, castable on yourself or another player; additional casts extend the duration instead of stacking the bonus, same as other timed buffs</li>
                  <li><code className="text-amber-300">SNEAK</code> rebuilt: ordinary movement now always reveals a hidden player &mdash; <code className="text-amber-300">SNEAK</code> is required to attempt staying hidden while moving. The roll is now Stealth + Agility/10 + Quickness/10, made harder by the highest Perception/5 among any players already in the room you&rsquo;re moving into, and now takes a 2-second round (1 second under Haste) instead of being instant</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Weather &amp; Storms</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Call Storm (Druidic 501) and Disperse Storm (502) previously had no effect at all when cast; Call Storm now intensifies your region&rsquo;s weather one step toward Hurricane, Disperse Storm calms it one step toward Sunny &mdash; both require being outdoors</li>
                  <li>Call Lightning (503) now requires being outdoors in at least Heavy Rain to cast, matching the original spell notes, instead of striking regardless of weather</li>
                  <li>New Hurricane knockdown &mdash; standing, sitting, or kneeling players caught outdoors in Hurricane-force winds now have a chance each minute to be knocked to the ground; heavier and more agile characters are safer</li>
                  <li>Resist Weather (506) now actually does something &mdash; a 20-minute buff (castable on yourself or another player) that cancels both the new Hurricane knockdown and the weather-based to-hit penalty from severe weather</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Movement-Restricting Spells</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Plant Snare (500) previously had no effect when cast; now entangles another player in grasping roots and vines outdoors, blocking their movement until it wears off or is removed</li>
                  <li>Freedom (505) previously didn&rsquo;t work &mdash; the potion/scroll version cleared an unrelated flag, and the spell itself had no live effect at all when cast; it now removes one active movement-restricting spell (e.g. Plant Snare) from the target, chosen at random if more than one is active</li>
                  <li>New Repel Plants (509) and Repel Plants and Webs (510) &mdash; 20-minute immunity buffs, extending on recast like other timed buffs; the first blocks Plant Snare, the second blocks Plant Snare and Web alike</li>
                  <li>Carapace (511) could previously be cast on other players or creatures like every other defense spell; fixed to caster-only, matching the original spell notes</li>
                </ul>
              </div>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-1">v11.18.0 &mdash; July 23, 2026</h2>
            <p className="text-gray-400 mb-3">Monsters can now actually cast their scripted spells for the first time, special attacks got several real bug fixes (target selection, missing flavor text, damage over-reduction, duplicated messages), and container commands now check what&rsquo;s in your hands before what&rsquo;s on the ground.</p>

            <div className="space-y-4 mb-8">
              <div>
                <h3 className="text-green-400 font-bold mb-1">Monster Spellcasting</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>Monsters configured with a spell (<code className="text-amber-300">SPELLUSE</code>/<code className="text-amber-300">SPELL</code>/<code className="text-amber-300">MANA</code>) can now actually cast it &mdash; this was fully parsed but never once invoked, so no monster in the game had ever cast a spell despite plenty being configured to</li>
                  <li>Full prepare-then-cast sequence: a windup announcement, then the spell&rsquo;s own cast time (matching how player spellcasting works), then a gesture line and the spell&rsquo;s normal hit-flavor text and damage &mdash; landing a hit on the monster while it&rsquo;s mid-cast disrupts the spell (unless the monster is flagged unable to be disrupted), and its mana is spent regardless</li>
                  <li>Spell damage to a player now uses the same dice, damage type, and mitigations (armor, elemental vulnerability/resistance, shields) as when a player casts that same spell at a monster &mdash; it no longer also stacks the melee hit-location reduction (20&ndash;40% for a limb or hand) on top, which was driving it far below what the spell should do</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Monster Special Attacks</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li>A monster&rsquo;s special attack (and spell cast) now targets a random player currently fighting it, or a random player in the room if nobody currently is &mdash; instead of being limited to whichever single player it happened to be locked onto in melee</li>
                  <li>A monster&rsquo;s limited number of special-attack uses is now actually enforced &mdash; previously unlimited regardless of how the monster was configured</li>
                  <li>Fixed a parser bug where a special attack&rsquo;s own scripted flavor text was never loaded due to a stray typo&rsquo;d token, so every special attack fell back to a generic &ldquo;uses a special attack&rdquo; line instead of things like &ldquo;turns one head to face you and looses a gout of flame!&rdquo;</li>
                  <li>Onlookers in the room now see a simplified outcome (e.g. &ldquo;&hellip; Minor damage.&rdquo;) when a special attack or spell lands &mdash; previously only the target&rsquo;s own private message showed that anything happened at all</li>
                  <li>The player actually hit no longer sees their own special attack or spell hit described twice &mdash; once in private detail, once again in the public room recap &mdash; the room broadcast now leaves out whoever already got the private version</li>
                </ul>
              </div>
              <div>
                <h3 className="text-green-400 font-bold mb-1">Container Commands</h3>
                <ul className="text-gray-300 space-y-1 ml-4 list-disc">
                  <li><code className="text-amber-300">OPEN</code>/<code className="text-amber-300">CLOSE</code>/<code className="text-amber-300">LOCK</code>/<code className="text-amber-300">UNLOCK</code>/lockpicking now check what you&rsquo;re holding before checking the room &mdash; previously all of these checked room items first, so holding a locked chest with an identical one lying nearby meant these commands would act on the one on the ground instead of the one in your hands (use <code className="text-amber-300">my &lt;item&gt;</code> to force your own inventory explicitly, same as before)</li>
                  <li><code className="text-amber-300">LOCK</code>/<code className="text-amber-300">UNLOCK</code> previously couldn&rsquo;t affect anything in your inventory at all, only room items &mdash; a keyed chest you were holding could never be locked or unlocked without first setting it down</li>
                </ul>
              </div>
            </div>
          </section>

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
