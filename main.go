package main

import (
	"duplicate-file-checker/internal/scanner"
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define flags with descriptions
	dryRun := flag.Bool("dry-run", false, "Do not delete files, just show which would be deleted")
	hashMethod := flag.String("hash", "sha256", "Hash method (md5, sha1, sha256, or bbb for byte-by-byte comparison)")
	logFormat := flag.String("log", "json", "Log format (json, csv, plain)")
	flag.Parse()

	maxWorkers := 4

	// Directory paths from command line arguments
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Error: You must provide at least one directory to scan.")
		os.Exit(1)
	}
	if len(args) == 1 {
		// Single folder mode
		fmt.Printf("Scanning single folder: %s\n", args[0])
		scanner.ProcessSingleFolder(args[0], *hashMethod, *dryRun, *logFormat, maxWorkers)
	} else if len(args) == 2 {
		// Two folder mode
		fmt.Printf("Comparing files between directories: %s and %s\n", args[0], args[1])
		scanner.ProcessTwoFolders(args[0], args[1], *hashMethod, *dryRun, *logFormat, maxWorkers)
	} else {
		fmt.Println("Error: Invalid number of arguments. Please provide one or two directory paths.")
		os.Exit(1)
	}
}
