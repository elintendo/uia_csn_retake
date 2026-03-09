package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type LogEntry struct {
	Level    string `json:"level"`
	Message  string `json:"message"`
	Source   string `json:"source"`
	ClientID string `json:"client_id"`
}

func main() {
	message := flag.String("message", "", "Log message")
	level := flag.String("level", "info", "Log level (info, warn, error)")
	server := flag.String("server", "http://localhost:8080", "Server URL")
	source := flag.String("source", "", "Log source")
	continuous := flag.Bool("continuous", false, "Send logs continuously")
	interval := flag.Int("interval", 5, "Interval in seconds for continuous mode")
	flag.Parse()

	if *source == "" {
		hostname, _ := os.Hostname()
		*source = hostname
	}

	clientID := uuid.New().String()

	if *continuous {
		runContinuous(*server, *source, clientID, *interval)
	} else {
		if *message == "" {
			fmt.Println("Usage: client -message \"your log message\" [-level info|warn|error] [-server URL] [-source name]")
			fmt.Println("       client -continuous [-interval seconds] [-server URL] [-source name]")
			os.Exit(1)
		}
		sendLog(*server, *level, *message, *source, clientID)
	}
}

func sendLog(server, level, message, source, clientID string) {
	entry := LogEntry{
		Level:    level,
		Message:  message,
		Source:   source,
		ClientID: clientID,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(server+"/logs", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Failed to send log: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Println("Log sent successfully")
	} else {
		fmt.Printf("Failed to send log: %s\n", resp.Status)
	}
}

func runContinuous(server, source, clientID string, interval int) {
	fmt.Printf("Starting continuous log generation (client: %s)\n", clientID[:8])
	fmt.Printf("Sending logs every %d seconds. Press Ctrl+C to stop.\n", interval)

	messages := []struct {
		level   string
		message string
	}{
		{"info", "Application started successfully"},
		{"info", "Processing request"},
		{"warn", "High memory usage detected"},
		{"error", "Database connection timeout"},
		{"info", "Cache cleared"},
		{"warn", "Slow query detected"},
		{"error", "Failed to connect to external service"},
		{"info", "User authentication successful"},
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		msg := messages[rand.Intn(len(messages))]
		sendLog(server, msg.level, msg.message, source, clientID)
	}
}
