package domain

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
)

type Point struct {
	Long float64 `json:"long,omitempty" validate:"required,numeric" db:"long" query:"long" gorm:"long"`
	Lat  float64 `json:"lat,omitempty" validate:"required,numeric" db:"lat" query:"lat" gorm:"lat"`
}

func (p Point) Value() (driver.Value, error) {
	// how you want to store it in DB (WKT)
	// if you never write this field, this still helps GORM understand it's a scalar
	if p.Lat == 0 && p.Long == 0 {
		return nil, nil // or return "POINT(0 0)" if you prefer
	}
	return fmt.Sprintf("POINT(%f %f)", p.Long, p.Lat), nil
}

func (p *Point) Scan(value any) error {
	if value == nil {
		return nil
	}

	var s string
	switch v := value.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("unexpected type for Point Scan: %T", value)
	}

	s = strings.TrimSpace(s)

	// Expect: "POINT(lon lat)"
	s = strings.TrimPrefix(s, "POINT(")
	s = strings.TrimSuffix(s, ")")

	parts := strings.Fields(s) // safer than Split(" ")
	if len(parts) != 2 {
		return fmt.Errorf("invalid POINT format: %q", s)
	}

	lon, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return fmt.Errorf("invalid lon %q: %w", parts[0], err)
	}

	lat, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return fmt.Errorf("invalid lat %q: %w", parts[1], err)
	}

	p.Long = lon
	p.Lat = lat
	return nil
}

func (p *Point) ToSinglePointWKT() string {
	return fmt.Sprintf("POINT(%f %f)", p.Long, p.Lat)
}

// Scan will override sqlx's default scan method
func (p *Points) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	case []byte:
		strValue = string(v)
	default:
		return fmt.Errorf("unexpected type for value: %T", value)
	}

	// Remove the "MULTIPOINT(" prefix
	strValue = strings.TrimPrefix(strValue, "MULTIPOINT(")
	// Remove the trailing ")"
	strValue = strings.TrimSuffix(strValue, ")")

	// Split the string value into individual coordinate pairs
	coordinatePairs := strings.Split(strValue, "),(")

	// Parse each coordinate pair into Point structs
	points := make([]Point, len(coordinatePairs))
	for i, pair := range coordinatePairs {
		// Remove any remaining parentheses
		pair = strings.Trim(pair, "()")

		// Split the coordinate pair into latitude and longitude
		coordinates := strings.Split(pair, " ")
		if len(coordinates) != 2 {
			return fmt.Errorf("invalid number of coordinates in pair: %s", pair)
		}

		lon, err := strconv.ParseFloat(coordinates[0], 64)
		if err != nil {
			return err
		}
		lat, err := strconv.ParseFloat(coordinates[1], 64)
		if err != nil {
			return err
		}

		points[i] = Point{Lat: lat, Long: lon}
	}

	*p = points
	return nil
}

type Points []Point
