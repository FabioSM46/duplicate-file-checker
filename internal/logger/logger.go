package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gocarina/gocsv"
)

// LogEntry represents a log entry that includes the file path, hash, and deletion status.
type LogEntry struct {
	Path    string `json:"path"`
	Hash    string `json:"hash"`
	Deleted bool   `json:"deleted"`
}

// WriteLog writes the duplicate information to a file based on the selected format.
func WriteLog(logData []map[string]interface{}, logFormat string) {
	// Create log file
	logFile, err := os.Create("./duplicates_log." + logFormat)
	if err != nil {
		log.Fatalf("Error creating log file: %v", err)
	}
	defer logFile.Close()

	switch logFormat {
	case "json":
		// Write JSON log
		var logEntries []LogEntry

		// Convert the logData into LogEntry objects
		for _, entry := range logData {
			// Check and assert types safely
			file, fileOk := entry["file"].(string)
			hash, hashOk := entry["hash"].(string)
			deleted, deletedOk := entry["deleted"].(bool)

			if !fileOk || !deletedOk {
				continue
			}

			// If the hash is missing, set it to an empty string
			if !hashOk {
				hash = ""
			}

			logEntries = append(logEntries, LogEntry{
				Path:    file,
				Hash:    hash,
				Deleted: deleted,
			})
		}

		// Encode and save the JSON log
		encoder := json.NewEncoder(logFile)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(logEntries)
		if err != nil {
			log.Fatalf("Error writing JSON log: %v", err)
		}
		fmt.Println("JSON log saved as duplicates_log.json")

	case "csv":
		// Write CSV log
		var csvData [][]string
		for _, entry := range logData {
			// Check and assert types safely
			hash, hashOk := entry["hash"].(string)
			file, fileOk := entry["file"].(string)
			deleted, deletedOk := entry["deleted"].(bool)

			if !hashOk || !fileOk || !deletedOk {
				continue
			}

			csvData = append(csvData, []string{
				hash,
				file,
				fmt.Sprintf("%v", deleted),
			})
		}

		err := gocsv.MarshalFile(&csvData, logFile)
		if err != nil {
			log.Fatalf("Error writing CSV log: %v", err)
		}
		fmt.Println("CSV log saved as duplicates_log.csv")

	case "plain":
		// Write plain text log
		for _, entry := range logData {
			// Check and assert types safely
			hash, hashOk := entry["hash"].(string)
			file, fileOk := entry["file"].(string)
			deleted, deletedOk := entry["deleted"].(bool)

			if !hashOk || !fileOk || !deletedOk {
				continue
			}

			_, err := logFile.WriteString(fmt.Sprintf("Hash: %s\n", hash))
			if err != nil {
				log.Fatalf("Error writing to log: %v", err)
			}
			_, err = logFile.WriteString(fmt.Sprintf("  File: %s\n", file))
			if err != nil {
				log.Fatalf("Error writing to log: %v", err)
			}
			_, err = logFile.WriteString(fmt.Sprintf("  Deleted: %v\n", deleted))
			if err != nil {
				log.Fatalf("Error writing to log: %v", err)
			}
			_, err = logFile.WriteString("\n")
			if err != nil {
				log.Fatalf("Error writing to log: %v", err)
			}
		}
		fmt.Println("Plain text log saved as duplicates_log.txt")
	}
}
