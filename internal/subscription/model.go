package subscription

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Month struct {
	t time.Time
}

func ParseMonthMMYYYY(s string) (Month, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return Month{}, fmt.Errorf("invalid month format: %q (expected MM-YYYY)", s)
	}

	mm, err := strconv.Atoi(parts[0])
	if err != nil || mm < 1 || mm > 12 {
		return Month{}, fmt.Errorf("invalid month: %q", parts[0])
	}

	yy, err := strconv.Atoi(parts[1])
	if err != nil || yy < 1 {
		return Month{}, fmt.Errorf("invalid year: %q", parts[1])
	}

	t := time.Date(yy, time.Month(mm), 1, 0, 0, 0, 0, time.UTC)
	return Month{t: t}, nil
}

func (m Month) Time() time.Time {
	return m.t
}

func (m Month) String() string {
	if m.t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%02d-%04d", int(m.t.Month()), m.t.Year())
}

func (m Month) Compare(other Month) int {
	if m.t.Before(other.t) {
		return -1
	}
	if m.t.After(other.t) {
		return 1
	}
	return 0
}
