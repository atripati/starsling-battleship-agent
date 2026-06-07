# Battleship Agent — StarSling Intern Challenge

A self-improving Battleship agent built in Go that plays 15-game matches against the StarSling game server over the Agent Auth protocol. The agent authenticates, places a fleet, and plays each game to completion using an adaptive firing strategy, persisting results across runs to track improvement.

## Results

Best attempt: **532 / 1000** with **14 wins, 1 loss**.

| Metric | Best Attempt |
|--------|-------------|
| Score | 532 / 1000 |
| Wins | 14 / 15 |
| Ships Sunk | 74 / 75 |
| Ships Lost | 47 |

Scores are consistent across runs (404, 472, 532, 474, 523), averaging ~480 with low variance — the agent reliably wins 12–14 of 15 games every attempt rather than getting lucky once.

## Architecture

```
battleship/
├── main.go        Game loop, self-improvement loop, memory persistence
├── types.go       API models, response constants, metrics structs
├── auth.go        JWT minting via auth-agent CLI
├── client.go      HTTP client with timeout + crash recovery
├── strategy.go    Defensive placement + three-mode firing
├── memory.json    Persistent cross-attempt learning (auto-generated)
└── go.mod
```

The code is organized by responsibility: transport (`client.go`, `auth.go`), domain models (`types.go`), decision logic (`strategy.go`), and orchestration (`main.go`). Strategy is fully decoupled from transport — the firing and placement functions take game state and return moves, with no knowledge of HTTP or auth.

## How It Works

### Authentication

The agent shells out to the `auth-agent` CLI to mint a fresh single-use JWT per request (each carries a one-time `jti` for replay protection). Credentials persist in `~/.agent-auth`, so a human approves the agent only once. An early version used `npx @auth/agent-cli` per request; the 2–3 second package-resolution overhead pushed some moves past the 10-second deadline and caused timeout disqualifications. Switching to the globally installed binary eliminated this, cutting per-request auth latency to well under a second.

### Game Loop

The agent is a state machine driven by the server's response envelope, switching on `responseType`:

```
MOVE_REQUIRED (PLACE_SHIPS) → place a validated fleet
MOVE_REQUIRED (SUBMIT_SHOT) → fire using the three-mode strategy
GAME_COMPLETED              → record metrics, unwrap .next, continue
ATTEMPT_COMPLETED           → save results, update memory
ATTEMPT_DISQUALIFIED        → abandon and retry the attempt
```

### Crash Recovery

Before creating a new attempt, the agent calls `getCurrentAttempt`. If an attempt is already active (e.g. from a crashed prior run), it resumes rather than calling `createAttempt` and hitting `409 ACTIVE_ATTEMPT_EXISTS`. This makes the agent restart-safe.

### Firing Strategy

Three modes, evaluated in priority order:

**Destroy** — When two or more unresolved hits align on the same row or column, the agent infers the ship's orientation and extends the line outward from both ends. This converts a partial hit into a sink with the fewest possible shots.

**Target** — A single unresolved hit with no alignment partner: probe the four orthogonal neighbors. The first hit among them promotes the ship to Destroy mode.

**Hunt** — No unresolved hits: build a probability-density map. For every untried cell, count how many placements of each still-unsunk ship could legally pass through it, given that misses and sunk cells block placement. Fire at the highest-density cell. A parity (checkerboard) filter is applied while the smallest surviving ship has length ≥ 2, halving the search space without ever skipping a reachable ship.

"Unresolved" hits exclude cells belonging to already-sunk ships. Sunk-ship cells are reconstructed by tracing connected hits outward from each SINK shot in all four directions, so the agent never wastes turns targeting around a ship it already destroyed.

### Defensive Placement

Scoring is dominated by ship-loss penalties: winning 14 of 15 games still nets a moderate score if you bleed 55+ of your own ship-cells along the way. The agent's placement is built to minimize losses:

- **Buffer zones** — no two ships touch, even diagonally. When an opponent hits one ship and switches to its own destroy mode, it cannot chain into an adjacent ship.
- **Edge bias** — candidate positions are scored by proximity to the board edge and tried edge-first. A density-based opponent hunts the center first, so edge-placed ships survive longer.

Switching from uniform-random to buffered edge-biased placement dropped average ships lost from ~60 to ~47 and lifted the best score from 495 to 532.

### Self-Improvement

A `memory.json` file persists across runs, storing the best score, every attempt's win/loss record, and per-game metrics (opponent, outcome, shots fired, accuracy). Each run loads this and reports the current best as a baseline, so the agent's progress is visible and analyzable across sessions rather than starting cold every time.

### Observability

Every game logs opponent, class, outcome, shot count, and accuracy on one line. Every attempt prints a result card (score, win/loss, ships sunk/lost, timing, new-best flag). The agent warns when a JWT mint exceeds 3 seconds, surfacing timeout risk before it causes a disqualification. The intent is that the console output alone tells you exactly what the agent did and why a score landed where it did.

## Design Decisions

**Go over the provided TypeScript SDK.** The challenge ships a TypeScript reference agent. I built in Go because it is my primary language and because hand-writing the auth handshake, transport, and strategy demonstrates understanding rather than wiring up an SDK.

**Optimize ship losses, not win count.** The biggest score lever, once you are already winning most games, is the penalty for losing your own ships — not squeezing out one more win. That insight drove the defensive-placement work and produced the largest single jump in score.

**Pragmatic self-improvement.** Rather than an online learning loop (overkill for a fixed 15-opponent roster in a 3-hour window), the agent runs multiple attempts, persists results, and reports the trend. The strategy improvements between runs were driven by reading the metrics, which is the loop that actually mattered here.

## Running

```bash
# One-time setup
npm install -g @auth/agent-cli
auth-agent --url=https://intern-battleship-game-server.vercel.app connect \
  --provider=https://intern-battleship-game-server.vercel.app \
  --capabilities getCompetitionRules createAttempt getCurrentAttempt placeShips submitShot abandonAttempt
# Approve in the browser when prompted

# Play
go run .
```

## What I Would Add With More Time

- **Conditional target PDF**: replace the heuristic destroy/target modes with a probability map filtered to placements that cover the current unresolved hits — mathematically optimal continuation without hardcoded direction logic.
- **Opponent-specific adaptation**: track per-opponent results across attempts and raise placement defensiveness against the hardest WARSHIP opponents.
- **Parallel JWT pre-minting**: mint the next token while the current request is in flight to remove auth latency from the critical path entirely.
- **Maximum-separation placement**: solve for the layout that maximizes the expected number of opponent shots to clear the fleet, further reducing ship-loss penalties.
