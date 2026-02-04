// Package main provides a CLI tool to derive the external HMAC key from a master key.
// This enables external consumers to verify document HMACs as described in
// docs/security/verification-guide.md.
//
// Usage:
//
//	go run scripts/crypto/derive-key/main.go -key "YOUR_BASE64_MASTER_KEY"
//
// Or using make:
//
//	make derive-key KEY="YOUR_BASE64_MASTER_KEY"
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/LerianStudio/fetcher/pkg/crypto"
)

func main() {
	// Define command-line flags
	keyFlag := flag.String("key", "", "Base64-encoded master key (APP_ENC_KEY)")
	helpFlag := flag.Bool("help", false, "Show usage information")
	flag.BoolVar(helpFlag, "h", false, "Show usage information (shorthand)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Derive External HMAC Key from Master Key\n\n")
		fmt.Fprintf(os.Stderr, "This tool derives the external HMAC key used for document signature\n")
		fmt.Fprintf(os.Stderr, "verification from your base64-encoded master key (APP_ENC_KEY).\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  go run scripts/crypto/derive-key/main.go -key \"YOUR_BASE64_MASTER_KEY\"\n")
		fmt.Fprintf(os.Stderr, "  make derive-key KEY=\"YOUR_BASE64_MASTER_KEY\"\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nOutput:\n")
		fmt.Fprintf(os.Stderr, "  The derived external HMAC key in hexadecimal format (64 characters).\n")
		fmt.Fprintf(os.Stderr, "  Use this key to verify HMAC signatures on decrypted documents.\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  $ go run scripts/crypto/derive-key/main.go -key \"dGhpcy1pcy1hLTMyLWJ5dGUtbWFzdGVyLWtleTEyMzQ=\"\n")
		fmt.Fprintf(os.Stderr, "  External HMAC Key (hex): <64-character-hex-string>\n\n")
		fmt.Fprintf(os.Stderr, "See docs/security/verification-guide.md for complete verification instructions.\n")
	}

	flag.Parse()

	// Show help if requested
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// Validate that key was provided
	if *keyFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: -key flag is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Decode the base64 master key
	masterKey, err := crypto.DecodeMasterKey(*keyFlag)
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
