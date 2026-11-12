package binance

import (
	"strings"
	"testing"
	"time"

	"newsbot/internal/exchange"
)

func TestParseStreamMessageFromDocs(t *testing.T) {
	data := `{\"catalogId\":161,\"catalogName\":\"Delisting\",\"publishDate\":1753257631403,` +
		`\"title\":\"Binance Will Delist ABC on 2025-08-01\",\"body\":\"This is...\"}`
	message := `{"type":"DATA","topic":"com_announcement_en","data":"` + data + `"}`

	a, ok, err := parseStreamMessage([]byte(message))
	if err != nil ||
		!ok ||
		a.Title != "Binance Will Delist ABC on 2025-08-01" ||
		a.Published.UnixMilli() != 1753257631403 ||
		!strings.HasSuffix(a.URL, "/list/161") {
		t.Fatalf("got %+v, %v", a, ok)
	}
}

func TestParseStreamMessageChecksSubscription(t *testing.T) {
	accepted := `{"type":"COMMAND","data":"SUCCESS","subType":"SUBSCRIBE","code":"00000000"}`
	rejected := `{"type":"COMMAND","data":"FAIL","subType":"SUBSCRIBE","code":"00000001"}`

	if _, ok, err := parseStreamMessage([]byte(accepted)); ok || err != nil {
		t.Fatal("subscription reply must be accepted silently")
	}

	if _, _, err := parseStreamMessage([]byte(rejected)); err == nil {
		t.Fatal("rejected subscription not reported")
	}
}

func TestSignedURLCarriesSignature(t *testing.T) {
	s := NewStream("key", "secret", exchange.AnnouncementPolicy{MaxAge: time.Minute})
	u := s.signedURL(time.UnixMilli(1753244327210))

	signature := u[strings.LastIndex(u, "=")+1:]

	if !strings.Contains(u, "topic=com_announcement_en&recvWindow=30000&timestamp=1753244327210&signature=") ||
		len(signature) != 64 {
		t.Fatalf("got %s", u)
	}
}
