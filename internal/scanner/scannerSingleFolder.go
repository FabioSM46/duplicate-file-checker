package scanner

import (
	"duplicate-file-checker/internal/hashing"
	"duplicate-file-checker/internal/logger"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
)

// ProcessSingleFolder scans the directory, hashes files, and identifies duplicates
func ProcessSingleFolder(dirPath, hashMethod string, dryRun bool, logFormat string) {
	// Scan directory for files
	files, err := ScanDirectory(dirPath)
	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		return
	}

	// Set up parallel processing
	var wg sync.WaitGroup
	resultChan := make(chan map[string]string, len(files))

	// Create progress bar
	bar := progressbar.NewOptions(len(files), progressbar.OptionSetDescription("Hashing files"), progressbar.OptionSetWidth(50))

	// Start hashing files in parallel
	for _, filePath := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
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

	// Prepare a slice to collect log data with the `deleted` and `hash` keys
	logData := []map[string]interface{}{}

	// Handle file deletion and log the results
	for hash, files := range duplicates {
		if len(files) > 1 {
			// If more than one file with the same hash, consider it a duplicate
			fmt.Printf("Duplicate files found for hash: %s\n", hash)
			for _, file := range files {
				fmt.Printf("  %s\n", file)
			}

			// Prompt for file deletion if not in dry-run mode
			if !dryRun {
				// Ask user if they want to delete all or handle files individually
				var deleteOption string
				fmt.Printf("Do you want to delete these duplicates? (y/n/all): ")
				fmt.Scanln(&deleteOption)
				deleteOption = strings.ToLower(deleteOption)

				// Determine which option the user chose
				if deleteOption == "all" {
					// Automatically delete all duplicates except the first one
					for _, file := range files[1:] { // Skip the first file (original)
						err := os.Remove(file)
						if err != nil {
							fmt.Printf("Error deleting file %s: %v\n", file, err)
						} else {
							fmt.Printf("File %s deleted.\n", file)
						}

						// Log the deletion result with the hash included
						logData = append(logData, map[string]interface{}{
							"file":    file,
							"deleted": true,
							"hash":    hash,
						})
					}
				} else {
					// Handle individual deletions based on user input
					for _, file := range files[1:] { // Skip the first file (original)
						fmt.Printf("Do you want to delete %s? (y/n): ", file)
						var input string
						fmt.Scanln(&input)
						deleted := false
						if strings.ToLower(input) == "y" {
							err := os.Remove(file)
							if err != nil {
								fmt.Printf("Error deleting file %s: %v\n", file, err)
							} else {
								fmt.Printf("File %s deleted.\n", file)
								deleted = true
							}
						} else {
							fmt.Printf("File %s kept.\n", file)
						}

						// Log the deletion result with the hash included
						logData = append(logData, map[string]interface{}{
							"file":    file,
							"deleted": deleted,
							"hash":    hash, // Add the hash here
						})
					}
				}
			} else {
				// Dry run: Display the files that would be deleted
				fmt.Println("[Dry Run] The following files would be deleted:")
				for _, file := range files[1:] { // Skip the first file (original)
					fmt.Printf("  %s\n", file)

					// Log the dry run result with deleted set to false and hash included
					logData = append(logData, map[string]interface{}{
						"file":    file,
						"deleted": false,
						"hash":    hash, // Add the hash here
					})
				}
			}
		}
	}

	// Log the results in the desired format (e.g., JSON, CSV)
	logger.WriteLog(logData, logFormat)

	// Wait for progress bar to complete before exiting
	bar.Finish()
}
