package websocket

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	URL      string
	Header   http.Header
	Silence  time.Duration
	Lifetime time.Duration
	Ping     time.Duration
	Opened   func(conn *Conn) error
	Every    time.Duration
	Tick     func() error
	Each     func(message []byte) error
}

func Listen(ctx context.Context, s Session) error {
	ctx, cancel := context.WithTimeout(ctx, s.Lifetime)
	defer cancel()

	conn, err := Dial(ctx, s.URL, s.Header)
	if err != nil {
		return err
	}

	var pinging sync.WaitGroup
	defer pinging.Wait()

	stop := context.AfterFunc(ctx, func() {
		conn.Close()
	})
	defer stop()

	defer conn.Close()
	defer cancel()

	if s.Opened != nil {
		if err := s.Opened(conn); err != nil {
			return err
		}
	}

	if s.Ping > 0 {
		pinging.Go(func() {
			keepAlive(ctx, conn, s.Ping)
		})
	}

	ticked := time.Now()

	for ctx.Err() == nil {
		if s.Tick != nil && time.Since(ticked) >= s.Every {
			if err := s.Tick(); err != nil {
				return err
			}

			ticked = time.Now()
		}

		message, err := conn.Read(s.Silence)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			return err
		}

		if err := s.Each(message); err != nil {
			return err
		}
	}

	return nil
}

func keepAlive(ctx context.Context, conn *Conn, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if conn.Ping() != nil {
				return
			}
		}
	}
}

type Retry struct {
	Min    time.Duration
	Max    time.Duration
	Steady time.Duration
}

func (r Retry) Keep(ctx context.Context, run func(context.Context) error, failed func(error)) {
	pause := r.Min

	for ctx.Err() == nil {
		started := time.Now()

		err := run(ctx)
		if ctx.Err() != nil {
			return
		}

		if time.Since(started) > r.Steady {
			pause = r.Min
		}

		failed(err)

		select {
		case <-ctx.Done():
			return
		case <-time.After(pause):
		}

		pause = min(pause*2, r.Max)
	}
}
