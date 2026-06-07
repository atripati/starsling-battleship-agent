package main

import "encoding/json"

const (
	ServerURL     = "https://intern-battleship-game-server.vercel.app"
	CompetitionID = "295cccc9137b5335cc581d67d655d6fa3b41dac6610dad0e7ed201625523ad8c"
	AgentID       = "sqHJJ4iRnuXikiYrZYAVbcZK4MPvYtN3"
)
const (
	RespMoveRequired        = "MOVE_REQUIRED"
	RespGameCompleted       = "GAME_COMPLETED"
	RespAttemptCompleted    = "ATTEMPT_COMPLETED"
	RespAttemptDisqualified = "ATTEMPT_DISQUALIFIED"
)
const (
	MovePlaceShips = "PLACE_SHIPS"
	MoveSubmitShot = "SUBMIT_SHOT"
)
const (
	OutcomeMiss = "MISS"
	OutcomeHit  = "HIT"
	OutcomeSink = "SINK"
)
const (
	GameAgentWin    = "AGENT_WIN"
	GameOpponentWin = "OPPONENT_WIN"
)
const (
	OrientHorizontal = "HORIZONTAL"
	OrientVertical   = "VERTICAL"
)

type Envelope struct {
	ResponseType string          `json:"responseType"`
	State        *GameState      `json:"state,omitempty"`
	GameOutcome  string          `json:"gameOutcome,omitempty"`
	Next         *Envelope       `json:"next,omitempty"`
	Result       *AttemptResult  `json:"result,omitempty"`
	Reason       string          `json:"reason,omitempty"`
	AttemptID    string          `json:"attemptId,omitempty"`
	Context      json.RawMessage `json:"context,omitempty"`
	Code         string          `json:"code,omitempty"`
	Message      string          `json:"message,omitempty"`
}
type GameState struct {
	CompetitionID           string       `json:"competitionId"`
	GameOrdinal             int          `json:"gameOrdinal"`
	TotalGames              int          `json:"totalGames"`
	Opponent                Opponent     `json:"opponent"`
	NextRequiredMove        string       `json:"nextRequiredMove"`
	NextMoveDeadlineAt      string       `json:"nextMoveDeadlineAt"`
	Board                   BoardRules   `json:"board"`
	YourFleet               []ShipInfo   `json:"yourFleet"`
	YourShots               []ShotRecord `json:"yourShots"`
	IncomingShots           []ShotRecord `json:"incomingShots"`
	SunkOpponentShipClasses []string     `json:"sunkOpponentShipClasses"`
}
type Opponent struct {
	OpponentID    string `json:"opponentId"`
	DisplayName   string `json:"displayName"`
	OpponentClass string `json:"opponentClass"`
	BaseScore     int    `json:"baseScore"`
}
type BoardRules struct {
	GridRows       int         `json:"gridRows"`
	GridCols       int         `json:"gridCols"`
	ShipClasses    []ShipClass `json:"shipClasses"`
	AllowAdjacency bool        `json:"allowAdjacency"`
}
type ShipClass struct {
	Class  string `json:"class"`
	Length int    `json:"length"`
}
type ShipInfo struct {
	ShipClass   string `json:"shipClass"`
	Orientation string `json:"orientation"`
	StartRow    int    `json:"startRow"`
	StartCol    int    `json:"startCol"`
	Sunk        bool   `json:"sunk"`
}
type ShotRecord struct {
	Row           int    `json:"row"`
	Col           int    `json:"col"`
	Outcome       string `json:"outcome"`
	SunkShipClass string `json:"sunkShipClass,omitempty"`
}
type Placement struct {
	ShipClass   string `json:"shipClass"`
	Orientation string `json:"orientation"`
	StartRow    int    `json:"startRow"`
	StartCol    int    `json:"startCol"`
}
type PlacementRequest struct {
	Placements []Placement `json:"placements"`
}

type ShotRequest struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

type AttemptResult struct {
	AttemptID         string `json:"attemptId"`
	FinalScore        int    `json:"finalScore"`
	Wins              int    `json:"wins"`
	Losses            int    `json:"losses"`
	HitDifferential   int    `json:"hitDifferential"`
	OpponentShipsSunk int    `json:"opponentShipsSunk"`
	AgentShipsLost    int    `json:"agentShipsLost"`
	IsNewBest         bool   `json:"isNewBest"`
	CompletionMessage string `json:"completionMessage"`
}
type GameRecord struct {
	GameOrdinal  int     `json:"gameOrdinal"`
	OpponentID   string  `json:"opponentId"`
	OpponentName string  `json:"opponentName"`
	Class        string  `json:"class"`
	Outcome      string  `json:"outcome"`
	ShotsTotal   int     `json:"shotsTotal"`
	ShotsHit     int     `json:"shotsHit"`
	Accuracy     float64 `json:"accuracy"`
	ShipsLost    int     `json:"shipsLost"`
	ShipsSunk    int     `json:"shipsSunk"`
}
type AttemptRecord struct {
	AttemptNum int          `json:"attemptNum"`
	Score      int          `json:"score"`
	Wins       int          `json:"wins"`
	Losses     int          `json:"losses"`
	Games      []GameRecord `json:"games"`
}
type Memory struct {
	BestScore int             `json:"bestScore"`
	Attempts  []AttemptRecord `json:"attempts"`
}
