package console

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"newsbot/internal/monitor"
	"newsbot/internal/telegram"
)

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type Publisher struct {
	out      io.Writer
	chartDir string
}

func NewPublisher(out io.Writer, chartDir string) *Publisher {
	return &Publisher{out: out, chartDir: chartDir}
}

func (p *Publisher) Publish(_ context.Context, post monitor.Post) error {
	_, err := fmt.Fprintf(p.out, "\n────────── DRY RUN ──────────\n%s\n", telegram.Message(post.Event))
	if err != nil {
		return err
	}

	if len(post.Chart) == 0 || p.chartDir == "" {
		return nil
	}

	path := filepath.Join(p.chartDir, unsafeName.ReplaceAllString(post.Event.ID, "_")+".png")

	if err := os.WriteFile(path, post.Chart, 0o644); err != nil {
		return err
	}

	_, err = fmt.Fprintf(p.out, "[chart] %s\n", path)

	return err
}
