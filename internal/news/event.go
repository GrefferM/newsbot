package news

import "time"

type Category string

const (
	ShortTrigger Category = "short"
	Incident     Category = "incident"
	Trade        Category = "trade"
	Reminder     Category = "reminder"
)

type Severity int

const (
	Normal Severity = iota
	High
	Critical
)

func SeverityOf(value, high, critical float64) Severity {
	switch {
	case value >= critical:
		return Critical
	case value >= high:
		return High
	default:
		return Normal
	}
}

type Origin struct {
	Name string
	URL  string
}

type Event struct {
	ID       string
	Source   string
	Category Category
	Kind     string
	Title    string
	URL      string
	Time     time.Time
	Symbols  []Symbol
	Details  []string
	Severity Severity
	Trend    Direction
	Muted    string
	Silent   bool
}
