package googlenews

import "testing"

func TestHeadlinesAreReadFromTheFeed(t *testing.T) {
	body := []byte(`<?xml version="1.0"?><rss><channel><title>x</title>
<item><title>Initia halts blockchain after $615K exploit - Crypto News</title><link>https://news.google.com/rss/articles/1</link>
<pubDate>Tue, 10 Nov 2026 09:30:00 GMT</pubDate><source url="https://crypto.news">Crypto News</source></item>
<item><title>No outlet</title><link>https://example.org/2</link><pubDate>soon</pubDate></item>
</channel></rss>`)

	got, err := parse(body)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}

	first := got[0]

	if first.Title != "Initia halts blockchain after $615K exploit" || first.Outlet != "Crypto News" || first.Published != "2026-11-10T09:30:00Z" {
		t.Errorf("headline %+v", first)
	}

	if got[1].Published != "soon" || got[1].Title != "No outlet" {
		t.Errorf("headline %+v", got[1])
	}
}
