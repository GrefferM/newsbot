package filestore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"newsbot/internal/agents"
	"newsbot/internal/news"
)

const tailBlock = 256 << 10

type JournalFile struct {
	mu   sync.Mutex
	path string
	log  func(err error)
}

func NewJournal(path string, log func(err error)) *JournalFile {
	return &JournalFile{path: path, log: log}
}

type journalEntry struct {
	SeenAt    time.Time     `json:"seen_at"`
	Source    string        `json:"source"`
	ID        string        `json:"id"`
	Category  news.Category `json:"category,omitempty"`
	Kind      string        `json:"kind,omitempty"`
	Title     string        `json:"title"`
	Published time.Time     `json:"published"`
	Symbols   []news.Symbol `json:"symbols,omitempty"`
	Severity  news.Severity `json:"severity"`
	Outcome   string        `json:"outcome"`
}

type failureEntry struct {
	SeenAt  time.Time `json:"seen_at"`
	Source  string    `json:"source"`
	Outcome string    `json:"outcome"`
	Error   string    `json:"error"`
}

func (j *JournalFile) Fail(f news.Failure) {
	j.write(failureEntry{
		SeenAt:  f.At,
		Source:  f.Source,
		Outcome: "помилка",
		Error:   f.Err.Error(),
	})
}

func (j *JournalFile) Record(o news.Observation) {
	e := o.Event

	j.write(journalEntry{
		SeenAt:    o.SeenAt,
		Source:    o.Source,
		ID:        e.ID,
		Category:  e.Category,
		Kind:      e.Kind,
		Title:     e.Title,
		Published: e.Time,
		Symbols:   e.Symbols,
		Severity:  e.Severity,
		Outcome:   o.Outcome,
	})
}

func (j *JournalFile) Keep(run agents.Record) {
	j.write(run)
}

func (j *JournalFile) Last(agent, topic string, count int) ([]agents.Record, error) {
	out := []agents.Record{}

	if count <= 0 {
		return out, nil
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	f, err := os.Open(j.path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}

	if err != nil {
		return nil, err
	}

	defer f.Close()

	mark := []byte(fmt.Sprintf(`"agent":%q`, agent))

	err = backwards(f, func(line []byte) bool {
		var record agents.Record

		if !bytes.Contains(line, mark) || json.Unmarshal(line, &record) != nil {
			return true
		}

		if record.Agent != agent || record.Topic != topic {
			return true
		}

		if record.Cleared {
			return false
		}

		out = append(out, record)

		return len(out) < count
	})

	slices.Reverse(out)

	return out, err
}

func backwards(f *os.File, visit func(line []byte) bool) error {
	info, err := f.Stat()
	if err != nil {
		return err
	}

	var rest []byte

	for end := info.Size(); end > 0; {
		start := max(0, end-tailBlock)
		block := make([]byte, end-start)

		if _, err := f.ReadAt(block, start); err != nil {
			return err
		}

		lines := bytes.Split(append(block, rest...), []byte{'\n'})
		rest = lines[0]

		for i := len(lines) - 1; i >= 1; i-- {
			if len(lines[i]) > 0 && !visit(lines[i]) {
				return nil
			}
		}

		end = start
	}

	if len(rest) > 0 {
		visit(rest)
	}

	return nil
}

func (j *JournalFile) write(entry any) {
	line, err := json.Marshal(entry)
	if err != nil {
		j.log(err)

		return
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	if err := j.append(append(line, '\n')); err != nil {
		j.log(err)
	}
}

func (j *JournalFile) append(line []byte) error {
	if err := os.MkdirAll(filepath.Dir(j.path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(j.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	_, err = f.Write(line)

	if cerr := f.Close(); err == nil {
		err = cerr
	}

	return err
}
