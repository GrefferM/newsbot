package tavily

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchSendsTheQueryWithTheKeyAndReadsResults(t *testing.T) {
	var (
		asked *http.Request
		body  []byte
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r
		body, _ = io.ReadAll(r.Body)

		_, _ = w.Write([]byte(`{"results":[{"title":"Blur closes","url":"https://example.org/a","content":"The site shuts down","score":0.9,"published_date":"Tue, 10 Nov 2026 09:00:00 GMT"}]}`))
	}))
	defer server.Close()

	client := Client{http: server.Client(), url: server.URL, key: "k"}

	got, err := client.Search(context.Background(), Query{Text: "blur news", Period: "week", News: true})
	if err != nil || len(got) != 1 || got[0].Excerpt != "The site shuts down" || got[0].Published == "" {
		t.Fatalf("got %+v, %v", got, err)
	}

	sent := `{"query":"blur news","search_depth":"basic","max_results":10,"include_published_date":true,"time_range":"week","topic":"news"}`

	if asked.Method != http.MethodPost || asked.Header.Get("Authorization") != "Bearer k" || string(body) != sent {
		t.Errorf("asked %s with %q: %s", asked.Method, asked.Header.Get("Authorization"), body)
	}
}

func TestARefusalIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := Client{http: server.Client(), url: server.URL, key: "k"}

	if _, err := client.Search(context.Background(), Query{Text: "x"}); err == nil {
		t.Error("a refusal passed as a result")
	}
}
