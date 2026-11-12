package trading

import (
	"math"
	"testing"
	"time"
)

var (
	strategy = Strategy{
		StopLoss:     0.30,
		TakeProfit:   0.30,
		Hold:         24 * time.Hour,
		EstimatedFee: 0.002,
	}

	limits = Limits{MaxExposure: 1, MinVolume: 1e6, CrowdedFunding: -0.0005, DailyLoss: 0.10}

	liquid = Market{Listed: true, Contract: "MOVEUSDT", Price: 1, Funding: 0.0001, QuoteVolume: 5e6}

	now = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
)

func TestAnEntryIsCheckedAndTheDailyLossHaltsIt(t *testing.T) {
	full := Book{}.Funded(1000, 0)
	full.Positions = []Position{{Size: 0.9}}

	crowded := liquid
	crowded.Funding = -0.001

	thin := liquid
	thin.QuoteVolume = 1e5

	cases := []struct {
		name   string
		market Market
		book   Book
		failed string
	}{
		{"clean", liquid, Book{}.Funded(1000, 0), ""},
		{"no perpetual", Market{}, Book{}.Funded(1000, 0), "Перп торгується"},
		{"exposure over the cap", liquid, full, "Експозиція після входу"},
		{"crowded short", crowded, Book{}.Funded(1000, 0), "Шорт не переповнений"},
		{"thin contract", thin, Book{}.Funded(1000, 0), "Ліквідність"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var failed string

			for _, check := range limits.Checks(c.market, Short, 0.2, c.book.Exposure()) {
				if !check.OK && failed == "" {
					failed = check.Name
				}
			}

			if failed != c.failed {
				t.Errorf("failed %q, want %q", failed, c.failed)
			}
		})
	}

	taken := Book{}.Funded(1000, 0)
	taken.Positions = []Position{{Size: 1}}

	if limits.Checks(liquid, Short, 0, taken.Exposure())[1].OK || !limits.Checks(liquid, Short, 0, full.Exposure())[1].OK {
		t.Error("an account with no free exposure passes no entry, whatever its size")
	}

	lost := Book{}.Funded(1000, 0)
	lost.Positions = []Position{{Size: 0.2, ClosedAt: now.Add(-time.Hour), PnL: -150}}

	if !lost.Halted(limits, now) || (Book{}).Funded(1000, 0).Halted(limits, now) {
		t.Error("the daily loss told wrong")
	}
}

func entry(size float64) Decision {
	return Decision{Symbol: "MOVE", Side: Short, Kind: KindPlan, Size: size, Market: liquid}
}

func TestPositionExitsOnStopTargetOrTime(t *testing.T) {
	p := strategy.Open(entry(0.2), Book{}.Funded(1000, 0), now)

	cases := []struct {
		name   string
		price  float64
		at     time.Time
		reason string
		done   bool
	}{
		{"still open", 0.9, now.Add(time.Hour), "", false},
		{"stop", 1.31, now.Add(time.Hour), ExitStop, true},
		{"target", 0.69, now.Add(time.Hour), ExitTarget, true},
		{"time", 0.95, now.Add(24 * time.Hour), ExitTime, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reason, done := p.ExitAt(c.price, c.at)

			if reason != c.reason || done != c.done {
				t.Errorf("got %q %v, want %q %v", reason, done, c.reason, c.done)
			}
		})
	}
}

func TestAClosedShortKeepsItsResultAndTheEquityIsTheExchanges(t *testing.T) {
	book := Book{}.Funded(1000, 0)
	p := strategy.Open(entry(0.2), book, now)
	book = book.With(p)

	if p.Stake != 200 || book.Exposure() != 0.2 {
		t.Fatalf("stake %v exposure %v", p.Stake, book.Exposure())
	}

	closed := strategy.Close(p, 0.8, ExitTime, now.Add(24*time.Hour))
	book = book.With(closed)

	want := 200 * (1/0.8 - 1 - 0.002)

	if math.Abs(closed.PnL-want) > 1e-9 || book.Equity != 1000 || book.Exposure() != 0 || len(book.OpenPositions()) != 0 {
		t.Errorf("result %v equity %v exposure %v", closed.PnL, book.Equity, book.Exposure())
	}

	settled := closed.Settled(48)
	book = book.With(settled).Funded(1048, 0)

	if settled.Return != 0.24 || book.Equity != 1048 || book.Start != 1000 || (Book{}).Funded(5000, 0).Start != 5000 {
		t.Errorf("the exchange's result and balance replace the desk's own: %+v, equity %v", settled, book.Equity)
	}

	small := Book{}.Funded(5000, 100)
	moved := small.Funded(4997.5, 100)

	if small.Start != 100 || small.Equity != 100 || moved.Equity != 97.5 || moved.Start != 100 {
		t.Errorf("a working capital sizes the desk, and the exchange's balance moves it: %+v then %+v", small, moved)
	}

	if reset := moved.Funded(10000, 100); reset.Equity != 97.5 || reset.Funded(10002, 100).Equity != 99.5 {
		t.Errorf("a balance that jumps by more than the whole account is a reset or a transfer, not a trade: %+v", reset)
	}

	if again := moved.Funded(4997.5, 200); again.Start != 200 || again.Equity != 200 || again.Funded(5000, 0).Start != 5000 {
		t.Errorf("a changed capital starts the count anew: %+v", again)
	}
}

func TestALongMirrorsAShort(t *testing.T) {
	strategy := Strategy{StopLoss: 0.10, TakeProfit: 0.20, Hold: time.Hour}
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	long := strategy.Filled(Position{Side: Long, Stake: 100, Opened: now, Deadline: now.Add(time.Hour)}, 100, 1)

	if long.Stop != 90 || long.Target != 120 {
		t.Fatalf("long stop %v, target %v", long.Stop, long.Target)
	}

	cases := map[float64]string{89: ExitStop, 121: ExitTarget, 100: ""}

	for price, want := range cases {
		if got, _ := long.Triggered(price); got != want {
			t.Errorf("at %v: %q, want %q", price, got, want)
		}
	}

	if won := strategy.Close(long, 120, ExitTarget, now); won.Return < 0.1999 || won.PnL < 19.99 {
		t.Errorf("a long closed higher lost: %+v", won)
	}

	if lost := strategy.Close(long, 90, ExitStop, now); lost.Return > -0.0999 {
		t.Errorf("a long closed lower won: %+v", lost)
	}

	limits := Limits{MaxExposure: 1, CrowdedFunding: -0.0005}
	crowded := Market{Listed: true, Funding: 0.001, QuoteVolume: 1}

	for side, ok := range map[Side]bool{Short: true, Long: false} {
		checks := limits.Checks(crowded, side, 0.1, 0)

		if checks[2].OK != ok {
			t.Errorf("%s with funding paid by longs: %+v", side, checks[2])
		}
	}
}
