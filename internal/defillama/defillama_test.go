package defillama

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAHackKeepsWhetherItsAmountIsKnown(t *testing.T) {
	var raws []rawHack

	body := `[{"date":1762905600,"name":"Bridge","classification":"Protocol Logic","technique":"Key","amount":12000000,"chain":["Ethereum"],"source":"https://x","returnedFunds":null},
{"date":1762905600,"name":"Wallet","amount":null}]`

	if err := json.Unmarshal([]byte(body), &raws); err != nil {
		t.Fatal(err)
	}

	known, unknown := raws[0].toDomain(), raws[1].toDomain()

	if known.Amount != 12e6 || !known.AmountKnown || known.Returned != 0 || known.Chains[0] != "Ethereum" || !known.Date.Equal(time.Unix(1762905600, 0)) {
		t.Errorf("known %+v", known)
	}

	if unknown.AmountKnown || unknown.Amount != 0 {
		t.Errorf("an unknown amount is not a zero one: %+v", unknown)
	}
}

func TestOnlyDollarStablecoinsWithAPriceAreWatched(t *testing.T) {
	var resp stablecoinsResponse

	body := `{"peggedAssets":[
{"name":"Tether","symbol":"USDT","pegType":"peggedUSD","price":0.999,"circulating":{"peggedUSD":180000000000}},
{"name":"Euro Coin","symbol":"EURC","pegType":"peggedEUR","price":1.08,"circulating":{"peggedEUR":200000000}},
{"name":"Quiet","symbol":"QT","pegType":"peggedUSD","price":null,"circulating":{"peggedUSD":5}}]}`

	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatal(err)
	}

	coins := resp.dollars()

	if len(coins) != 1 || coins[0].Symbol != "USDT" || coins[0].Price != 0.999 || coins[0].Supply != 180e9 {
		t.Errorf("coins %+v", coins)
	}
}
