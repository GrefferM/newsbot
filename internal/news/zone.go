package news

import "time"

var Kyiv = mustLocation("Europe/Kyiv")

func mustLocation(name string) *time.Location {
	zone, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}

	return zone
}
