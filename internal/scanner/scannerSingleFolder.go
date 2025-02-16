package scanner

import (
	"duplicate-file-checker/internal/hashing"
	"duplicate-file-checker/internal/logger"
	"fmt"
	"os"
	"sync"

	"github.com/schollz/progressbar/v3"
)

func ProcessSingleFolder(dirPath, hashMethod string, dryRun bool, logFormat string, maxWorkers int) {
	// Scan directory for files
	files, err := ScanDirectory(dirPath)
	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		return
	}

	// Create a buffered channel to hold results
	resultChan := make(chan map[string]string, len(files))

	// Set up a worker pool with maxWorkers
	var wg sync.WaitGroup
	workerPool := make(chan struct{}, maxWorkers)

	// Create progress bar
	bar := progressbar.NewOptions(len(files), progressbar.OptionSetDescription("Hashing files"), progressbar.OptionSetWidth(50))

	// Start workers for parallel file hashing
	for _, filePath := range files {
		wg.Add(1)
		workerPool <- struct{}{}

		go func(path string) {
			defer wg.Done()
			defer func() { <-workerPool }()

			hashValue, err := hashing.HashFile(path, hashMethod)
			if err != nil {
				fmt.Printf("Error hashing file %s: %v\n", path, err)
				return
			}

			resultChan <- map[string]string{hashValue: path}
			bar.Add(1)
		}(filePath)
	}

	// Close resultChan after all goroutines are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and process them
	duplicates := make(map[string][]string)
	for result := range resultChan {
		for hash, filePath := range result {
			duplicates[hash] = append(duplicates[hash], filePath)
		}
	}

	// Prepare log data slice
	logData := []map[string]interface{}{}

	// List to hold duplicates to be deleted
	var duplicatesToDelete []string

	// Collect duplicates (excluding the first file, keeping one copy)
	for _, files := range duplicates {
		if len(files) > 1 {
			duplicatesToDelete = append(duplicatesToDelete, files[1:]...)
		}
	}

	// **Fix: Only ask for confirmation if duplicates exist**
	if len(duplicatesToDelete) > 0 {
		fmt.Println("Duplicates found (these files will be deleted):")
		for _, file := range duplicatesToDelete {
			fmt.Printf("  %s\n", file)
		}

		// Ask for confirmation once
		fmt.Print("Do you want to delete all duplicates? (y/n): ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}

		if input != "y" && input != "Y" {
			fmt.Println("Skipping deletion.")
			return
		}

		// Proceed with deletion if confirmed
		if !dryRun {
			for _, file := range duplicatesToDelete {
				if err := os.Remove(file); err != nil {
					fmt.Printf("Error deleting file %s: %v\n", file, err)
				} else {
					fmt.Printf("Deleted: %s\n", file)
				}
				logData = append(logData, map[string]interface{}{"file": file, "deleted": true})
			}
		} else {
			fmt.Println("[Dry Run] The following files would be deleted:")
			for _, file := range duplicatesToDelete {
				fmt.Printf("  %s\n", file)
				logData = append(logData, map[string]interface{}{"file": file, "deleted": false})
			}
		}

		// Log results
		logger.WriteLog(logData, logFormat)
	} else {
		fmt.Println("No duplicates found.")
	}

	// Finish progress bar
	bar.Finish()
}
