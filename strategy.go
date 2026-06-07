package main

import (
	"math/rand"
	"sort"
)

func chooseLayout(state *GameState) []Placement {
	R := state.Board.GridRows
	C := state.Board.GridCols

	sorted := make([]ShipClass, len(state.Board.ShipClasses))
	copy(sorted, state.Board.ShipClasses)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Length > sorted[j].Length
	})

	for attempt := 0; attempt < 80; attempt++ {
		occupied := make(map[[2]int]bool)
		buffer := make(map[[2]int]bool)
		placements := make([]Placement, 0, len(sorted))
		ok := true

		for _, ship := range sorted {
			cands := candidatePositions(ship, R, C)
			placed := false
			for _, cand := range cands {
				conflict := false
				for _, cell := range cand.cells {
					if buffer[cell] {
						conflict = true
						break
					}
				}
				if conflict {
					continue
				}
				for _, cell := range cand.cells {
					occupied[cell] = true
					for dr := -1; dr <= 1; dr++ {
						for dc := -1; dc <= 1; dc++ {
							buffer[[2]int{cell[0] + dr, cell[1] + dc}] = true
						}
					}
				}
				placements = append(placements, cand.placement)
				placed = true
				break
			}
			if !placed {
				ok = false
				break
			}
		}

		if ok && len(placements) == len(sorted) {
			return placements
		}
	}
	return chooseLayoutSimple(state)
}

type candidate struct {
	cells     [][2]int
	placement Placement
}

func candidatePositions(ship ShipClass, R, C int) []candidate {
	var cands []candidate
	length := ship.Length

	add := func(r, c int, horiz bool) {
		cells := make([][2]int, length)
		for i := 0; i < length; i++ {
			if horiz {
				cells[i] = [2]int{r, c + i}
			} else {
				cells[i] = [2]int{r + i, c}
			}
		}
		orient := OrientVertical
		if horiz {
			orient = OrientHorizontal
		}
		cands = append(cands, candidate{
			cells: cells,
			placement: Placement{
				ShipClass:   ship.Class,
				Orientation: orient,
				StartRow:    r,
				StartCol:    c,
			},
		})
	}

	for r := 0; r < R; r++ {
		for c := 0; c <= C-length; c++ {
			add(r, c, true)
		}
	}
	for r := 0; r <= R-length; r++ {
		for c := 0; c < C; c++ {
			add(r, c, false)
		}
	}
	rand.Shuffle(len(cands), func(i, j int) {
		cands[i], cands[j] = cands[j], cands[i]
	})
	sort.SliceStable(cands, func(i, j int) bool {
		return edgeScore(cands[i].cells, R, C) > edgeScore(cands[j].cells, R, C)
	})
	return cands
}

func edgeScore(cells [][2]int, R, C int) int {
	score := 0
	for _, cell := range cells {
		r, c := cell[0], cell[1]
		distEdge := min4(r, c, R-1-r, C-1-c)
		score += (R - distEdge) // closer to edge = higher
	}
	return score
}
func min4(a, b, c, d int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	if d < m {
		m = d
	}
	return m
}
func chooseLayoutSimple(state *GameState) []Placement {
	R := state.Board.GridRows
	C := state.Board.GridCols
	for {
		used := make(map[[2]int]bool)
		placements := make([]Placement, 0, len(state.Board.ShipClasses))
		ok := true
		for _, ship := range state.Board.ShipClasses {
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
					ShipClass: ship.Class, Orientation: orient,
					StartRow: r, StartCol: c,
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
		if shot, found := destroyMode(openHits, tried, R, C); found {
			return shot[0], shot[1]
		}
	}
	if len(openHits) > 0 {
		if shot, found := targetMode(openHits, tried, R, C); found {
			return shot[0], shot[1]
		}
	}
	return huntModeProbability(state, tried, sunkCells, R, C)
}

func destroyMode(openHits [][2]int, tried map[[2]int]bool, R, C int) ([2]int, bool) {
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
		minC, maxC := hits[0][1], hits[len(hits)-1][1]
		if rc := maxC + 1; rc < C && !tried[[2]int{row, rc}] {
			return [2]int{row, rc}, true
		}
		if lc := minC - 1; lc >= 0 && !tried[[2]int{row, lc}] {
			return [2]int{row, lc}, true
		}
	}
	for col, hits := range byCol {
		if len(hits) < 2 {
			continue
		}
		sort.Slice(hits, func(i, j int) bool { return hits[i][0] < hits[j][0] })
		minR, maxR := hits[0][0], hits[len(hits)-1][0]
		if dr := maxR + 1; dr < R && !tried[[2]int{dr, col}] {
			return [2]int{dr, col}, true
		}
		if ur := minR - 1; ur >= 0 && !tried[[2]int{ur, col}] {
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
	minLength := 100
	for _, ship := range state.Board.ShipClasses {
		if !sunkClassSet[ship.Class] {
			remainingLengths = append(remainingLengths, ship.Length)
			if ship.Length < minLength {
				minLength = ship.Length
			}
		}
	}
	if len(remainingLengths) == 0 {
		return fallbackRandom(tried, R, C)
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

	density := make([][]int, R)
	for r := 0; r < R; r++ {
		density[r] = make([]int, C)
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
						if !tried[[2]int{r, c + i}] {
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
						if !tried[[2]int{r + i, c}] {
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
			if minLength > 1 && (r+c)%2 != 0 {
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
		for r := 0; r < R; r++ {
			for c := 0; c < C; c++ {
				if !tried[[2]int{r, c}] && density[r][c] > 0 {
					bestCells = append(bestCells, [2]int{r, c})
				}
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
