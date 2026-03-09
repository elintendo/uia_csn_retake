package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Source  string `json:"source"`
}

func main() {
	message := flag.String("message", "", "Log message")
	level := flag.String("level", "info", "Log level (info, warn, error)")
	server := flag.String("server", "http://localhost:8080", "Server URL")
	source := flag.String("source", "", "Log source")
	flag.Parse()

	if *message == "" {
		fmt.Println("Usage: client -message \"your log message\" [-level info|warn|error] [-server URL] [-source name]")
		os.Exit(1)
	}

	if *source == "" {
		hostname, _ := os.Hostname()
		*source = hostname
	}

	entry := LogEntry{
		Level:   *level,
		Message: *message,
		Source:  *source,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(*server+"/logs", "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Println("Log sent successfully")
	} else {
		fmt.Printf("Failed to send log: %s\n", resp.Status)
	}
}
