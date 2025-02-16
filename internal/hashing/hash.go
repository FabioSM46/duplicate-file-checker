package hashing

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
)

// HashFile hashes a file using the specified method (md5, sha1, sha256, bbb).
// Uses a buffer to handle large files efficiently.
func HashFile(filePath string, hashMethod string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file %s: %v", filePath, err)
	}
	defer file.Close()

	// Special case for "bbb" (byte-by-byte comparison)
	if hashMethod == "bbb" {
		return generateBBBHash(file)
	}

	// Choose hashing algorithm
	var hasher hash.Hash
	switch hashMethod {
	case "md5":
		hasher = md5.New()
	case "sha1":
		hasher = sha1.New()
	case "sha256":
		hasher = sha256.New()
	default:
		return "", fmt.Errorf("unsupported hash method: %s", hashMethod)
	}

	// Use a 1MB buffer to process large files efficiently
	buffer := make([]byte, 4*1024*1024) // 4MB
	_, err = io.CopyBuffer(hasher, file, buffer)
	if err != nil {
		return "", fmt.Errorf("error hashing file %s with %s: %v", filePath, hashMethod, err)
	}

	// Return computed hash as string
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// generateBBBHash performs an efficient byte-by-byte hash with buffering.
func generateBBBHash(file *os.File) (string, error) {
	const bufferSize = 1024 * 1024 // 1MB buffer

	// Rewind file to beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("error seeking file: %v", err)
	}

	hasher := md5.New() // Fast identifier for byte-by-byte comparison
	reader := bufio.NewReaderSize(file, bufferSize)

	buffer := make([]byte, bufferSize)

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			_, _ = hasher.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error reading file: %v", err)
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
