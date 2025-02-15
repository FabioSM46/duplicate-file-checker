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

// ProcessTwoFolders checks for duplicate files between two directories
func ProcessTwoFolders(folder1, folder2, hashMethod string, dryRun bool, logFormat string) {
	// Scan both directories for files
	files1, err := ScanDirectory(folder1)
	if err != nil {
		fmt.Printf("Error scanning directory %s: %v\n", folder1, err)
		return
	}
	files2, err := ScanDirectory(folder2)
	if err != nil {
		fmt.Printf("Error scanning directory %s: %v\n", folder2, err)
		return
	}

	// Set up parallel processing
	var wg sync.WaitGroup
	var mu sync.Mutex // Mutex to safely write to maps in goroutines

	// Create progress bar
	bar := progressbar.NewOptions(len(files1)+len(files2), progressbar.OptionSetDescription("Hashing files"), progressbar.OptionSetWidth(50))

	// Hash maps to store hash -> file paths
	hashMap1 := make(map[string][]string) // Now stores multiple file paths per hash
	hashMap2 := make(map[string][]string) // Now stores multiple file paths per hash

	// Function to hash files and store results in the appropriate map
	hashFiles := func(files []string, hashMap map[string][]string) {
		for _, filePath := range files {
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				hashValue, err := hashing.HashFile(path, hashMethod)
				if err != nil {
					fmt.Printf("Error hashing file %s: %v\n", path, err)
					return
				}

				mu.Lock()
				hashMap[hashValue] = append(hashMap[hashValue], path) // Store all file paths for the same hash
				mu.Unlock()

				bar.Add(1)
			}(filePath)
		}
	}

	// Start hashing for both folders
	hashFiles(files1, hashMap1)
	hashFiles(files2, hashMap2)

	// Wait for all hashing goroutines to complete
	wg.Wait()

	// Prepare a slice to collect log data
	logData := []map[string]interface{}{}

	// Ask if the user wants to delete all duplicates
	var deleteAllOption string
	fmt.Printf("Do you want to delete all duplicates? (y/n): ")
	fmt.Scanln(&deleteAllOption)
	deleteAllOption = strings.ToLower(deleteAllOption)

	// Keep track of deleted files to avoid double deletion
	deletedFiles := make(map[string]bool)

	// Handle duplicate detection and deletion if the user agrees
	if deleteAllOption == "y" {
		// Process duplicates in hashMap1 (first folder)
		for hash, files1 := range hashMap1 {
			if files2, exists := hashMap2[hash]; exists {
				// Process each file in the first folder (files1) and check for duplicates in the second folder (files2)
				for _, file1 := range files1 {
					for _, file2 := range files2 {
						// Only proceed if the duplicate from the second folder hasn't already been deleted
						if _, alreadyDeleted := deletedFiles[file2]; !alreadyDeleted {
							fmt.Printf("Duplicate found:\n  %s\n  %s\n", file1, file2)

							if !dryRun {
								// Delete the duplicate (file2) from the second folder
								if err := os.Remove(file2); err == nil {
									fmt.Printf("Deleted: %s\n", file2)
									logData = append(logData, map[string]interface{}{"file": file2, "deleted": true, "hash": hash})
									deletedFiles[file2] = true // Mark this file as deleted
								} else {
									fmt.Printf("Error deleting %s: %v\n", file2, err)
								}
							} else {
								// Just log the duplicate files without deleting them in dry-run mode
								logData = append(logData, map[string]interface{}{"file1": file1, "file2": file2, "deleted": false, "hash": hash})
							}
						}
					}
				}
			}
		}
	} else {
		fmt.Println("Skipping deletion of duplicates.")
	}

	// Log results
	logger.WriteLog(logData, logFormat)

	// Finish progress bar
	bar.Finish()
}
