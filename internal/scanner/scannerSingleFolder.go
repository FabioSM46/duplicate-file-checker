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
	workerPool := make(chan struct{}, maxWorkers) // Create a pool of maxWorkers workers

	// Create progress bar
	bar := progressbar.NewOptions(len(files), progressbar.OptionSetDescription("Hashing files"), progressbar.OptionSetWidth(50))

	// Start workers for parallel file hashing
	for _, filePath := range files {
		wg.Add(1)
		workerPool <- struct{}{} // Block if we have reached maxWorkers

		go func(path string) {
			defer wg.Done()
			defer func() { <-workerPool }() // Release the worker after completion

			// Calculate the hash for each file and send it to the result channel
			hashValue, err := hashing.HashFile(path, hashMethod)
			if err != nil {
				fmt.Printf("Error hashing file %s: %v\n", path, err)
				return
			}

			// Send the hash value along with the file path to the result channel
			resultChan <- map[string]string{hashValue: path}

			// Increment progress bar after hashing each file
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
		// Each result contains a hash and the corresponding file path
		for hash, filePath := range result {
			duplicates[hash] = append(duplicates[hash], filePath)
		}
	}

	// Prepare a slice to collect log data
	logData := []map[string]interface{}{}

	// List to hold duplicates that are going to be deleted
	var duplicatesToDelete []string

	// Flag to check if user confirmed deletion
	var confirmDeletion bool

	// Flag to check if any duplicates were found
	var foundDuplicates bool

	// Collect duplicates (excluding the first file, the original one)
	for _, files := range duplicates {
		if len(files) > 1 {
			// Append duplicates (excluding the first one) to the deletion list
			duplicatesToDelete = append(duplicatesToDelete, files[1:]...)
		}
	}

	// Display only the duplicates that will be deleted
	if len(duplicatesToDelete) > 0 {
		fmt.Println("Duplicates found (these files will be deleted):")
		for _, file := range duplicatesToDelete {
			fmt.Printf("  %s\n", file)
		}
	}

	// Ask for confirmation only once, after displaying duplicates
	if !foundDuplicates {
		fmt.Print("Do you want to delete all duplicates? (y/n): ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}
		if input == "y" || input == "Y" {
			confirmDeletion = true
		} else if input == "n" || input == "N" {
			confirmDeletion = false
		} else {
			fmt.Println("Invalid input, skipping deletion.")
			confirmDeletion = false
		}
		// Set the flag so we don't ask again
		foundDuplicates = true
	}

	// If confirmed, proceed with deletion (except for the first one)
	for hash, files := range duplicates {
		if len(files) > 1 {
			if confirmDeletion {
				if !dryRun {
					// Delete all duplicates except the first one (keep one copy)
					for _, file := range files[1:] {
						if err := os.Remove(file); err != nil {
							fmt.Printf("Error deleting file %s: %v\n", file, err)
						} else {
							fmt.Printf("File %s deleted.\n", file)
						}

						// Log deletion with the hash value
						logData = append(logData, map[string]interface{}{
							"file":    file,
							"deleted": true,
							"hash":    hash, // Using hash here for logging
						})
					}
				} else {
					// Dry run: Display the files that would be deleted
					fmt.Println("[Dry Run] The following files would be deleted:")
					for _, file := range files[1:] {
						fmt.Printf("  %s\n", file)
						logData = append(logData, map[string]interface{}{
							"file":    file,
							"deleted": false,
							"hash":    hash, // Using hash here for logging
						})
					}
				}
			}
		}
	}

	// Log the results in the desired format (e.g., JSON, CSV)
	logger.WriteLog(logData, logFormat)

	// Wait for progress bar to complete before exiting
	bar.Finish()

}
