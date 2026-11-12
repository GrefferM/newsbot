package autopilot

import (
	"testing"
	"time"

	"newsbot/internal/research"
	"newsbot/internal/trading"
)

var (
	longPlan  = research.Scenario{Condition: "закриття вище 1120", Direction: research.DirectionLong, WakeAbove: 1120, SizePct: 7, Stop: 1055, Take: 1250}
	shortPlan = research.Scenario{Condition: "відскок до 1085", Direction: research.DirectionShort, WakeAbove: 1085, SizePct: 10, Stop: 1130, Take: 1010}
	noon      = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
)

func TestAWatchWakesAtItsPriceAndRestsBetweenChecks(t *testing.T) {
	up := Watch{Plan: longPlan}
	down := Watch{Plan: research.Scenario{WakeBelow: 990}}

	if up.Woken(1119) || !up.Woken(1120) || down.Woken(991) || !down.Woken(990) {
		t.Error("woken at the wrong price")
	}

	asked := up.Asked(noon, "зачекати")

	if asked.Checks != 1 || asked.Rested(noon.Add(time.Minute), 15*time.Minute) || !asked.Rested(noon.Add(time.Hour), 15*time.Minute) {
		t.Errorf("rest told wrong: %+v", asked)
	}
}

func TestADraftTakesItsStopAndTargetFromThePlanAtThePriceNow(t *testing.T) {
	long, err := Watch{Asset: "ZEC", Plan: longPlan}.Draft("d", "ZECUSDT", 1125, 1000, time.Hour, noon)
	if err != nil || long.Side != trading.Long || long.Stake != 70 {
		t.Fatalf("long %+v, %v", long, err)
	}

	if stop, take := 1125*(1-long.StopLoss), 1125*(1+long.TakeProfit); stop < 1054.9 || stop > 1055.1 || take < 1249.9 || take > 1250.1 {
		t.Errorf("long stop %v, take %v", stop, take)
	}

	short, err := Watch{Asset: "ZEC", Plan: shortPlan}.Draft("d", "ZECUSDT", 1090, 1000, time.Hour, noon)
	if err != nil || short.Side != trading.Short || short.Stake != 100 {
		t.Fatalf("short %+v, %v", short, err)
	}

	if stop, take := 1090*(1+short.StopLoss), 1090*(1-short.TakeProfit); stop < 1129.9 || stop > 1130.1 || take < 1009.9 || take > 1010.1 {
		t.Errorf("short stop %v, take %v", stop, take)
	}

	if _, err := (Watch{Asset: "ZEC", Plan: longPlan}).Draft("d", "ZECUSDT", 1260, 1000, time.Hour, noon); err == nil {
		t.Error("a long drafted above its own target")
	}
}

func TestAPlanIsArmedOnceAndFoundAgain(t *testing.T) {
	ws, w, err := Watches{}.Arm("w1", "ZEC", longPlan, noon, 24*time.Hour)
	if err != nil || w.State != Armed || !w.Expires.Equal(noon.Add(24*time.Hour)) {
		t.Fatalf("armed %+v, %v", w, err)
	}

	if _, _, err := ws.Arm("w2", "ZEC", longPlan, noon, time.Hour); err == nil {
		t.Error("the same plan armed twice")
	}

	if _, _, err := ws.Arm("w3", "ZEC", research.Scenario{Condition: "?"}, noon, time.Hour); err == nil {
		t.Error("a plan without numbers armed")
	}

	ws = ws.With(w.Settled(Fired, "ордер поставлено"))

	if len(ws.Active()) != 0 || len(ws.Of("ZEC")) != 1 || ws.Of("ZEC")[0].State != Fired {
		t.Errorf("after firing: %+v", ws)
	}

	if _, _, err := ws.Arm("w4", "ZEC", longPlan, noon, time.Hour); err != nil {
		t.Errorf("a fired plan cannot be armed again: %v", err)
	}
}
