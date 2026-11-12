package news

import "time"

type Observation struct {
	SeenAt  time.Time
	Source  string
	Event   Event
	Outcome string
}

type Failure struct {
	At     time.Time
	Source string
	Err    error
}
