package rag

import (
	"net/url"
	"strconv"

	"dappco.re/go"
)

// parseEndpointURL normalises host-style endpoints into a parsed URL.
// Bare host:port values are treated as HTTP URLs so callers can use either
// "localhost:11434" or "http://localhost:11434".
func parseEndpointURL(endpoint string) core.Result {
	if endpoint == "" {
		return core.Ok(&url.URL{Scheme: "http"})
	}
	if !core.Contains(endpoint, "://") {
		endpoint = core.Concat("http://", endpoint)
	}
	return core.ResultOf(url.Parse(endpoint))
}

// parseEndpointPort converts and validates a TCP port parsed from an endpoint.
func parseEndpointPort(scope string, portText string) core.Result {
	port, err := strconv.Atoi(portText)
	if err != nil {
		return core.Fail(core.E(scope, core.Sprintf("invalid port: %s", portText), err))
	}
	if port < 1 || port > 65535 {
		return core.Fail(core.E(scope, core.Sprintf("port out of range: %d", port), nil))
	}
	return core.Ok(port)
}
