package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

type LogEntry struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite", "./logs.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME,
		level TEXT,
		message TEXT,
		source TEXT
	)`)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/logs", handleLogs)
	http.HandleFunc("/logs/analyze", handleAnalyze)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var entry LogEntry
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		entry.Timestamp = time.Now()
		_, err := db.Exec("INSERT INTO logs (timestamp, level, message, source) VALUES (?, ?, ?, ?)",
			entry.Timestamp, entry.Level, entry.Message, entry.Source)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	case "GET":
		rows, err := db.Query("SELECT id, timestamp, level, message, source FROM logs ORDER BY timestamp DESC LIMIT 100")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var logs []LogEntry
		for rows.Next() {
			var entry LogEntry
			err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.Level, &entry.Message, &entry.Source)
			if err != nil {
				continue
			}
			logs = append(logs, entry)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logs)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get recent logs
	rows, err := db.Query("SELECT level, message FROM logs ORDER BY timestamp DESC LIMIT 50")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logSummary string
	for rows.Next() {
		var level, message string
		rows.Scan(&level, &message)
		logSummary += level + ": " + message + "\n"
	}

	// Call LLM service (placeholder - needs API key)
	analysis := analyzeLogs(logSummary)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"analysis": analysis})
}

func analyzeLogs(logs string) string {
	apiKey := "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"
	// apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return "LLM analysis not configured. Set OPENROUTER_API_KEY environment variable."
	}

	prompt := "Analyze these log entries and provide: 1) Summary of issues, 2) Severity classification, 3) Recommendations:\n\n" + logs

	reqBody := map[string]interface{}{
		"model": "arcee-ai/trinity-large-preview:free",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 500,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "Error creating request: " + err.Error()
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "Error calling LLM API: " + err.Error()
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content
				}
			}
		}
	}

	return "Unable to parse LLM response"
}
