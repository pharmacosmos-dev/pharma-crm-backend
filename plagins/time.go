package plagins

import "time"

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

// func convertStringToCustomeTime(str string) (customTime CustomTime, err error) {
// 	var t time.Time
// 	t, err = time.Parse(DateTime, str)
// 	if err != nil {
// 		return
// 	}

// 	return CustomTime{t}
// }

// default duration: 23 hours and 59 minutes
func AddDefaultDuration(defaultTime CustomTime, t *CustomTime) CustomTime {
	if t == nil {
		return defaultTime.Add(Duration23HoursAnd59Minutes)
	}
	return CustomTime(defaultTime.Add(Duration23HoursAnd59Minutes))
}
