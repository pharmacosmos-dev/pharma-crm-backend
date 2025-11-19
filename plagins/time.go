package plagins

import (
	"fmt"
	"time"
)

var (
	TashkentTimeDif             = 5 * time.Hour
	Duration23HoursAnd59Minutes = 23*time.Hour + 59*time.Minute
	Duration24Hours             = 24 * time.Hour
	DateTime                    = "2006-01-02 15:04:05"
)

type CustomTime time.Time

func (c CustomTime) ToUTC() CustomTime {
	return CustomTime(time.Time(c).In(time.UTC))
}

func (c CustomTime) GetString() string {
	return time.Time(c).Format(DateTime)
}

func (c CustomTime) GetTime() time.Time {
	return time.Time(c)
}

func (c CustomTime) Add(dur time.Duration) CustomTime {
	return CustomTime(time.Time(c).Add(dur))
}

func (c CustomTime) PrevDay() CustomTime {
	return CustomTime(c.Add(-Duration24Hours))
}

// UnmarshalParam implements the binding.UnmarshalParam interface for query parameter binding
func (ct *CustomTime) UnmarshalParam(param string) error {
	if param == "" {
		return nil
	}

	// Define the expected format(s) for your dates
	formats := []string{
		"2006-01-02T15:04:05Z07:00", // RFC3339
	}

	for _, format := range formats {
		t, err := time.Parse(format, param)
		if err == nil {
			*ct = CustomTime(t)
			return nil
		}
	}

	return fmt.Errorf("unable to parse time parameter: %s", param)
}

// default duration: 23 hours and 59 minutes
func AddDefaultDuration(defaultTime CustomTime, t *CustomTime) CustomTime {
	if t == nil {
		return defaultTime.Add(Duration23HoursAnd59Minutes)
	}
	return CustomTime(defaultTime.Add(Duration23HoursAnd59Minutes))
}
