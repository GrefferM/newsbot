package telegram

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"newsbot/internal/monitor"
	"newsbot/internal/news"
)

type recorded struct {
	path    string
	caption string
	photo   int
	text    string
}

func fakeTelegram(t *testing.T) (*httptest.Server, *[]recorded) {
	t.Helper()

	var mu sync.Mutex
	var calls []recorded

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := recorded{path: r.URL.Path}

		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
			_ = r.ParseMultipartForm(1 << 20)
			rec.caption = r.FormValue("caption")

			if f, _, err := r.FormFile("photo"); err == nil {
				data, _ := io.ReadAll(f)
				rec.photo = len(data)
			}
		} else {
			body, _ := io.ReadAll(r.Body)
			rec.text = string(body)
		}

		mu.Lock()
		calls = append(calls, rec)
		mu.Unlock()

		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":7}}`))
	}))

	return server, &calls
}

func publisherFor(server *httptest.Server) *Publisher {
	p := NewPublisher("TOKEN", "42")
	p.apiBase = server.URL

	return p
}

func TestPublishSendsChartWithCaption(t *testing.T) {
	server, calls := fakeTelegram(t)
	defer server.Close()

	post := monitor.Post{
		Event: news.Event{Category: news.ShortTrigger, Kind: "Різке падіння", Title: "ABC"},
		Chart: []byte("png-bytes"),
	}

	if err := publisherFor(server).Publish(context.Background(), post); err != nil {
		t.Fatal(err)
	}

	got := *calls

	if len(got) != 1 ||
		got[0].path != "/botTOKEN/sendPhoto" ||
		got[0].photo != 9 ||
		!strings.Contains(got[0].caption, "Різке падіння") {
		t.Fatalf("got %+v", got)
	}
}

func TestPublishSplitsLongCaption(t *testing.T) {
	server, calls := fakeTelegram(t)
	defer server.Close()

	post := monitor.Post{
		Event: news.Event{Category: news.Incident, Title: strings.Repeat("x", 1500)},
		Chart: []byte("png"),
	}

	if err := publisherFor(server).Publish(context.Background(), post); err != nil {
		t.Fatal(err)
	}

	got := *calls

	if len(got) != 2 ||
		got[0].path != "/botTOKEN/sendPhoto" ||
		got[0].caption != "" ||
		got[1].path != "/botTOKEN/sendMessage" ||
		!strings.Contains(got[1].text, `"message_id":7`) {
		t.Fatalf("got %+v", got)
	}
}

func TestPublishWithoutChartSendsText(t *testing.T) {
	server, calls := fakeTelegram(t)
	defer server.Close()

	post := monitor.Post{Event: news.Event{Title: "plain"}}

	if err := publisherFor(server).Publish(context.Background(), post); err != nil {
		t.Fatal(err)
	}

	if len(*calls) != 1 || (*calls)[0].path != "/botTOKEN/sendMessage" || !strings.Contains((*calls)[0].text, "plain") {
		t.Fatalf("got %+v", *calls)
	}
}
