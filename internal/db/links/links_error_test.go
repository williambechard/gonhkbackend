package links

import (
	"errors"
	"net/http"
)

// No tests remaining; all imports removed

type errorRoundTripper struct{}

func (e *errorRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("forced error")
}
