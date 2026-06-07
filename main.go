package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

const memoryFile = "memory.json"

func loadMemory() *Memory {
	data, err := os.ReadFile(memoryFile)
	if err != nil {
		return &Memory{}
	}
	var mem Memory
	if err := json.Unmarshal(data, &mem); err != nil {
		return &Memory{}
	}
	return &mem
}
func saveMemory(mem *Memory) {
	data, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		fmt.Printf("  ⚠ Failed to save memory: %v\n", err)
		return
	}
	os.WriteFile(memoryFile, data, 0644)
}
func playAttempt(attemptNum int, mem *Memory) (*AttemptResult, []GameRecord, error) {
	fmt.Printf("\n══════════════════════════════════════════\n")
	fmt.Printf("  ATTEMPT %d", attemptNum)
	if mem.BestScore > 0 {
		fmt.Printf(" (best so far: %d)", mem.BestScore)
	}
	fmt.Printf("\n══════════════════════════════════════════\n\n")
	env, err := resumeOrCreateAttempt()
	if err != nil {
		return nil, nil, fmt.Errorf("start attempt: %w", err)
	}
	var games []GameRecord
	var currentGame *GameRecord
	shotCount := 0
	hitCount := 0
	for {
		switch env.ResponseType {
		case RespMoveRequired:
			state := env.State

			if state.NextRequiredMove == MovePlaceShips {
				shotCount = 0
				hitCount = 0
				currentGame = &GameRecord{
					GameOrdinal:  state.GameOrdinal,
					OpponentID:   state.Opponent.OpponentID,
					OpponentName: state.Opponent.DisplayName,
					Class:        state.Opponent.OpponentClass,
				}

				fmt.Printf("  Game %d/%d vs %-28s [%s]  ",
					state.GameOrdinal, state.TotalGames,
					state.Opponent.DisplayName, state.Opponent.OpponentClass)

				layout := chooseLayout(state)
				env, err = placeShips(layout)
				if err != nil {
					return nil, games, fmt.Errorf("placeShips game %d: %w", state.GameOrdinal, err)
				}

			} else {
				row, col := chooseShot(state)
				env, err = submitShot(row, col)
				if err != nil {
					return nil, games, fmt.Errorf("submitShot game %d: %w", state.GameOrdinal, err)
				}
				shotCount++
				if len(state.YourShots) > 0 {
					last := state.YourShots[len(state.YourShots)-1]
					if last.Outcome == OutcomeHit || last.Outcome == OutcomeSink {
						hitCount++
					}
				}
			}

		case RespGameCompleted:
			if currentGame != nil {
				if env.GameOutcome == GameAgentWin {
					currentGame.Outcome = "WIN"
				} else {
					currentGame.Outcome = "LOSS"
				}
				currentGame.ShotsTotal = shotCount
				currentGame.ShotsHit = hitCount
				if shotCount > 0 {
					currentGame.Accuracy = float64(hitCount) / float64(shotCount) * 100
				}
				games = append(games, *currentGame)

				icon := "✅"
				if env.GameOutcome == GameOpponentWin {
					icon = "❌"
				}
				fmt.Printf("%s %s  (%d shots, %.0f%% accuracy)\n",
					icon, env.GameOutcome, shotCount, currentGame.Accuracy)
			}
			env = env.Next

		case RespAttemptCompleted:
			if currentGame != nil && currentGame.Outcome == "" {
				currentGame.Outcome = "WIN"
				currentGame.ShotsTotal = shotCount
				currentGame.ShotsHit = hitCount
				if shotCount > 0 {
					currentGame.Accuracy = float64(hitCount) / float64(shotCount) * 100
				}
				games = append(games, *currentGame)
				fmt.Printf("✅ %s  (%d shots, %.0f%% accuracy)\n",
					GameAgentWin, shotCount, currentGame.Accuracy)
			}

			r := env.Result
			fmt.Printf("\n  ┌─────────────────────────────────┐\n")
			fmt.Printf("  │  ATTEMPT %d COMPLETE              │\n", attemptNum)
			fmt.Printf("  ├─────────────────────────────────┤\n")
			fmt.Printf("  │  Score:      %4d / 1000         │\n", r.FinalScore)
			fmt.Printf("  │  Wins:       %2d  | Losses: %2d    │\n", r.Wins, r.Losses)
			fmt.Printf("  │  Ships sunk: %2d  | Lost:   %2d    │\n", r.OpponentShipsSunk, r.AgentShipsLost)
			fmt.Printf("  │  New best:   %v               │\n", r.IsNewBest)
			fmt.Printf("  └─────────────────────────────────┘\n")
			return r, games, nil

		case RespAttemptDisqualified:
			return nil, games, fmt.Errorf("DISQUALIFIED: %s", env.Reason)

		default:
			return nil, games, fmt.Errorf("unknown responseType: %s", env.ResponseType)
		}
	}
}

func main() {
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║  BATTLESHIP AGENT — StarSling Challenge  ║")
	fmt.Println("║  Self-improving closed-loop agent        ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	mem := loadMemory()
	if mem.BestScore > 0 {
		fmt.Printf("\n  📂 Loaded memory: best score %d from %d previous attempts\n", mem.BestScore, len(mem.Attempts))
	}
	fmt.Println("\n  Verifying auth...")
	_, err := getRules()
	if err != nil {
		log.Fatalf("  Auth check failed: %v", err)
	}
	fmt.Println("  ✅ Auth working")
	maxAttempts := 5
	var scores []int

	for i := 1; i <= maxAttempts; i++ {
		start := time.Now()
		result, games, err := playAttempt(i, mem)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("\n  ❌ Attempt %d failed: %v\n", i, err)
			abandonAttempt()
			fmt.Println("  Waiting 3 seconds before next attempt...")
			time.Sleep(3 * time.Second)
			continue
		}

		scores = append(scores, result.FinalScore)
		fmt.Printf("  ⏱ Time: %s\n", elapsed.Round(time.Millisecond))
		record := AttemptRecord{
			AttemptNum: i,
			Score:      result.FinalScore,
			Wins:       result.Wins,
			Losses:     result.Losses,
			Games:      games,
		}
		mem.Attempts = append(mem.Attempts, record)
		if result.FinalScore > mem.BestScore {
			mem.BestScore = result.FinalScore
		}
		saveMemory(mem)
		if len(scores) > 1 {
			fmt.Printf("\n  📈 Score history: %v\n", scores)
			best := scores[0]
			for _, s := range scores {
				if s > best {
					best = s
				}
			}
			fmt.Printf("  Best: %d / 1000\n", best)
		}
		if i < maxAttempts {
			fmt.Println("\n  ⏳ Waiting 3 seconds before next attempt...")
			time.Sleep(3 * time.Second)
		}
	}
	fmt.Println("\n══════════════════════════════════════════")
	fmt.Println("  FINAL SUMMARY")
	fmt.Println("══════════════════════════════════════════")
	fmt.Printf("  Attempts: %d\n", len(scores))
	fmt.Printf("  Scores:   %v\n", scores)
	if len(scores) > 0 {
		best := scores[0]
		for _, s := range scores {
			if s > best {
				best = s
			}
		}
		fmt.Printf("  Best:     %d / 1000\n", best)
	}
	fmt.Printf("  Memory saved to %s\n", memoryFile)
	fmt.Println("══════════════════════════════════════════")
}
