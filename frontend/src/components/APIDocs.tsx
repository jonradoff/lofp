export default function APIDocs({ onBack }: { onBack: () => void }) {
  return (
    <div className="flex items-start justify-center h-full p-8 overflow-y-auto">
      <div className="max-w-3xl w-full font-mono">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-amber-500 text-2xl font-bold">Bot API Documentation</h1>
          <button onClick={onBack} className="text-gray-400 hover:text-white text-sm">&larr; Back</button>
        </div>

        <div className="bg-[#111] border border-[#333] rounded-lg p-4 mb-6">
          <p className="text-gray-300 text-sm leading-relaxed">
            Control game characters programmatically via the WebSocket API. Bots were not part of the original
            1992&ndash;1999 game &mdash; this is a new feature added to help repopulate the world. Bots can&rsquo;t
            do anything a human player can&rsquo;t; they follow the same rules and rate limits.
          </p>
          <p className="text-gray-400 text-xs mt-3">
            <a href="/bot-agent-spec.md" className="text-amber-400 hover:text-amber-300 underline">
              Machine-readable agent specification (bot-agent-spec.md)
            </a>
            &nbsp;&mdash; for AI agents and automated tools.
          </p>
        </div>

        <div className="space-y-6 text-sm">

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-2">1. Getting an API Key</h2>
            <ol className="text-gray-300 space-y-2 ml-4 list-decimal">
              <li>Log in at <a href="https://lofp.metavert.io" className="text-amber-400 hover:underline">lofp.metavert.io</a></li>
              <li>On the main menu, click <span className="text-amber-300">&#9881; Bot</span> next to your character</li>
              <li>Click <span className="text-amber-300">Generate Key</span> (optionally check &ldquo;Allow GM commands&rdquo; for GM characters)</li>
              <li><strong>Copy the key immediately</strong> &mdash; it&rsquo;s only shown once (SHA-256 hashed in storage)</li>
              <li>To get a new key, generate again &mdash; the old one is automatically revoked</li>
            </ol>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-2">2. Connection Flow</h2>
            <ol className="text-gray-300 space-y-2 ml-4 list-decimal">
              <li>Connect to <code className="text-green-400 bg-[#0a0a0a] px-1 rounded">wss://lofp.metavert.io/ws/game</code></li>
              <li>You&rsquo;ll receive a welcome message</li>
              <li>Send an <code className="text-green-400 bg-[#0a0a0a] px-1 rounded">auth_apikey</code> message with your key</li>
              <li>On success, your character logs in and you receive the room description</li>
              <li>Send <code className="text-green-400 bg-[#0a0a0a] px-1 rounded">command</code> messages to play</li>
            </ol>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-2">3. Message Format</h2>
            <p className="text-gray-400 mb-3">All messages are JSON with <code className="text-green-400">type</code> and <code className="text-green-400">data</code> fields.</p>

            <h3 className="text-green-400 font-bold mb-1">Sending</h3>
            <pre className="bg-[#0a0a0a] border border-[#333] rounded p-3 text-xs text-gray-300 overflow-x-auto mb-4">{`// Authenticate
{"type": "auth_apikey", "data": {"key": "lofp_abc123..."}}

// Send a game command
{"type": "command", "data": {"input": "look"}}
{"type": "command", "data": {"input": "attack skeleton"}}
{"type": "command", "data": {"input": "say Hello everyone!"}}`}</pre>

            <h3 className="text-green-400 font-bold mb-1">Receiving</h3>
            <pre className="bg-[#0a0a0a] border border-[#333] rounded p-3 text-xs text-gray-300 overflow-x-auto mb-4">{`// Auth result
{"type": "auth_result", "data": {"success": true, "character": "MyBot"}}

// Game output (from your commands)
{"type": "result", "data": {"messages": ["[City Gate]", "You stand at..."], "roomName": "[City Gate]"}}

// Broadcast (from other players/monsters)
{"type": "broadcast", "data": {"messages": ["A skeleton attacks you!"]}}`}</pre>

            <h3 className="text-green-400 font-bold mb-1">Response Fields</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead><tr className="text-gray-400 border-b border-[#333]">
                  <th className="text-left py-1 pr-3">Field</th><th className="text-left py-1">Description</th>
                </tr></thead>
                <tbody className="text-gray-300">
                  <tr className="border-b border-[#222]"><td className="py-1 pr-3 text-amber-300">messages</td><td>Array of text lines</td></tr>
                  <tr className="border-b border-[#222]"><td className="py-1 pr-3 text-amber-300">roomName</td><td>Current room name (on LOOK)</td></tr>
                  <tr className="border-b border-[#222]"><td className="py-1 pr-3 text-amber-300">roomDesc</td><td>Room description text</td></tr>
                  <tr className="border-b border-[#222]"><td className="py-1 pr-3 text-amber-300">exits</td><td>Array of exit direction names</td></tr>
                  <tr className="border-b border-[#222]"><td className="py-1 pr-3 text-amber-300">items</td><td>Array of visible item names</td></tr>
                  <tr className="border-b border-[#222]"><td className="py-1 pr-3 text-amber-300">playerState</td><td>Character stats (BP, mana, level, etc.)</td></tr>
                  <tr><td className="py-1 pr-3 text-amber-300">promptIndicators</td><td>Status codes: !=bleeding, J=combat, S=stunned, etc.</td></tr>
                </tbody>
              </table>
            </div>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-2">4. Bot Behavior</h2>
            <ul className="text-gray-300 space-y-1 ml-4 list-disc">
              <li>Bots appear on WHO with <span className="text-amber-300">[Bot]</span> next to their name</li>
              <li>Rate limit: 10 commands/sec (same as human players)</li>
              <li>Chat flood: 5 broadcast messages per 10 seconds</li>
              <li>GM commands require &ldquo;Allow GM commands&rdquo; to be checked when generating the key</li>
              <li>Bots follow all normal game rules (combat, death, roundtime)</li>
            </ul>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-2">5. Code Examples</h2>

            <h3 className="text-green-400 font-bold mb-1 mt-4">Python</h3>
            <pre className="bg-[#0a0a0a] border border-[#333] rounded p-3 text-xs text-gray-300 overflow-x-auto mb-4">{`# pip install websocket-client
import json, os, websocket

API_KEY = os.environ["LOFP_API_KEY"]
ws = websocket.WebSocket()
ws.connect("wss://lofp.metavert.io/ws/game")

# Skip welcome message
ws.recv()

# Authenticate
ws.send(json.dumps({"type": "auth_apikey", "data": {"key": API_KEY}}))
result = json.loads(ws.recv())
print("Auth:", result)

# Send a command
ws.send(json.dumps({"type": "command", "data": {"input": "look"}}))
result = json.loads(ws.recv())
for line in result.get("data", {}).get("messages", []):
    print(line)`}</pre>

            <h3 className="text-green-400 font-bold mb-1">Node.js</h3>
            <pre className="bg-[#0a0a0a] border border-[#333] rounded p-3 text-xs text-gray-300 overflow-x-auto mb-4">{`// npm install ws
const WebSocket = require("ws");
const ws = new WebSocket("wss://lofp.metavert.io/ws/game");

ws.on("open", () => {
  ws.send(JSON.stringify({
    type: "auth_apikey",
    data: { key: process.env.LOFP_API_KEY }
  }));
});

ws.on("message", (raw) => {
  const msg = JSON.parse(raw);
  if (msg.type === "auth_result" && msg.data.success) {
    ws.send(JSON.stringify({
      type: "command", data: { input: "look" }
    }));
  }
  if (msg.data?.messages) {
    msg.data.messages.forEach(line => console.log(line));
  }
});`}</pre>

            <h3 className="text-green-400 font-bold mb-1">TypeScript</h3>
            <pre className="bg-[#0a0a0a] border border-[#333] rounded p-3 text-xs text-gray-300 overflow-x-auto mb-4">{`// npm install ws @types/ws tsx
import WebSocket from "ws";

interface GameMessage {
  type: string;
  data: {
    success?: boolean;
    character?: string;
    messages?: string[];
    roomName?: string;
    playerState?: { bodyPoints: number; maxBodyPoints: number };
  };
}

const ws = new WebSocket("wss://lofp.metavert.io/ws/game");

ws.on("open", () => {
  ws.send(JSON.stringify({
    type: "auth_apikey",
    data: { key: process.env.LOFP_API_KEY }
  }));
});

ws.on("message", (raw: WebSocket.Data) => {
  const msg: GameMessage = JSON.parse(raw.toString());
  if (msg.type === "auth_result" && msg.data.success) {
    console.log(\`Logged in as \${msg.data.character}\`);
    ws.send(JSON.stringify({
      type: "command", data: { input: "look" }
    }));
  }
  for (const line of msg.data.messages ?? []) {
    console.log(line);
  }
});`}</pre>
          </section>

          <section>
            <h2 className="text-amber-400 text-lg font-bold mb-2">6. Full Examples</h2>
            <p className="text-gray-300 text-sm">
              Complete runnable bot examples with error handling and event loops are available in the{' '}
              <a href="https://github.com/jonradoff/lofp/tree/main/bots" className="text-amber-400 hover:underline">
                /bots directory on GitHub
              </a>.
            </p>
          </section>

        </div>

        <div className="mt-8 pt-4 border-t border-[#333] text-gray-600 text-xs text-center">
          Legends of Future Past &mdash; Bot API &mdash; MIT License
        </div>
      </div>
    </div>
  )
}
