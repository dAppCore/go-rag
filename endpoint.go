package rag

import (
	"net/url"
	"strconv"

	"dappco.re/go"
)

// parseEndpointURL normalises host-style endpoints into a parsed URL.
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

// parseEndpointPort converts and validates a TCP port parsed from an endpoint.
func parseEndpointPort(scope string, portText string) (int, error) {
	port, err := strconv.Atoi(portText)
	if err != nil {
		return 0, core.E(scope, core.Sprintf("invalid port: %s", portText), err)
	}
	if port < 1 || port > 65535 {
		return 0, core.E(scope, core.Sprintf("port out of range: %d", port), nil)
	}
	return port, nil
}
