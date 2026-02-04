// Package main provides a CLI tool to derive the external HMAC key from a master key.
// This enables external consumers to verify document HMACs as described in
// docs/security/verification-guide.md.
//
// Usage (in order of precedence):
//
//  1. Environment variable: APP_ENC_KEY="YOUR_BASE64_MASTER_KEY" go run scripts/crypto/derive-key/main.go
//  2. Stdin: echo "YOUR_BASE64_MASTER_KEY" | go run scripts/crypto/derive-key/main.go
//  3. CLI flag (NOT RECOMMENDED - exposes key in process list): go run scripts/crypto/derive-key/main.go -key "YOUR_BASE64_MASTER_KEY"
//
// Or using make:
//
//	APP_ENC_KEY="YOUR_BASE64_MASTER_KEY" make derive-key
package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/crypto"
)

func main() {
	// Define command-line flags
	keyFlag := flag.String("key", "", "Base64-encoded master key (NOT RECOMMENDED - use APP_ENC_KEY env var or stdin instead)")
	helpFlag := flag.Bool("help", false, "Show usage information")
	flag.BoolVar(helpFlag, "h", false, "Show usage information (shorthand)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Derive External HMAC Key from Master Key\n\n")
		fmt.Fprintf(os.Stderr, "This tool derives the external HMAC key used for document signature\n")
		fmt.Fprintf(os.Stderr, "verification from your base64-encoded master key (APP_ENC_KEY).\n\n")
		fmt.Fprintf(os.Stderr, "Usage (in order of precedence):\n")
		fmt.Fprintf(os.Stderr, "  1. Environment variable (RECOMMENDED):\n")
		fmt.Fprintf(os.Stderr, "     APP_ENC_KEY=\"YOUR_BASE64_MASTER_KEY\" go run scripts/crypto/derive-key/main.go\n\n")
		fmt.Fprintf(os.Stderr, "  2. Stdin:\n")
		fmt.Fprintf(os.Stderr, "     echo \"YOUR_BASE64_MASTER_KEY\" | go run scripts/crypto/derive-key/main.go\n\n")
		fmt.Fprintf(os.Stderr, "  3. CLI flag (NOT RECOMMENDED - exposes key in process list):\n")
		fmt.Fprintf(os.Stderr, "     go run scripts/crypto/derive-key/main.go -key \"YOUR_BASE64_MASTER_KEY\"\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nOutput:\n")
		fmt.Fprintf(os.Stderr, "  The derived external HMAC key in hexadecimal format (64 characters).\n")
		fmt.Fprintf(os.Stderr, "  Use this key to verify HMAC signatures on decrypted documents.\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  $ APP_ENC_KEY=\"dGhpcy1pcy1hLTMyLWJ5dGUtbWFzdGVyLWtleTEyMzQ=\" go run scripts/crypto/derive-key/main.go\n")
		fmt.Fprintf(os.Stderr, "  External HMAC Key (hex): <64-character-hex-string>\n\n")
		fmt.Fprintf(os.Stderr, "See docs/security/verification-guide.md for complete verification instructions.\n")
	}

	flag.Parse()

	// Show help if requested
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// Get the key from various sources (in order of precedence)
	keyBase64 := getKeyFromSources(*keyFlag)

	if keyBase64 == "" {
		fmt.Fprintf(os.Stderr, "Error: No master key provided\n\n")
		fmt.Fprintf(os.Stderr, "Provide the key via:\n")
		fmt.Fprintf(os.Stderr, "  - APP_ENC_KEY environment variable (recommended)\n")
		fmt.Fprintf(os.Stderr, "  - stdin (pipe or redirect)\n")
		fmt.Fprintf(os.Stderr, "  - -key flag (not recommended)\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Decode the base64 master key
	masterKey, err := crypto.DecodeMasterKey(keyBase64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nHint: Ensure your key is valid base64 and at least 32 bytes when decoded.\n")
		os.Exit(1)
	}

	// Create the key deriver
	deriver, err := crypto.NewHKDFKeyDeriver(masterKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get the external HMAC key and encode as hex
	externalHMACKey := deriver.GetExternalHMACKey()
	hexKey := hex.EncodeToString(externalHMACKey)

	// Output the derived key
	fmt.Printf("External HMAC Key (hex): %s\n", hexKey)
}

// getKeyFromSources returns the master key from the first available source:
// 1. APP_ENC_KEY environment variable
// 2. stdin (if data is available)
// 3. -key CLI flag (fallback, not recommended)
func getKeyFromSources(keyFlag string) string {
	// 1. Try environment variable first (most secure)
	if envKey := os.Getenv("APP_ENC_KEY"); envKey != "" {
		return strings.TrimSpace(envKey)
	}

	// 2. Try reading from stdin if data is available
	if stdinKey := readFromStdin(); stdinKey != "" {
		return stdinKey
	}

	// 3. Fall back to CLI flag (least secure - exposes in process list)
	return strings.TrimSpace(keyFlag)
}

// readFromStdin attempts to read the key from stdin if data is available.
// Returns empty string if stdin is a terminal or no data is available.
func readFromStdin() string {
	// Check if stdin has data (is not a terminal)
	stat, err := os.Stdin.Stat()
	if err != nil {
		return ""
	}

	// Check if stdin is a pipe or has data
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// stdin is a terminal, not a pipe - don't block waiting for input
		return ""
	}

	// Read from stdin
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return ""
	}

	return strings.TrimSpace(line)
}
