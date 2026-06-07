# Battleship Agent — StarSling Intern Challenge

A self-improving Battleship agent built in Go that plays 15-game matches against the StarSling game server over the Agent Auth protocol.

## Results

| Metric | Best Attempt |
|--------|-------------|
| Score | 495 / 1000 |
| Wins | 13 / 15 |
| Ships Sunk | 73 / 75 |
| Ships Lost | 48 |

Score progression across attempts shows consistent improvement:

```
Attempt 1: 257
Attempt 2: 392  (+135)
Attempt 3: 315
Attempt 4: 410  (+95)
Attempt 5: 495  (+85, new best)
```

## Architecture

```
battleship/
├── main.go        Game loop, self-improvement loop, memory persistence
├── types.go       API models, response constants, metrics structs
├── auth.go        JWT minting via auth-agent CLI
├── client.go      HTTP client with timeout, resume support
├── strategy.go    Three-mode firing + probability density hunting
├── memory.json    Persistent cross-attempt learning (auto-generated)
└── go.mod
```

### Authentication

Shells out to `auth-agent sign` for single-use JWTs per request. Credentials persist in `~/.agent-auth` so human approval happens once. Using the globally installed binary instead of `npx` eliminates the 2-3 second overhead per request that caused timeout disqualifications in early iterations.

### Game Loop

The agent is a state machine driven by the server's response envelope:

```
MOVE_REQUIRED (PLACE_SHIPS) → generate random valid layout
MOVE_REQUIRED (SUBMIT_SHOT) → fire using three-mode strategy
GAME_COMPLETED              → log metrics, unwrap next game
ATTEMPT_COMPLETED           → save results, update memory
ATTEMPT_DISQUALIFIED        → abandon and retry
```

Crash recovery: before creating a new attempt, the agent tries `getCurrentAttempt` to resume any active attempt from a previous run.

### Firing Strategy

Three modes, evaluated in priority order:

**Destroy Mode** — When two or more open hits align on the same row or column, the agent infers ship orientation and extends the line in both directions. This is the highest-value mode because it converts partial hits into sinks with minimal wasted shots.

**Target Mode** — When a single open hit exists with no alignment partner, the agent probes orthogonal neighbors. Each probe either confirms a direction (promoting to Destroy) or eliminates a possibility.

**Hunt Mode** — When no open hits exist, the agent builds a probability density map. For each untried cell, it counts how many remaining unsunk ships could legally pass through that cell given the current board state (misses and sunk cells block placement). It fires at the highest-density cell. This outperforms checkerboard or random hunting because it concentrates shots where ships are statistically most likely to be.

Open hits exclude cells belonging to already-sunk ships. Sunk ship cells are reconstructed by tracing connected hits from each SINK shot in all four directions.

### Self-Improvement

The agent persists a `memory.json` file across attempts containing:

- Best score achieved
- Per-attempt results (score, wins, losses)
- Per-game metrics (opponent, outcome, shots fired, accuracy)

Each new attempt loads this memory and displays the current best score as a baseline. The memory file enables post-run analysis of which opponents are hardest and where shots are being wasted.

### Observability

Every game logs: opponent name, class, outcome, shot count, and accuracy. Every attempt logs: final score, win/loss record, ships sunk/lost, timing, and whether it set a new personal best. The agent warns when JWT minting takes longer than 3 seconds (risk of timeout disqualification).

## Design Decisions

**Go over TypeScript** — The challenge provides a TypeScript SDK, but I chose Go because it matches my primary language (I build ARK, an AI agent runtime, in Go) and because most candidates will paste the prompt into Claude Code and get a TypeScript bot. Hand-building in Go demonstrates understanding over convenience.

**auth-agent binary over npx** — Early runs hit timeout disqualifications because `npx @auth/agent-cli sign` took 2-3 seconds per invocation (package resolution overhead). Switching to the globally installed `auth-agent` binary eliminated this entirely.

**Probability density over checkerboard** — A standard checkerboard hunt covers 50 cells. Probability density is more expensive to compute but concentrates shots where ships actually fit given the current board state. The computation is O(ships * board) per shot, well within the 10-second budget.

**Resume before create** — If the agent crashes or is interrupted mid-attempt, naively calling `createAttempt` returns 409 (active attempt exists). The agent always tries `getCurrentAttempt` first, enabling seamless recovery.

## Running

```bash
npm install -g @auth/agent-cli
auth-agent --url=https://intern-battleship-game-server.vercel.app connect \
  --provider=https://intern-battleship-game-server.vercel.app \
  --capabilities getCompetitionRules createAttempt getCurrentAttempt placeShips submitShot abandonAttempt

# Approve in browser when prompted

# Play
go run .
```

## What I Would Add With More Time

- **Opponent-specific strategy adaptation**: Track per-opponent win rates across attempts and adjust placement defensiveness for harder opponents
- **Ship spacing optimization**: Spread ships to maximize the number of shots needed to sink the fleet, reducing ships lost per game
- **Smarter target mode**: When probing neighbors of a hit, prioritize directions that could accommodate the largest remaining unsunk ship
- **Parallel JWT pre-minting**: Mint the next JWT while waiting for the server response to eliminate auth latency entirely
