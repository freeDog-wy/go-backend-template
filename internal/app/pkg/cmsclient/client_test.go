package cmsclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestGetBuildsRequestAndAppliesOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gateway/api/v1/public/locales" || r.URL.Query().Get("page") != "2" {
			t.Fatalf("request URL = %s", r.URL.String())
		}
		if r.Header.Get("X-Request-ID") != "request-1" || r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("headers = %#v", r.Header)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":[]}`))
	}))
	defer server.Close()

	client, err := New(server.URL+"/gateway", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	body, err := client.Get(context.Background(), "/api/v1/public/locales", url.Values{"page": {"2"}}, RequestOptions{
		Headers: http.Header{"X-Request-ID": {"request-1"}},
		Authorizer: AuthorizerFunc(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer token")
			return nil
		}),
	})
	if err != nil || string(body) != `{"success":true,"data":[]}` {
		t.Fatalf("Get() = %q, %v", body, err)
	}
}

func TestDecodeEnvelope(t *testing.T) {
	envelope, err := DecodeEnvelope([]byte(`{"success":true,"data":{"id":7},"meta":{"page":1}}`))
	if err != nil || string(envelope.Data) != `{"id":7}` || string(envelope.Meta) != `{"page":1}` {
		t.Fatalf("DecodeEnvelope() = %#v, %v", envelope, err)
	}
	_, err = DecodeEnvelope([]byte(`{"success":false,"error":{"code":"DENIED","message":"denied"}}`))
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "DENIED" {
		t.Fatalf("DecodeEnvelope() error = %#v", err)
	}
}
