package startup

import "regexp"

// uriPattern matches connection strings that may contain credentials
// (e.g. amqp://user:pass@host, mongodb://user:pass@host).
var uriPattern = regexp.MustCompile(`\w+://[^\s]+`)

// SanitizeError redacts connection strings and credential fragments
// from bootstrap errors before they are printed to stderr.
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}

	return uriPattern.ReplaceAllString(err.Error(), "[redacted-uri]")
}
