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
	TimeQueryFormat             = "2006-01-02T15:04:05+07:00"
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
		return nil // leave pointer nil
	}

	t, err := time.Parse(time.RFC3339, param)
	if err != nil {
		return fmt.Errorf("invalid time format: %v", err)
	}

	temp := CustomTime(t)
	ct = &temp
	return nil
}

// default duration: 23 hours and 59 minutes
func AddDefaultDuration(defaultTime CustomTime, t *CustomTime) CustomTime {
	// && _, err := time.Parse(TimeQueryFormat, time.Time(*t)); err != nil
	if t == nil {
		return defaultTime.Add(Duration23HoursAnd59Minutes)
	}
	return *t
}
