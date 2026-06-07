package main

import (
	"math/rand"
	"sort"
)

func chooseLayout(state *GameState) []Placement {
	R := state.Board.GridRows
	C := state.Board.GridCols
	ships := state.Board.ShipClasses
	for {
		used := make(map[[2]int]bool)
		placements := make([]Placement, 0, len(ships))
		ok := true
		for _, ship := range ships {
			placed := false
			for tries := 0; tries < 200; tries++ {
				horiz := rand.Intn(2) == 0
				length := ship.Length
				var r, c int
				if horiz {
					r = rand.Intn(R)
					c = rand.Intn(C - length + 1)
				} else {
					r = rand.Intn(R - length + 1)
					c = rand.Intn(C)
				}
				cells := make([][2]int, length)
				overlap := false
				for i := 0; i < length; i++ {
					if horiz {
						cells[i] = [2]int{r, c + i}
					} else {
						cells[i] = [2]int{r + i, c}
					}
					if used[cells[i]] {
						overlap = true
						break
					}
				}
				if overlap {
					continue
				}
				for _, cell := range cells {
					used[cell] = true
				}
				orient := OrientVertical
				if horiz {
					orient = OrientHorizontal
				}
				placements = append(placements, Placement{
					ShipClass:   ship.Class,
					Orientation: orient,
					StartRow:    r,
					StartCol:    c,
				})
				placed = true
				break
			}
			if !placed {
				ok = false
				break
			}
		}
		if ok {
			return placements
		}
	}
}
func chooseShot(state *GameState) (int, int) {
	R := state.Board.GridRows
	C := state.Board.GridCols

	tried := make(map[[2]int]bool)
	for _, s := range state.YourShots {
		tried[[2]int{s.Row, s.Col}] = true
	}

	sunkCells := buildSunkCells(state)
	var openHits [][2]int
	for _, s := range state.YourShots {
		if s.Outcome == OutcomeHit && !sunkCells[[2]int{s.Row, s.Col}] {
			openHits = append(openHits, [2]int{s.Row, s.Col})
		}
	}
	if len(openHits) >= 2 {
		shot, found := destroyMode(openHits, tried, R, C)
		if found {
			return shot[0], shot[1]
		}
	}
	if len(openHits) > 0 {
		shot, found := targetMode(openHits, tried, R, C)
		if found {
			return shot[0], shot[1]
		}
	}
	return huntModeProbability(state, tried, sunkCells, R, C)
}
func destroyMode(openHits [][2]int, tried map[[2]int]bool, R, C int) ([2]int, bool) {
	hitSet := make(map[[2]int]bool)
	for _, h := range openHits {
		hitSet[h] = true
	}
	byRow := make(map[int][][2]int)
	byCol := make(map[int][][2]int)
	for _, h := range openHits {
		byRow[h[0]] = append(byRow[h[0]], h)
		byCol[h[1]] = append(byCol[h[1]], h)
	}
	for row, hits := range byRow {
		if len(hits) < 2 {
			continue
		}
		sort.Slice(hits, func(i, j int) bool { return hits[i][1] < hits[j][1] })
		minC := hits[0][1]
		maxC := hits[len(hits)-1][1]
		rc := maxC + 1
		if rc < C && !tried[[2]int{row, rc}] {
			return [2]int{row, rc}, true
		}
		lc := minC - 1
		if lc >= 0 && !tried[[2]int{row, lc}] {
			return [2]int{row, lc}, true
		}
	}
	for col, hits := range byCol {
		if len(hits) < 2 {
			continue
		}
		sort.Slice(hits, func(i, j int) bool { return hits[i][0] < hits[j][0] })
		minR := hits[0][0]
		maxR := hits[len(hits)-1][0]
		dr := maxR + 1
		if dr < R && !tried[[2]int{dr, col}] {
			return [2]int{dr, col}, true
		}
		ur := minR - 1
		if ur >= 0 && !tried[[2]int{ur, col}] {
			return [2]int{ur, col}, true
		}
	}

	return [2]int{}, false
}
func targetMode(openHits [][2]int, tried map[[2]int]bool, R, C int) ([2]int, bool) {
	dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for _, hit := range openHits {
		for _, d := range dirs {
			nr, nc := hit[0]+d[0], hit[1]+d[1]
			if nr >= 0 && nr < R && nc >= 0 && nc < C && !tried[[2]int{nr, nc}] {
				return [2]int{nr, nc}, true
			}
		}
	}
	return [2]int{}, false
}
func huntModeProbability(state *GameState, tried map[[2]int]bool, sunkCells map[[2]int]bool, R, C int) (int, int) {

	sunkClassSet := make(map[string]bool)
	for _, c := range state.SunkOpponentShipClasses {
		sunkClassSet[c] = true
	}
	var remainingLengths []int
	for _, ship := range state.Board.ShipClasses {
		if !sunkClassSet[ship.Class] {
			remainingLengths = append(remainingLengths, ship.Length)
		}
	}
	if len(remainingLengths) == 0 {
		return fallbackRandom(tried, R, C)
	}
	density := make([][]int, R)
	for r := 0; r < R; r++ {
		density[r] = make([]int, C)
	}
	blocked := make(map[[2]int]bool)
	for _, s := range state.YourShots {
		if s.Outcome == OutcomeMiss {
			blocked[[2]int{s.Row, s.Col}] = true
		}
	}
	for cell := range sunkCells {
		blocked[cell] = true
	}

	for _, length := range remainingLengths {
		for r := 0; r < R; r++ {
			for c := 0; c <= C-length; c++ {
				valid := true
				for i := 0; i < length; i++ {
					if blocked[[2]int{r, c + i}] {
						valid = false
						break
					}
				}
				if valid {
					for i := 0; i < length; i++ {
						cell := [2]int{r, c + i}
						if !tried[cell] {
							density[r][c+i]++
						}
					}
				}
			}
		}
		for r := 0; r <= R-length; r++ {
			for c := 0; c < C; c++ {
				valid := true
				for i := 0; i < length; i++ {
					if blocked[[2]int{r + i, c}] {
						valid = false
						break
					}
				}
				if valid {
					for i := 0; i < length; i++ {
						cell := [2]int{r + i, c}
						if !tried[cell] {
							density[r+i][c]++
						}
					}
				}
			}
		}
	}
	bestScore := -1
	var bestCells [][2]int
	for r := 0; r < R; r++ {
		for c := 0; c < C; c++ {
			if tried[[2]int{r, c}] {
				continue
			}
			if density[r][c] > bestScore {
				bestScore = density[r][c]
				bestCells = [][2]int{{r, c}}
			} else if density[r][c] == bestScore {
				bestCells = append(bestCells, [2]int{r, c})
			}
		}
	}

	if len(bestCells) == 0 {
		return fallbackRandom(tried, R, C)
	}
	pick := bestCells[rand.Intn(len(bestCells))]
	return pick[0], pick[1]
}
func fallbackRandom(tried map[[2]int]bool, R, C int) (int, int) {
	var candidates [][2]int
	for r := 0; r < R; r++ {
		for c := 0; c < C; c++ {
			if !tried[[2]int{r, c}] {
				candidates = append(candidates, [2]int{r, c})
			}
		}
	}
	pick := candidates[rand.Intn(len(candidates))]
	return pick[0], pick[1]
}
func buildSunkCells(state *GameState) map[[2]int]bool {
	sunkCells := make(map[[2]int]bool)
	for _, s := range state.YourShots {
		if s.Outcome == OutcomeSink {
			sunkCells[[2]int{s.Row, s.Col}] = true
			markConnectedHits(state, s.Row, s.Col, sunkCells)
		}
	}
	return sunkCells
}
func markConnectedHits(state *GameState, sinkRow, sinkCol int, sunkCells map[[2]int]bool) {
	shotMap := make(map[[2]int]string)
	for _, s := range state.YourShots {
		shotMap[[2]int{s.Row, s.Col}] = s.Outcome
	}
	dirs := [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	for _, d := range dirs {
		r, c := sinkRow+d[0], sinkCol+d[1]
		for {
			outcome, exists := shotMap[[2]int{r, c}]
			if !exists || outcome == OutcomeMiss {
				break
			}
			sunkCells[[2]int{r, c}] = true
			r += d[0]
			c += d[1]
		}
	}
}
