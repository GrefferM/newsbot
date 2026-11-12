package tools

import "testing"

func TestAListIsShapedOnTheServer(t *testing.T) {
	body := []byte(`[{"symbol":"A","priceChangePercent":"-3.1","x":1},{"symbol":"B","priceChangePercent":"-30.2","x":2},{"symbol":"C","priceChangePercent":"5","x":3}]`)

	got, err := shape{Fields: []string{"symbol", "priceChangePercent"}, SortBy: "priceChangePercent", Order: "asc", Limit: 2}.apply(body)
	if err != nil || got != `[{"priceChangePercent":"-30.2","symbol":"B"},{"priceChangePercent":"-3.1","symbol":"A"}]` {
		t.Fatalf("got %s err %v", got, err)
	}

	if same, _ := (shape{}).apply(body); same != string(body) {
		t.Errorf("unshaped body changed: %s", same)
	}
}
