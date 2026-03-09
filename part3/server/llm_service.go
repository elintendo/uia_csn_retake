package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type LLMService struct {
	apiKey     string
	apiURL     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

func NewLLMService(config *Config) *LLMService {
	return &LLMService{
		apiKey:    config.LLM.APIKey,
		apiURL:    config.LLM.APIURL,
		model:     config.LLM.Model,
		maxTokens: config.LLM.MaxTokens,
		httpClient: &http.Client{
			Timeout: time.Duration(config.LLM.TimeoutSeconds) * time.Second,
		},
	}
}

func (llm *LLMService) AnalyzeLogs(logs []map[string]string) (string, error) {
	if llm.apiKey == "" {
		return "", errors.New("LLM API key not configured")
	}

	logSummary := ""
	for _, log := range logs {
		logSummary += fmt.Sprintf("[%s] %s: %s\n", log["source"], log["level"], log["message"])
	}

	prompt := `Analyze these log entries and provide:
1) Summary of main issues
2) Severity classification (Critical/High/Medium/Low)
3) Recommendations for system administrators

Logs:
` + logSummary

	reqBody := map[string]interface{}{
		"model": llm.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": llm.maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", llm.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+llm.apiKey)

	resp, err := llm.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", errors.New("unable to parse LLM response")
}
