package scanner

import (
	"duplicate-file-checker/internal/hashing"
	"duplicate-file-checker/internal/logger"
	"fmt"
	"os"
	"sync"

	"github.com/schollz/progressbar/v3"
)

func ProcessTwoFolders(dir1, dir2, hashMethod string, dryRun bool, logFormat string, maxWorkers int) {
	// Scan both directories for files
	files1, err := ScanDirectory(dir1)
	if err != nil {
		fmt.Printf("Error scanning directory %s: %v\n", dir1, err)
		return
	}
	files2, err := ScanDirectory(dir2)
	if err != nil {
		fmt.Printf("Error scanning directory %s: %v\n", dir2, err)
		return
	}

	allFiles := append(files1, files2...)
	resultChan := make(chan map[string]string, len(allFiles))

	var wg sync.WaitGroup
	workerPool := make(chan struct{}, maxWorkers)
	bar := progressbar.NewOptions(len(allFiles), progressbar.OptionSetDescription("Hashing files"), progressbar.OptionSetWidth(50))

	// Hash files in parallel
	for _, filePath := range allFiles {
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

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	hashMap := make(map[string][]string)
	for result := range resultChan {
		for hash, filePath := range result {
			hashMap[hash] = append(hashMap[hash], filePath)
		}
	}

	// Identify duplicates between folders
	var duplicatesToDelete []string
	for _, files := range hashMap {
		var inDir1, inDir2 []string
		for _, file := range files {
			if contains(files1, file) {
				inDir1 = append(inDir1, file)
			} else if contains(files2, file) {
				inDir2 = append(inDir2, file)
			}
		}
		if len(inDir1) > 0 && len(inDir2) > 0 {
			duplicatesToDelete = append(duplicatesToDelete, inDir2...) // Keep files from dir1, delete from dir2
		}
	}

	logData := []map[string]interface{}{}

	if len(duplicatesToDelete) > 0 {
		fmt.Println("Duplicates found in second folder (these files will be deleted):")
		for _, file := range duplicatesToDelete {
			fmt.Printf("  %s\n", file)
		}

		fmt.Print("Do you want to delete all duplicates in the second folder? (y/n): ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			return
		}

		if input != "y" && input != "Y" {
			fmt.Println("Skipping deletion.")
			// Log the found duplicates even if they are not deleted
			for _, file := range duplicatesToDelete {
				logData = append(logData, map[string]interface{}{"file": file, "deleted": false})
			}
			logger.WriteLog(logData, logFormat) // Ensure logs are always written
			return
		}

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

		logger.WriteLog(logData, logFormat)
	} else {
		fmt.Println("No duplicates found between the two folders.")
	}

	bar.Clear()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
