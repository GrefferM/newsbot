package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"newsbot/internal/binance"
	"newsbot/internal/httpclient"
	"newsbot/internal/news"
	"newsbot/internal/trading"
)

const checkStake = 50

func checkTestnet(ctx context.Context, cfg config, contract string, log *slog.Logger) error {
	if cfg.TestnetKey == "" || cfg.TestnetSecret == "" {
		return errors.New("-check-testnet needs BINANCE_TESTNET_KEY and BINANCE_TESTNET_SECRET")
	}

	ticker, listed, err := binance.NewFutures(httpclient.New(), tickerMaxAge).Ticker(ctx, news.Symbol(strings.TrimSuffix(contract, "USDT")))
	if err != nil || !listed {
		return fmt.Errorf("price of %s: listed=%v err=%v", contract, listed, err)
	}

	broker := binance.NewTestnet(cfg.TestnetKey, cfg.TestnetSecret)
	now := time.Now().UTC()

	planned := cfg.Strategy.Filled(trading.Position{
		ID:       "check",
		Contract: contract,
		Stake:    checkStake,
		Opened:   now,
		Deadline: now.Add(cfg.Strategy.Hold),
	}, ticker.Price, 0)

	fill, err := broker.Enter(ctx, planned)
	if err != nil {
		return fmt.Errorf("enter: %w", err)
	}

	position := cfg.Strategy.Filled(planned, fill.Price, fill.Quantity)
	log.Info("entered", "contract", contract, "price", fill.Price, "quantity", fill.Quantity)

	protectErr := broker.Protect(ctx, position)
	if protectErr != nil {
		log.Error("protect failed", "err", protectErr)
	} else {
		log.Info("protected", "stop", position.Stop, "target", position.Target)
	}

	_, closed, err := broker.Exited(ctx, position)
	if err != nil {
		log.Error("position check failed", "err", err)
	} else {
		log.Info("position checked", "closed_by_exchange", closed)
	}

	price, err := broker.Close(ctx, position)
	if err != nil {
		return fmt.Errorf("close: %w", err)
	}

	log.Info("closed", "price", price, "return", position.Entry/price-1)

	return protectErr
}
