package binance

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"newsbot/internal/exchange"
	"newsbot/internal/news"
	"newsbot/internal/websocket"
)

const (
	streamURL       = "wss://api.binance.com/sapi/wss"
	streamTopic     = "com_announcement_en"
	streamWindow    = 30000
	streamPing      = 30 * time.Second
	streamSilence   = 90 * time.Second
	streamLifetime  = 23 * time.Hour
	streamBuffer    = 100
	streamRetryMin  = 5 * time.Second
	streamRetryMax  = 5 * time.Minute
	streamHealthy   = time.Minute
	streamPollEvery = time.Second
)

type Stream struct {
	key, secret string
	policy      exchange.AnnouncementPolicy
	items       chan exchange.Announcement
	failures    chan error
	start       sync.Once
}

func NewStream(key, secret string, policy exchange.AnnouncementPolicy) *Stream {
	return &Stream{
		key:      key,
		secret:   secret,
		policy:   policy,
		items:    make(chan exchange.Announcement, streamBuffer),
		failures: make(chan error, 1),
	}
}

func (s *Stream) Name() string {
	return "binance-stream"
}

func (s *Stream) Interval() time.Duration {
	return streamPollEvery
}

func (s *Stream) Poll(ctx context.Context) ([]news.Event, error) {
	s.start.Do(func() {
		go s.run(ctx)
	})

	select {
	case err := <-s.failures:
		return nil, err
	default:
	}

	now := time.Now()

	var out []news.Event

	for {
		select {
		case a := <-s.items:
			out = append(out, s.policy.Event(a, now))
		default:
			return out, nil
		}
	}
}

func (s *Stream) run(ctx context.Context) {
	retry := websocket.Retry{Min: streamRetryMin, Max: streamRetryMax, Steady: streamHealthy}

	retry.Keep(ctx, s.session, s.fail)
}

func (s *Stream) fail(err error) {
	if err == nil {
		return
	}

	select {
	case s.failures <- err:
	default:
	}
}

func (s *Stream) session(ctx context.Context) error {
	return websocket.Listen(ctx, websocket.Session{
		URL:      s.signedURL(time.Now()),
		Header:   http.Header{"X-MBX-APIKEY": {s.key}},
		Silence:  streamSilence,
		Lifetime: streamLifetime,
		Ping:     streamPing,
		Opened: func(conn *websocket.Conn) error {
			subscribe, _ := json.Marshal(map[string]string{"command": "SUBSCRIBE", "value": streamTopic})

			return conn.WriteText(subscribe)
		},
		Each: func(message []byte) error {
			a, ok, err := parseStreamMessage(message)
			if err != nil || !ok {
				return err
			}

			select {
			case s.items <- a:
			case <-ctx.Done():
			}

			return nil
		},
	})
}

func (s *Stream) signedURL(now time.Time) string {
	nonce := make([]byte, 16)
	rand.Read(nonce)

	query := fmt.Sprintf(
		"random=%s&topic=%s&recvWindow=%d&timestamp=%d",
		hex.EncodeToString(nonce), streamTopic, streamWindow, now.UnixMilli(),
	)

	return streamURL + "?" + query + "&signature=" + sign(s.secret, query)
}

func sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))

	return hex.EncodeToString(mac.Sum(nil))
}

type streamEnvelope struct {
	Type    string `json:"type"`
	SubType string `json:"subType"`
	Code    string `json:"code"`
	Topic   string `json:"topic"`
	Data    string `json:"data"`
}

type streamAnnouncement struct {
	CatalogID   int    `json:"catalogId"`
	PublishDate int64  `json:"publishDate"`
	Title       string `json:"title"`
}

func parseStreamMessage(message []byte) (exchange.Announcement, bool, error) {
	var envelope streamEnvelope
	if json.Unmarshal(message, &envelope) != nil {
		return exchange.Announcement{}, false, nil
	}

	if envelope.Type == "COMMAND" && envelope.Data != "SUCCESS" {
		err := fmt.Errorf("binance stream %s rejected: %s %s", envelope.SubType, envelope.Code, envelope.Data)

		return exchange.Announcement{}, false, err
	}

	if envelope.Type != "DATA" || envelope.Topic != streamTopic {
		return exchange.Announcement{}, false, nil
	}

	var a streamAnnouncement
	if json.Unmarshal([]byte(envelope.Data), &a) != nil || a.Title == "" {
		return exchange.Announcement{}, false, nil
	}

	return exchange.Announcement{
		Title:     a.Title,
		URL:       fmt.Sprintf(catalogURL, a.CatalogID),
		Published: time.UnixMilli(a.PublishDate),
	}, true, nil
}
