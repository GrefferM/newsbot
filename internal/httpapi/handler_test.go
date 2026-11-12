package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"newsbot/internal/agenda"
	"newsbot/internal/autotrade"
	"newsbot/internal/market"
	"newsbot/internal/monitor"
	"newsbot/internal/news"
	"newsbot/internal/research"
	"newsbot/internal/watchlist"
)

type memoryArchive struct{}

func (memoryArchive) Save(research.Archive) error {
	return nil
}

func serve(t *testing.T, h http.Handler, method, path, body string) (int, string) {
	t.Helper()

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))

	return rec.Code, rec.Body.String()
}

func TestHealthNewsAndPlanAreServed(t *testing.T) {
	keeper := research.NewKeeper(memoryArchive{}, research.Archive{})

	if err := keeper.Keep(research.Finding{ID: "n1", Asset: "BLUR", Published: "2026-10-12", Text: "Сайт закривається", Source: "x"}); err != nil {
		t.Fatal(err)
	}

	vote := research.Finding{ID: "n2", Asset: "ZEC", Day: time.Now().AddDate(0, 0, 2).Format(time.DateOnly), Text: "Голосування", Source: "y"}

	if err := keeper.Keep(vote); err != nil {
		t.Fatal(err)
	}

	unlock := research.Finding{ID: "n3", Asset: "AGT", Day: vote.Day, Hour: "20:43", Text: "Розблокування", Source: "z"}

	if err := keeper.Keep(unlock); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(HandlerDeps{Status: monitor.NewStatus(time.Now()), Recent: monitor.NewRecent(5, nil, nil), Research: keeper, Agenda: agenda.NewService(agenda.Deps{Coins: keeper})})
	plan := `{"summary":"Слабка","scenarios":[{"condition":"Делістинг","action":"шорт","risk":"мінімальний","why":"рух відомий"}],"made":"2026-11-10T10:00:00Z"}`

	cases := []struct {
		method, path, body string
		code               int
		has                string
	}{
		{"GET", "/healthz", "", 200, `"started"`},
		{"GET", "/events", "", 200, "[]"},
		{"GET", "/calendar", "", 200, `"all_day": true`},
		{"GET", "/calendar", "", 200, `"source": "ZEC"`},
		{"GET", "/calendar", "", 200, vote.Day + `T20:43:00Z"`},
		{"DELETE", "/events?id=none", "", 409, "такої події немає"},
		{"GET", "/coins/blur/news", "", 200, "Сайт закривається"},
		{"GET", "/coins/BLUR/plan", "", 404, "плану ще немає"},
		{"PUT", "/coins/BLUR/plan", plan, 200, `"ok": true`},
		{"GET", "/coins/BLUR/plan", "", 200, "Делістинг"},
		{"DELETE", "/news/n1", "", 200, `"ok": true`},
		{"DELETE", "/news/n1", "", 409, "такої новини немає"},
	}

	for _, c := range cases {
		code, body := serve(t, h, c.method, c.path, c.body)

		if code != c.code || !strings.Contains(body, c.has) {
			t.Errorf("%s %s: %d %s", c.method, c.path, code, body)
		}
	}
}

type autoFile struct {
	saved []autotrade.State
}

func (f *autoFile) Save(state autotrade.State) error {
	f.saved = append(f.saved, state)

	return nil
}

func TestAutoTradingIsSwitchedOverTheAPI(t *testing.T) {
	file := &autoFile{}
	h := NewHandler(HandlerDeps{Auto: autotrade.NewService(autotrade.Deps{Store: file}, autotrade.State{})})

	cases := []struct {
		method, body string
		has          string
	}{
		{"GET", "", `"enabled": false`},
		{"PUT", `{"enabled":true}`, `"ok": true`},
		{"GET", "", `"enabled": true`},
	}

	for _, c := range cases {
		code, body := serve(t, h, c.method, "/auto", c.body)

		if code != 200 || !strings.Contains(body, c.has) {
			t.Errorf("%s /auto %s: %d %s", c.method, c.body, code, body)
		}
	}

	if len(file.saved) != 1 || !file.saved[0].Enabled {
		t.Errorf("the switch is saved: %+v", file.saved)
	}
}

type livePrices map[string]float64

func (l livePrices) Ticker(_ context.Context, s news.Symbol) (market.Ticker, bool, error) {
	price, listed := l[string(s)]

	return market.Ticker{Price: price}, listed, nil
}

func TestTheCoinsCarryThePerpetualsLivePriceOverTheScans(t *testing.T) {
	scan := watchlist.Scan{Coins: map[string]watchlist.Coin{
		"BTC": {Asset: "BTC", Contract: "BTCUSDT", Price: 71423.3, Volume: 1, HasChanges: true},
		"OLD": {Asset: "OLD", Contract: "OLDUSDT", Price: 2, Volume: 1, HasChanges: true},
	}}
	h := NewHandler(HandlerDeps{
		Watchlist: watchlist.NewService(watchlist.Deps{}, watchlist.List{}, scan),
		Live:      livePrices{"BTC": 68722},
	})

	code, body := serve(t, h, "GET", "/coins", "")

	for _, want := range []string{`"price": 68722`, `"price": 2,`} {
		if code != 200 || !strings.Contains(body, want) {
			t.Errorf("%s: %d %s", want, code, body)
		}
	}
}
