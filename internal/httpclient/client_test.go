package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestARefusalCarriesItsStatusAndTheWaitTheServerAsked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/busy" {
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(http.StatusTooManyRequests)

			return
		}

		if !strings.Contains(r.Header.Get("User-Agent"), "newsbot") {
			w.WriteHeader(http.StatusForbidden)

			return
		}

		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := New()

	var body struct {
		OK bool `json:"ok"`
	}

	if err := client.JSON(context.Background(), server.URL+"/fine", &body); err != nil || !body.OK {
		t.Fatalf("a good answer: %+v, %v", body, err)
	}

	_, err := client.Bytes(context.Background(), server.URL+"/busy")

	var refusal *StatusError

	if !errors.As(err, &refusal) || refusal.Code != http.StatusTooManyRequests || refusal.RetryAfter() != 2*time.Minute {
		t.Errorf("a refusal: %v", err)
	}
}

func TestAHostThatSaidTooManyRequestsIsLeftAloneForAWhile(t *testing.T) {
	asked := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asked++

		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer other.Close()

	client := New()

	_, first := client.Bytes(context.Background(), server.URL+"/a")
	_, second := client.Bytes(context.Background(), server.URL+"/b")

	var refused *StatusError
	var held *HeldError

	if !errors.As(first, &refused) || !errors.As(second, &held) || asked != 1 {
		t.Fatalf("the host was asked %d times: %v, %v", asked, first, second)
	}

	if time.Until(held.Until) < 50*time.Second {
		t.Errorf("the pause is too short: %v", held.Until)
	}

	if _, err := client.Bytes(context.Background(), other.URL); err != nil {
		t.Errorf("another host must not be held: %v", err)
	}
}
