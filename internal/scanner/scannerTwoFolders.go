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

// ProcessTwoFolders compares files between the first directory and the second directory, hashes files, and identifies duplicates
func ProcessTwoFolders(dir1, dir2, hashMethod string, dryRun bool, logFormat string) {
	// Scan the first directory for files
	files1, err := ScanDirectory(dir1)
	if err != nil {
		fmt.Printf("Error scanning directory %s: %v\n", dir1, err)
		return
	}

	// Scan the second directory for files
	files2, err := ScanDirectory(dir2)
	if err != nil {
		fmt.Printf("Error scanning directory %s: %v\n", dir2, err)
		return
	}

	// Set up parallel processing
	var wg sync.WaitGroup
	resultChan := make(chan map[string]string, len(files2))

	// Create progress bar for the second directory
	bar := progressbar.NewOptions(len(files2), progressbar.OptionSetDescription("Hashing files in second folder"), progressbar.OptionSetWidth(50))

	// Start hashing files from the second directory in parallel
	for _, filePath := range files2 {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			// Calculate the hash for each file in the second directory
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

	// Now check for each file in the first directory and compare with the second directory
	for _, file1 := range files1 {
		// Calculate the hash for each file in the first directory
		hashValue, err := hashing.HashFile(file1, hashMethod)
		if err != nil {
			fmt.Printf("Error hashing file %s: %v\n", file1, err)
			continue
		}

		// Check if any file in the second directory has the same hash
		if filesWithSameHash, found := duplicates[hashValue]; found {
			// If a duplicate is found, it means there is a file in the second directory that matches
			fmt.Printf("Duplicate file found in second directory for file: %s\n", file1)
			for _, file2 := range filesWithSameHash {
				// Skip the first file, as it's considered the original (the one we're comparing)
				if file2 == file1 {
					continue
				}

				fmt.Printf("  Duplicate: %s\n", file2)

				// Prompt for file deletion from the second directory if not in dry-run mode
				if !dryRun {
					// Ask the user for action: y = delete, n = skip, all = delete all except the current one
					fmt.Printf("Do you want to delete %s from the second folder? (y/n/all): ", file2)
					var input string
					fmt.Scanln(&input)
					input = strings.ToLower(input)
					deleted := false

					// Handle different responses: delete, skip, or delete all duplicates
					switch input {
					case "y":
						err := os.Remove(file2)
						if err != nil {
							fmt.Printf("Error deleting file %s: %v\n", file2, err)
						} else {
							fmt.Printf("File %s deleted from the second folder.\n", file2)
							deleted = true
						}
					case "n":
						fmt.Printf("File %s kept in the second folder.\n", file2)
					case "all":
						// Delete all files with this hash except the first one (file1)
						for _, deleteFile := range filesWithSameHash {
							if deleteFile == file1 {
								continue // Skip the first file, which is considered the original
							}

							err := os.Remove(deleteFile)
							if err != nil {
								fmt.Printf("Error deleting file %s: %v\n", deleteFile, err)
							} else {
								fmt.Printf("File %s deleted from the second folder.\n", deleteFile)
							}
						}
						deleted = true
					default:
						fmt.Println("Invalid option. Skipping deletion.")
					}

					// Log the deletion result with the hash included
					logData = append(logData, map[string]interface{}{
						"file":    file2,
						"deleted": deleted,
						"hash":    hashValue, // Add the hash here
					})
				} else {
					// Dry run: Display the files that would be deleted
					fmt.Println("[Dry Run] The following files would be deleted from the second folder:")
					fmt.Printf("  %s\n", file2)

					// Log the dry run result with deleted set to false and hash included
					logData = append(logData, map[string]interface{}{
						"file":    file2,
						"deleted": false,
						"hash":    hashValue, // Add the hash here
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
