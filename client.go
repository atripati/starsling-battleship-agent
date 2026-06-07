package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var basePath = fmt.Sprintf("%s/competitions/%s", ServerURL, CompetitionID)
var httpClient = &http.Client{
	Timeout: 8 * time.Second,
}

func doRequest(method, path string, body interface{}) (*Envelope, error) {
	token, err := mintJWT()
	if err != nil {
		return nil, fmt.Errorf("mint JWT: %w", err)
	}

	var reqBody io.Reader
	var hasBody bool
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(data)
		hasBody = true
	}

	url := basePath + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if reqID := resp.Header.Get("x-request-id"); reqID != "" {
		_ = reqID
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d %s: %s", resp.StatusCode, path, string(respBody))
	}
	var env Envelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
	}
	return &env, nil
}
func getRules() (*Envelope, error) {
	return doRequest("GET", "/rules", nil)
}
func createAttempt() (*Envelope, error) {
	return doRequest("POST", "/attempts", nil)
}
func getCurrentAttempt() (*Envelope, error) {
	return doRequest("GET", "/attempts/current", nil)
}
func placeShips(placements []Placement) (*Envelope, error) {
	return doRequest("POST", "/attempts/current/placements", PlacementRequest{Placements: placements})
}
func submitShot(row, col int) (*Envelope, error) {
	return doRequest("POST", "/attempts/current/shots", ShotRequest{Row: row, Col: col})
}
func abandonAttempt() (*Envelope, error) {
	return doRequest("POST", "/attempts/current/abandon", nil)
}
func resumeOrCreateAttempt() (*Envelope, error) {
	env, err := getCurrentAttempt()
	if err == nil && env.ResponseType == RespMoveRequired {
		fmt.Println("  ♻ Resumed existing active attempt")
		return env, nil
	}
	return createAttempt()
}
