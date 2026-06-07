package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type signResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
}

func mintJWT() (string, error) {
	start := time.Now()

	cmd := exec.Command("auth-agent", "sign", AgentID,
		"--capabilities", "getCompetitionRules", "createAttempt",
		"getCurrentAttempt", "placeShips", "submitShot", "abandonAttempt")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("auth-agent sign failed: %s (stderr: %s)", err, string(out))
	}
	raw := out
	for i := 0; i < len(raw); i++ {
		if raw[i] == '{' {
			raw = raw[i:]
			break
		}
	}
	var resp signResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("parse sign response: %w (raw: %s)", err, string(out))
	}
	if resp.Token == "" {
		return "", fmt.Errorf("auth-agent returned empty token")
	}
	elapsed := time.Since(start)
	if elapsed > 3*time.Second {
		fmt.Printf("    ⚠ JWT mint took %s (slow — risk of timeout)\n", elapsed.Round(time.Millisecond))
	}
	return resp.Token, nil
}
