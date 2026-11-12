package testnet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"newsbot/internal/trading"
	"newsbot/internal/websocket"
)

const (
	streamURL      = "wss://fstream.binancefuture.com/ws/"
	fillsSilence   = 10 * time.Minute
	fillsKeepAlive = 30 * time.Minute
	fillsLifetime  = 23 * time.Hour
	listenKeyPath  = "/fapi/v1/listenKey"
)

func (t *Broker) Fills(ctx context.Context, each func(trading.Order)) error {
	var key struct {
		Key string `json:"listenKey"`
	}

	if err := t.do(ctx, http.MethodPost, t.base+listenKeyPath, true, &key); err != nil {
		return fmt.Errorf("listen key: %w", err)
	}

	return websocket.Listen(ctx, websocket.Session{
		URL:      t.stream + key.Key,
		Silence:  fillsSilence,
		Lifetime: fillsLifetime,
		Every:    fillsKeepAlive,
		Tick: func() error {
			return t.do(ctx, http.MethodPut, t.base+listenKeyPath, true, &struct{}{})
		},
		Each: func(message []byte) error {
			if order, filled := parseFill(message); filled {
				each(order)
			}

			return nil
		},
	})
}

type rawOrderUpdate struct {
	Type  string `json:"e"`
	Event int64  `json:"E"`
	At    int64  `json:"T"`
	Order struct {
		ID       int64  `json:"i"`
		Client   string `json:"c"`
		Symbol   string `json:"s"`
		Side     string `json:"S"`
		Last     string `json:"L"`
		LastQty  string `json:"l"`
		Price    string `json:"p"`
		Executed string `json:"z"`
	} `json:"o"`
}

func parseFill(message []byte) (trading.Order, bool) {
	var raw rawOrderUpdate

	if json.Unmarshal(message, &raw) != nil || raw.Type != "ORDER_TRADE_UPDATE" {
		return trading.Order{}, false
	}

	executed := parse(raw.Order.Executed)
	if executed <= 0 {
		return trading.Order{}, false
	}

	return trading.Order{
		ID:       strconv.FormatInt(raw.Order.ID, 10),
		Tag:      raw.Order.Client,
		Contract: raw.Order.Symbol,
		Price:    parse(raw.Order.Price),
		Executed: executed,
	}, true
}
