package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/market"
	"newsbot/internal/news"
	"newsbot/internal/radar"
	"newsbot/internal/schedule"
)

type livePulse []market.Pulse

func (p livePulse) Pulse(context.Context) ([]market.Pulse, error) {
	return p, nil
}

var wakeRule = radar.Rule{Interest: 0.02, Longs: 0.02}

func shifted(interest, longs float64) *market.Crowd {
	return &market.Crowd{Interest: []float64{interest, 0}, Longs: []float64{longs, 0}}
}

var pulseNow = livePulse{
	{Asset: "ZEC", Contract: "ZECUSDT", Price: 1050, Change1h: -0.08, VolumeSurge: 4, BuyShare: 0.4, Known: true, Sigma: 0.005, Crowd: shifted(-0.05, 0.03)},
	{Asset: "COOKIE", Contract: "COOKIEUSDT", Price: 0.009, Change1h: 0.05, VolumeSurge: 6, BuyShare: 0.7, Known: true, Sigma: 0.005, Crowd: shifted(0.04, -0.02)},
	{Asset: "BTC", Contract: "BTCUSDT", Price: 73000, Change1h: 0.001, VolumeSurge: 0.9, BuyShare: 0.5, Known: true, Sigma: 0.005},
}

func TestThePulseIsFilteredSortedAndCut(t *testing.T) {
	tool := PulseTool{Market: pulseNow, Rule: wakeRule}

	largest, err := use(t, tool, `{"views":[{"sort_by":"move_1h_size_fraction","order":"desc","limit":2}]}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	if zec, cookie := strings.Index(largest, `"ZEC"`), strings.Index(largest, `"COOKIE"`); zec < 0 || cookie < zec || strings.Contains(largest, `"BTC"`) {
		t.Errorf("the two largest moves, larger first: %s", largest)
	}

	both, err := use(t, tool, `{"views":[{"min":{"volume_1h_vs_average":2,"taker_buy_share_1h_fraction":0.6}},{"max":{"change_1h_fraction":-0.01}}]}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Count(both, `"matched":1`) != 2 || strings.Index(both, `"COOKIE"`) > strings.Index(both, `"ZEC"`) {
		t.Errorf("two views answered apart, in order: %s", both)
	}

	for _, wrong := range []string{`{"views":[{"sort_by":"mood"}]}`, `{"views":[]}`, `{}`} {
		if _, err := use(t, tool, wrong, &agents.Turn{}); err == nil {
			t.Errorf("passed: %s", wrong)
		}
	}
}

func TestTradFiIsAskedApartByTheSameRuleAndWithoutBTC(t *testing.T) {
	gold := market.Pulse{Asset: "XAU", Contract: "XAUUSDT", Change1h: 0.006, Crowd: shifted(0.03, 0.03)}
	tool := PulseTool{Market: pulseNow, TradFi: livePulse{gold}, Rule: wakeRule}

	got, err := use(t, tool, `{"market":"tradfi","views":[{"woke_up_only":true}]}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, `"XAU"`) || strings.Contains(got, `"ZEC"`) || strings.Contains(got, "btc_1h_fraction") {
		t.Errorf("tradfi pulse: %s", got)
	}
}

func TestAnIdeaNeedsACoinOfThePulseADirectionAndWhatCancelsIt(t *testing.T) {
	tool := IdeaTool{Market: pulseNow, Limit: 2, Crowded: -0.0005}
	turn := &agents.Turn{}

	wrong := []string{
		`{"symbol":"DOGE","direction":"лонг","why":"росте","cancel":"нижче 0.07","event_effect":"події немає"}`,
		`{"symbol":"ZEC","direction":"вниз","why":"падає","cancel":"вище 1100","event_effect":"події немає"}`,
		`{"symbol":"ZEC","direction":"шорт","why":"падає","cancel":" ","event_effect":"події немає"}`,
	}

	for _, args := range wrong {
		if _, err := use(t, tool, args, turn); err == nil {
			t.Errorf("passed: %s", args)
		}
	}

	for _, args := range []string{
		`{"symbol":"zec","direction":"шорт","why":"−8% за годину на обігу ×4","cancel":"повернення вище 1100","event_effect":"події немає"}`,
		`{"symbol":"COOKIE","direction":"лонг","why":"+5% на обігу ×6","cancel":"година нижче 0.0085","event_effect":"події немає"}`,
	} {
		if _, err := use(t, tool, args, turn); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := use(t, tool, `{"symbol":"ZEC","direction":"шорт","why":"ще раз","cancel":"x","event_effect":"події немає"}`, turn); err == nil {
		t.Error("a second idea about one coin passed")
	}

	if _, err := use(t, tool, `{"symbol":"BTC","direction":"лонг","why":"тримається","cancel":"нижче 72000","event_effect":"події немає"}`, turn); err == nil {
		t.Error("an idea over the limit passed")
	}

	if len(turn.Ideas) != 2 || turn.Ideas[0].Asset != "ZEC" || turn.Ideas[0].Cancel != "повернення вище 1100" || turn.Ideas[0].Price != 1050 {
		t.Errorf("ideas %+v", turn.Ideas)
	}

	crowded := IdeaTool{Market: livePulse{{Asset: "INIT", Price: 0.19, Funding: -0.078}}, Limit: 2, Crowded: -0.0005}

	if _, err := use(t, crowded, `{"symbol":"INIT","direction":"шорт","why":"падає","cancel":"вище 0.215","event_effect":"події немає"}`, &agents.Turn{}); err == nil || !strings.Contains(err.Error(), "переповнений") {
		t.Errorf("a short into a crowded short: %v", err)
	}

	if _, err := use(t, crowded, `{"symbol":"INIT","direction":"лонг","why":"сквіз","cancel":"нижче 0.18","event_effect":"події немає"}`, &agents.Turn{}); err != nil {
		t.Errorf("a long against a crowded short refused: %v", err)
	}
}

func TestThePulseTellsWhichCoinsWokeUpAndAnIdeaNamesItsEvent(t *testing.T) {
	rule := wakeRule

	only, err := use(t, PulseTool{Market: pulseNow, Rule: rule}, `{"views":[{"woke_up_only":true}]}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(only, `"matched":2`) || strings.Contains(only, `"BTC"`) || !strings.Contains(only, `"woke_up":{"minutes_ago":5`) || !strings.Contains(only, `"open_interest_5m_fraction":-0.05`) {
		t.Errorf("the coins that woke up: %s", only)
	}

	tool := IdeaTool{Market: pulseNow, Limit: 3, Crowded: -0.0005, Rule: rule}
	turn := &agents.Turn{}

	refused := []string{
		`{"symbol":"ZEC","direction":"шорт","why":"падає","cancel":"вище 1100","event":"лістинг на новій біржі — Видання","event_effect":"проти ідеї"}`,
		`{"symbol":"ZEC","direction":"шорт","why":"падає","cancel":"вище 1100","event":"","event_effect":"за ідею"}`,
		`{"symbol":"ZEC","direction":"шорт","why":"падає","cancel":"вище 1100","event":"злам — Видання","event_effect":"події немає"}`,
		`{"symbol":"ZEC","direction":"шорт","why":"падає","cancel":"вище 1100"}`,
	}

	for _, args := range refused {
		if _, err := use(t, tool, args, turn); err == nil {
			t.Errorf("passed: %s", args)
		}
	}

	if _, err := use(t, tool, `{"symbol":"ZEC","direction":"шорт","why":"падає","cancel":"вище 1100","event":"злам мосту — Видання","event_effect":"за ідею"}`, turn); err != nil {
		t.Fatal(err)
	}

	if _, err := use(t, tool, `{"symbol":"BTC","direction":"шорт","why":"стоїть","cancel":"вище 74000","event_effect":"події немає"}`, turn); err != nil {
		t.Fatal(err)
	}

	if zec, btc := turn.Ideas[0], turn.Ideas[1]; zec.Event != "злам мосту — Видання" || !zec.Woke || btc.Woke || btc.Event != "" {
		t.Errorf("ideas %+v", turn.Ideas)
	}
}

func TestWhatHistoryMeasuredIsToldOnlyAboutAMarkedCoinAndItsCase(t *testing.T) {
	calm := radar.Coin{Pulse: market.Pulse{Change1h: 0.05}}
	woke := radar.Coin{Wake: &radar.Wake{}}
	both := radar.Coin{Wake: &radar.Wake{}, Trend: &market.Run{Hours: 24}}

	if measured(calm) != nil {
		t.Error("an unmarked coin was given measurements")
	}

	if got := measured(woke); len(got) != 1 || got[0] != wokeMeasured {
		t.Errorf("a contract that woke up: %v", got)
	}

	if got := measured(radar.Coin{Launch: &radar.Launch{}}); len(got) != 1 || got[0] != launchMeasured {
		t.Errorf("a contract that launched from quiet: %v", got)
	}

	if got := measured(both); len(got) != 2 || got[0] != trendMeasured {
		t.Errorf("a trend that woke up: %v", got)
	}
}

func TestTheDeskBriefListsOnlyMarkedContractsWithTheirKeyFactors(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	crypto := radar.Board{At: now, Bench: -0.001, Coins: []radar.Coin{
		{Pulse: market.Pulse{Asset: "CALM", Change1h: 0.001}},
		{Pulse: market.Pulse{Asset: "GRAM", Change1h: 0.002}, Trend: &market.Run{Hours: 72, Move: 0.2}},
		{Pulse: market.Pulse{Asset: "MAGMA", Change1h: 0.12, Funding: 0.0009, LegMinutes: 50, LegMove: 0.15, Crowd: &market.Crowd{LongShare: 0.62, HourChange: 0.04}}, Wake: &radar.Wake{Minutes: 20, Interest: 0.05, Longs: 0.03}},
		{Pulse: market.Pulse{Asset: "KGEN", Change15m: -0.04, Change1h: -0.01}, Wake: &radar.Wake{Minutes: 5, Interest: -0.03, Longs: 0.04}, Stories: []radar.Story{{Title: "KGEN unlock"}}},
	}}
	tool := BriefTool{
		Radar:   board(crypto),
		Recent:  stubEvents{{Kind: "делістинг", Title: "Binance Will Delist KGEN", Symbols: []news.Symbol{"KGEN"}}},
		Agenda:  calendar{},
		Crowded: -0.0005,
	}

	got, err := use(t, tool, `{}`, &agents.Turn{})
	if err != nil {
		t.Fatal(err)
	}

	order := []int{strings.Index(got, `"KGEN"`), strings.Index(got, `"MAGMA"`), strings.Index(got, `"GRAM"`)}

	if order[0] < 0 || order[1] < order[0] || order[2] < order[1] || strings.Contains(got, `"CALM"`) {
		t.Fatalf("the newest wake first, then the older, then a trend, and nothing unmarked: %s", got)
	}

	for _, want := range []string{`"marks":["woke_up"]`, `"minutes_ago":5`, `"long_accounts_share_fraction":0.62`, `"вниз"`, "KGEN unlock", "Binance Will Delist KGEN", longCrowded, wokeMeasured, `"steady_trend_hours":72`, `"btc_1h_fraction":-0.001`} {
		if !strings.Contains(got, want) {
			t.Errorf("no %s in %s", want, got)
		}
	}
}

func TestAnIdeaMayNameATradFiContract(t *testing.T) {
	tool := IdeaTool{Market: pulseNow, TradFi: livePulse{{Asset: "CL", Contract: "CLUSDT", Price: 61.2}}, Limit: 2, Crowded: -0.0005}
	turn := &agents.Turn{}

	if _, err := use(t, tool, `{"symbol":"CL","direction":"лонг","why":"росте на обігу","cancel":"нижче 60","event_effect":"події немає"}`, turn); err != nil {
		t.Fatal(err)
	}

	if len(turn.Ideas) != 1 || turn.Ideas[0].Asset != "CL" || turn.Ideas[0].Price != 61.2 {
		t.Errorf("ideas %+v", turn.Ideas)
	}

	if _, err := use(t, tool, `{"symbol":"NOPE","direction":"лонг","why":"росте","cancel":"нижче","event_effect":"події немає"}`, turn); err == nil {
		t.Error("an idea about a contract of neither board passed")
	}
}

func TestTheMeasuredReactionIsToldOnlyWhenAReleaseOfThatKindIsAhead(t *testing.T) {
	ppi := schedule.NewEntry("США", time.Now(), "PPI", "")
	cpi := schedule.NewEntry("США", time.Now(), "CPI", "").Of(schedule.Inflation)

	unlock := schedule.Entry{Kind: schedule.Unlock, Assets: []string{"KAITO"}, Share: 0.07}

	if unlocksMeasured([]schedule.Entry{cpi}) != "" || unlocksMeasured([]schedule.Entry{cpi, unlock}) != unlockMeasured {
		t.Error("what was measured about unlocks goes only with an unlock in the calendar")
	}

	if releaseMeasured([]schedule.Entry{ppi}) != "" || releaseMeasured([]schedule.Entry{ppi, cpi}) != macroMeasured {
		t.Error("the measurement belongs to inflation, jobs and the Fed's decision alone")
	}
}
