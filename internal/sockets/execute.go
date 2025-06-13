package sockets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Alter-Sitanshu/CodeEditor/internal/env"
)

var API_KEY string = env.GetString("JUDGE0_KEY", "")

type Judge0Executor struct {
	baseURL string
	client  *http.Client
}

type ExecuteRequest struct {
	Code     string `json:"code" validate:"required"`
	Language string `json:"language" validate:"required,oneof=go python javascript java"`
	Input    string `json:"input"`
}

type ExecuteResponse struct {
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code"`
	Runtime  string `json:"runtime"`
	Status   string `json:"status"`
}

type Judge0Request struct {
	SourceCode string `json:"source_code"`
	LanguageID int    `json:"language_id"`
	Stdin      string `json:"stdin,omitempty"`
}

type Judge0Response struct {
	Token  string `json:"token"`
	Status struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
	} `json:"status"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Time   string `json:"time"`
}

func NewJudge0Executor() *Judge0Executor {
	return &Judge0Executor{
		baseURL: "https://judge0-ce.p.rapidapi.com",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *Judge0Executor) ExecuteCode(req ExecuteRequest) (*ExecuteResponse, error) {
	langID := e.getLanguageID(req.Language)
	if langID == 0 {
		return &ExecuteResponse{
			Error:  "Unsupported language",
			Status: "error",
		}, nil
	}

	judgeReq := Judge0Request{
		SourceCode: req.Code,
		LanguageID: langID,
		Stdin:      req.Input,
	}

	jsonData, _ := json.Marshal(judgeReq)

	httpReq, _ := http.NewRequest("POST", e.baseURL+"/submissions", bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-rapidapi-host", "judge0-ce.p.rapidapi.com")
	httpReq.Header.Set("x-rapidapi-key", API_KEY)

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return &ExecuteResponse{
			Error:  fmt.Sprintf("Request failed: %v", err),
			Status: "error",
		}, nil
	}
	defer resp.Body.Close()

	var submitResp struct {
		Token string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&submitResp)

	return e.pollResult(submitResp.Token)
}

func (e *Judge0Executor) getLanguageID(language string) int {
	// Judge0 language IDs
	langMap := map[string]int{
		"go":         95, // Go 1.18.5
		"python":     92, // Python 3.10.0
		"javascript": 93, // Node.js 18.15.0
		"java":       91, // Java 17.0.6
	}
	return langMap[language]
}

func (e *Judge0Executor) pollResult(token string) (*ExecuteResponse, error) {
	maxAttempts := 10
	pollInterval := 2 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Get submission result
		url := fmt.Sprintf("%s/submissions/%s", e.baseURL, token)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-rapidapi-host", "judge0-ce.p.rapidapi.com")
		req.Header.Set("x-rapidapi-key", API_KEY)

		resp, err := e.client.Do(req)
		if err != nil {
			return &ExecuteResponse{
				Error:  fmt.Sprintf("Poll request failed: %v", err),
				Status: "error",
			}, nil
		}
		defer resp.Body.Close()

		var judgeResp Judge0Response
		if err := json.NewDecoder(resp.Body).Decode(&judgeResp); err != nil {
			return &ExecuteResponse{
				Error:  fmt.Sprintf("Failed to decode response: %v", err),
				Status: "error",
			}, nil
		}

		// Checking if execution is complete
		statusID := judgeResp.Status.ID

		if statusID <= 2 {
			// Still processing
			time.Sleep(pollInterval)
			continue
		}

		// Execution completed, return result
		response := &ExecuteResponse{
			Output:  judgeResp.Stdout,
			Runtime: judgeResp.Time,
		}

		if statusID == 3 {
			response.Status = "success"
			response.ExitCode = 0
		} else {
			response.Status = "error"
			response.Error = judgeResp.Stderr
			if response.Error == "" {
				response.Error = judgeResp.Status.Description
			}
			response.ExitCode = 1
		}

		return response, nil
	}

	return &ExecuteResponse{
		Error:  "Execution timeout - result not ready",
		Status: "timeout",
	}, nil
}
