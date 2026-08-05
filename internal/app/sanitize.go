package app

import "strings"

func RedactAPIKey(data string, apiKey string) string {
	if apiKey == "" {
		return data
	}
	return strings.ReplaceAll(data, apiKey, "[REDACTED]")
}