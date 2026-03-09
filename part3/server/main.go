package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

type Config struct {
	Server ServerConfig `json:"server"`
	LLM    LLMConfig    `json:"llm"`
}

type ServerConfig struct {
	Port         string `json:"port"`
	DatabasePath string `json:"database_path"`
}

type LLMConfig struct {
	Provider       string `json:"provider"`
	APIURL         string `json:"api_url"`
	APIKey         string `json:"api_key"`
	Model          string `json:"model"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	MaxTokens      int    `json:"max_tokens"`
}

type LogEntry struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
	ClientID  string    `json:"client_id"`
}

type Server struct {
	db            *sql.DB
	llmService    *LLMService
	activeClients sync.Map
	mu            sync.RWMutex
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	server, err := NewServer(config)
	if err != nil {
		log.Fatal("Failed to initialize server:", err)
	}
	defer server.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/logs", server.handleLogs)
	mux.HandleFunc("/logs/analyze", server.handleAnalyze)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/stats", server.handleStats)

	port := config.Server.Port
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on port %s", port)
	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal("Server error:", err)
	}
}

func loadConfig() (*Config, error) {
	// Default configuration
	config := &Config{
		Server: ServerConfig{
			Port:         "8080",
			DatabasePath: "./logs.db",
		},
		LLM: LLMConfig{
			Provider:       "openrouter",
			APIURL:         "https://openrouter.ai/api/v1/chat/completions",
			APIKey:         "",
			Model:          "arcee-ai/trinity-large-preview:free",
			TimeoutSeconds: 30,
			MaxTokens:      500,
		},
	}

	// Try to load from config.json
	file, err := os.Open("config.json")
	if err != nil {
		// Config file not found, check environment variable
		if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
			config.LLM.APIKey = apiKey
		}
		log.Println("No config.json found, using defaults and environment variables")
		return config, nil
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}

	// Environment variable overrides config file
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		config.LLM.APIKey = apiKey
	}

	return config, nil
}

func NewServer(config *Config) (*Server, error) {
	db, err := sql.Open("sqlite", config.Server.DatabasePath)
	if err != nil {
		return nil, err
	}

	if err := initDatabase(db); err != nil {
		return nil, err
	}

	return &Server{
		db:         db,
		llmService: NewLLMService(config),
	}, nil
}

func initDatabase(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME,
		level TEXT,
		message TEXT,
		source TEXT,
		client_id TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_level ON logs(level);
	`
	_, err := db.Exec(schema)
	return err
}

func (s *Server) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		s.handleLogSubmission(w, r)
	case "GET":
		s.handleLogRetrieval(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLogSubmission(w http.ResponseWriter, r *http.Request) {
	var entry LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	entry.Timestamp = time.Now()
	s.activeClients.Store(entry.ClientID, time.Now())

	s.mu.Lock()
	_, err := s.db.Exec(
		"INSERT INTO logs (timestamp, level, message, source, client_id) VALUES (?, ?, ?, ?, ?)",
		entry.Timestamp, entry.Level, entry.Message, entry.Source, entry.ClientID,
	)
	s.mu.Unlock()

	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Log received entry in real-time
	log.Printf("[%s] %s | %s | %s | client: %s",
		entry.Timestamp.Format("15:04:05"),
		entry.Level,
		entry.Source,
		entry.Message,
		entry.ClientID[:8])

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleLogRetrieval(w http.ResponseWriter, r *http.Request) {
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "100"
	}

	s.mu.RLock()
	rows, err := s.db.Query(
		"SELECT id, timestamp, level, message, source, client_id FROM logs ORDER BY timestamp DESC LIMIT ?",
		limit,
	)
	s.mu.RUnlock()

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []LogEntry
	for rows.Next() {
		var entry LogEntry
		if err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.Level, &entry.Message, &entry.Source, &entry.ClientID); err != nil {
			continue
		}
		logs = append(logs, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	rows, err := s.db.Query("SELECT level, message, source FROM logs ORDER BY timestamp DESC LIMIT 50")
	s.mu.RUnlock()

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []map[string]string
	for rows.Next() {
		var level, message, source string
		rows.Scan(&level, &message, &source)
		logs = append(logs, map[string]string{
			"level":   level,
			"message": message,
			"source":  source,
		})
	}

	analysis, err := s.llmService.AnalyzeLogs(logs)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"analysis": "LLM service unavailable: " + err.Error(),
			"status":   "degraded",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"analysis": analysis,
		"status":   "ok",
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	var totalLogs, errorCount int
	s.db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&totalLogs)
	s.db.QueryRow("SELECT COUNT(*) FROM logs WHERE level = 'error'").Scan(&errorCount)
	s.mu.RUnlock()

	activeCount := 0
	s.activeClients.Range(func(key, value interface{}) bool {
		activeCount++
		return true
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_logs":     totalLogs,
		"error_count":    errorCount,
		"active_clients": activeCount,
	})
}
