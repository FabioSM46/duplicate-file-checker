package hashing

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// HashFile hashes a file using the specified hash method (md5, sha1, sha256, byte-by-byte)
// It returns the hash value as a string.
func HashFile(filePath string, hashMethod string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file %s: %v", filePath, err)
	}
	defer file.Close()

	var hashValue []byte
	switch hashMethod {
	case "md5":
		hash := md5.New()
		_, err := io.Copy(hash, file)
		if err != nil {
			return "", fmt.Errorf("error hashing file %s with MD5: %v", filePath, err)
		}
		hashValue = hash.Sum(nil)
	case "sha1":
		hash := sha1.New()
		_, err := io.Copy(hash, file)
		if err != nil {
			return "", fmt.Errorf("error hashing file %s with SHA1: %v", filePath, err)
		}
		hashValue = hash.Sum(nil)
	case "sha256":
		hash := sha256.New()
		_, err := io.Copy(hash, file)
		if err != nil {
			return "", fmt.Errorf("error hashing file %s with SHA256: %v", filePath, err)
		}
		hashValue = hash.Sum(nil)
	case "byte-by-byte":
		// For byte-by-byte, we can compare the file size and contents
		content, err := io.ReadAll(file)
		if err != nil {
			return "", fmt.Errorf("error reading file %s: %v", filePath, err)
		}
		return fmt.Sprintf("%x", content), nil
	default:
		return "", fmt.Errorf("unsupported hash method: %s", hashMethod)
	}

	// Return the computed hash as a string
	return fmt.Sprintf("%x", hashValue), nil
}
