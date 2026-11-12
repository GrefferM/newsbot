package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"newsbot/internal/news"
)

const publisherStatusName = "telegram"

type Dispatcher struct {
	publisher   Publisher
	journal     Journal
	illustrator Illustrator
	enricher    *Enricher
	recent      *Recent
	status      *Status
	spacing     time.Duration
	log         *slog.Logger
}

type DispatcherDeps struct {
	Publisher   Publisher
	Journal     Journal
	Illustrator Illustrator
	Enricher    *Enricher
	Recent      *Recent
	Status      *Status
	Spacing     time.Duration
	Log         *slog.Logger
}

func NewDispatcher(deps DispatcherDeps) *Dispatcher {
	return &Dispatcher{
		publisher:   deps.Publisher,
		journal:     deps.Journal,
		illustrator: deps.Illustrator,
		enricher:    deps.Enricher,
		recent:      deps.Recent,
		status:      deps.Status,
		spacing:     deps.Spacing,
		log:         deps.Log,
	}
}

func (d *Dispatcher) Run(ctx context.Context, in <-chan news.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-in:
			if !d.Dispatch(ctx, e) {
				return
			}
		}
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, e news.Event) bool {
	d.enricher.Enrich(ctx, &e)
	if err := d.recent.Add(e); err != nil {
		d.log.Warn("save recent events", "err", err)
	}

	if e.Silent {
		return true
	}

	err := d.publisher.Publish(ctx, Post{Event: e, Chart: d.chart(ctx, e)})
	if err != nil {
		d.status.Fail(publisherStatusName, err)
		d.journal.Fail(news.Failure{
			At:     time.Now().UTC(),
			Source: publisherStatusName,
			Err:    fmt.Errorf("%s: %w", e.ID, err),
		})
		d.log.Error("send failed", "id", e.ID, "err", err)
	} else {
		d.status.OK(publisherStatusName, 1, 1)
		d.log.Info("sent", "id", e.ID, "kind", e.Kind)
	}

	return sleep(ctx, d.spacing)
}

func (d *Dispatcher) chart(ctx context.Context, e news.Event) []byte {
	if d.illustrator == nil || len(e.Symbols) == 0 {
		return nil
	}

	png, ok := d.illustrator.Illustrate(ctx, e)
	if !ok {
		return nil
	}

	return png
}
