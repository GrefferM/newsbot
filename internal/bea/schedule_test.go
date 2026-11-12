package bea

import (
	"testing"
	"time"
)

func TestMajorReleasesAreReadFromTheSchedule(t *testing.T) {
	body := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nSUMMARY:GDP (Second Estimate) and Corporate Profits\\, 3rd Quar\r\n ter 2026\r\n" +
		"DTSTART;VALUE=DATE-TIME:20261120T133000Z\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nSUMMARY:GDP by County\\, 2025\r\nDTSTART;VALUE=DATE-TIME:20261125T133000Z\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nSUMMARY:Personal Income and Outlays\\, October 2026\r\nDTSTART;VALUE=DATE-TIME:20261125T133000Z\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	got := parse(body)
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}

	if got[0].Title != "GDP (Second Estimate) and Corporate Profits, 3rd Quarter 2026" || !got[0].Time.Equal(time.Date(2026, 11, 20, 13, 30, 0, 0, time.UTC)) {
		t.Errorf("first %+v", got[0])
	}
}
