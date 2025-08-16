package links

import "net/http"

// No tests remaining; all imports removed

type badRoundTripper struct{}

func (b *badRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: 200,
		Body:       &badBody{},
	}
	return resp, nil
}

type badBody struct{}

func (b *badBody) Read(p []byte) (int, error) {
	copy(p, []byte("not-json"))
	return len("not-json"), nil
}
func (b *badBody) Close() error { return nil }
