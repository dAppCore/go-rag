package rag

import (
	"net/url"

	"dappco.re/go/core"
)

// parseEndpointURL normalizes host-style endpoints into a parsed URL.
// Bare host:port values are treated as HTTP URLs so callers can use either
// "localhost:11434" or "http://localhost:11434".
func parseEndpointURL(endpoint string) (*url.URL, error) {
	if endpoint == "" {
		return &url.URL{Scheme: "http"}, nil
	}
	if !core.Contains(endpoint, "://") {
		endpoint = core.Concat("http://", endpoint)
	}
	return url.Parse(endpoint)
}
