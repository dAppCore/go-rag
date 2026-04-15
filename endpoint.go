package rag

import (
	"net/url"
	"strings"
)

// parseEndpointURL normalizes host-style endpoints into a parsed URL.
// Bare host:port values are treated as HTTP URLs so callers can use either
// "localhost:11434" or "http://localhost:11434".
func parseEndpointURL(endpoint string) (*url.URL, error) {
	if endpoint == "" {
		return &url.URL{Scheme: "http"}, nil
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}
	return url.Parse(endpoint)
}
