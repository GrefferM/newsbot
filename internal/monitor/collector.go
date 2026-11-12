package monitor

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"time"

	"newsbot/internal/news"
)

const (
	maxBackoff  = 15 * time.Minute
	outcomeSent = "надіслано"
)

type Collector struct {
	seen    SeenRepository
	journal Journal
	status  *Status
	log     *slog.Logger
}

func NewCollector(seen SeenRepository, journal Journal, status *Status, log *slog.Logger) *Collector {
	return &Collector{
		seen:    seen,
		journal: journal,
		status:  status,
		log:     log,
	}
}

func (c *Collector) Run(ctx context.Context, src Source, startDelay time.Duration, out chan<- news.Event) {
	delay := startDelay
	backoff := src.Interval()

	for {
		if !sleep(ctx, delay) {
			return
		}

		err := c.collect(ctx, src, out)
		if ctx.Err() != nil {
			return
		}

		if err != nil {
			backoff = nextBackoff(backoff, src.Interval(), err)
			delay = backoff

			c.status.Fail(src.Name(), err)
			c.journal.Fail(news.Failure{
				At:     time.Now().UTC(),
				Source: src.Name(),
				Err:    err,
			})
			c.log.Warn("poll failed", "source", src.Name(), "err", err, "retry_in", backoff)

			continue
		}

		backoff = src.Interval()
		delay = withJitter(src.Interval())
	}
}

func (c *Collector) collect(ctx context.Context, src Source, out chan<- news.Event) error {
	events, err := src.Poll(ctx)
	if err != nil {
		return err
	}

	seeded := c.seen.IsSeeded(src.Name())
	fresh := c.forward(ctx, src.Name(), events, seeded, out)

	if !seeded {
		c.seen.MarkSeeded(src.Name())
		c.log.Info("source seeded silently", "source", src.Name(), "items", len(events))
	}

	c.status.OK(src.Name(), len(events), fresh)

	return nil
}

func (c *Collector) forward(
	ctx context.Context,
	source string,
	events []news.Event,
	seeded bool,
	out chan<- news.Event,
) int {
	fresh := 0

	for _, e := range events {
		if !c.seen.MarkNew(e.ID) || !seeded {
			continue
		}

		c.record(source, e)

		if e.Muted != "" {
			continue
		}

		select {
		case out <- e:
			fresh++
		case <-ctx.Done():
			return fresh
		}
	}

	return fresh
}

func (c *Collector) record(source string, e news.Event) {
	outcome := outcomeSent
	if e.Muted != "" {
		outcome = e.Muted
	}

	c.journal.Record(news.Observation{
		SeenAt:  time.Now().UTC(),
		Source:  source,
		Event:   e,
		Outcome: outcome,
	})
}

func nextBackoff(current, interval time.Duration, err error) time.Duration {
	next := min(max(current*2, interval), maxBackoff)

	var hint RetryHint
	if errors.As(err, &hint) && hint.RetryAfter() > next {
		next = hint.RetryAfter()
	}

	return next
}

func withJitter(interval time.Duration) time.Duration {
	return interval + time.Duration(rand.Int64N(int64(interval/10)+1))
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
