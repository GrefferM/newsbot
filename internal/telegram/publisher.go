package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"newsbot/internal/monitor"
)

const (
	defaultAPIBase  = "https://api.telegram.org"
	requestTimeout  = 30 * time.Second
	maxFloodRetries = 3
	maxCaption      = 1024
)

type Publisher struct {
	token   string
	chatID  string
	apiBase string
	http    *http.Client
}

func NewPublisher(token, chatID string) *Publisher {
	return &Publisher{
		token:   token,
		chatID:  chatID,
		apiBase: defaultAPIBase,
		http:    &http.Client{Timeout: requestTimeout},
	}
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
	Parameters struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

type request struct {
	method      string
	body        []byte
	contentType string
}

func (p *Publisher) Publish(ctx context.Context, post monitor.Post) error {
	text := Message(post.Event)

	if len(post.Chart) == 0 {
		_, err := p.call(ctx, p.textRequest(text, 0))

		return err
	}

	if utf8.RuneCountInString(text) <= maxCaption {
		_, err := p.call(ctx, p.photoRequest(post.Chart, text))

		return err
	}

	photoID, err := p.call(ctx, p.photoRequest(post.Chart, ""))
	if err != nil {
		return err
	}

	_, err = p.call(ctx, p.textRequest(text, photoID))

	return err
}

func (p *Publisher) textRequest(text string, replyTo int) request {
	payload := map[string]any{
		"chat_id":                  p.chatID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	}

	if replyTo > 0 {
		payload["reply_parameters"] = map[string]any{
			"message_id":                  replyTo,
			"allow_sending_without_reply": true,
		}
	}

	body, _ := json.Marshal(payload)

	return request{
		method:      "sendMessage",
		body:        body,
		contentType: "application/json",
	}
}

func (p *Publisher) photoRequest(png []byte, caption string) request {
	var buf bytes.Buffer

	w := multipart.NewWriter(&buf)

	_ = w.WriteField("chat_id", p.chatID)

	if caption != "" {
		_ = w.WriteField("caption", caption)
		_ = w.WriteField("parse_mode", "HTML")
	}

	part, _ := w.CreateFormFile("photo", "chart.png")
	_, _ = part.Write(png)
	_ = w.Close()

	return request{
		method:      "sendPhoto",
		body:        buf.Bytes(),
		contentType: w.FormDataContentType(),
	}
}

func (p *Publisher) call(ctx context.Context, r request) (int, error) {
	for attempt := 0; ; attempt++ {
		resp, status, err := p.send(ctx, r)
		if err != nil {
			return 0, err
		}

		if resp.OK {
			return resp.Result.MessageID, nil
		}

		if status != http.StatusTooManyRequests || attempt == maxFloodRetries {
			return 0, fmt.Errorf("telegram %s: HTTP %d: %s", r.method, status, resp.Description)
		}

		if !waitFlood(ctx, resp.Parameters.RetryAfter) {
			return 0, ctx.Err()
		}
	}
}

func (p *Publisher) send(ctx context.Context, r request) (apiResponse, int, error) {
	url := fmt.Sprintf("%s/bot%s/%s", p.apiBase, p.token, r.method)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(r.body))
	if err != nil {
		return apiResponse{}, 0, p.redacted(err)
	}

	req.Header.Set("Content-Type", r.contentType)

	httpResp, err := p.http.Do(req)
	if err != nil {
		return apiResponse{}, 0, p.redacted(err)
	}

	defer httpResp.Body.Close()

	var resp apiResponse
	_ = json.NewDecoder(httpResp.Body).Decode(&resp)

	return resp, httpResp.StatusCode, nil
}

func (p *Publisher) redacted(err error) error {
	return errors.New("telegram: " + strings.ReplaceAll(err.Error(), p.token, "<token>"))
}

func waitFlood(ctx context.Context, seconds int) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Duration(max(seconds, 1)) * time.Second):
		return true
	}
}
