package domain

import (
	"database/sql"
	"encoding/json"

	"time"

	"github.com/pharma-crm-backend/domain/constants"
)

// region int64

type NullInt64 struct {
	sql.NullInt64
}

func NewNullInt64(val int) NullInt64 {
	return NullInt64{
		NullInt64: sql.NullInt64{Int64: int64(val), Valid: val != 0},
	}
}

func (ni NullInt64) MarshalJSON() ([]byte, error) {
	if ni.Valid {
		return json.Marshal(ni.Int64)
	}
	return json.Marshal(nil)
}

func (ni *NullInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ni.Valid = false
		return nil
	}

	ni.Valid = true
	return json.Unmarshal(data, &ni.Int64)
}

func (ni NullInt64) ToInt() int {
	if ni.Valid {
		return int(ni.Int64)
	}

	return 0
}

// endregion

// region string

type NullString struct {
	sql.NullString
}

func NewNullString(val string) NullString {
	return NullString{
		sql.NullString{
			String: val, Valid: val != constants.NoValue,
		},
	}
}

func (ni NullString) MarshalJSON() ([]byte, error) {
	if ni.Valid {
		return json.Marshal(ni.String)
	}

	return json.Marshal(nil)
}

func (ns *NullString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == "\"\"" {
		ns.Valid = false
		return nil
	}

	ns.Valid = true
	return json.Unmarshal(data, &ns.String)
}

// endregion

// region float64

type NullFloat64 struct {
	sql.NullFloat64
}

func NewNullFloat64(val float64) NullFloat64 {
	if val == 0 {
		return NullFloat64{}
	}

	return NullFloat64{sql.NullFloat64{Float64: val, Valid: true}}
}

func (nf NullFloat64) MarshalJSON() ([]byte, error) {
	if nf.Valid {
		return json.Marshal(nf.Float64)
	}

	return json.Marshal(nil)
}

func (nf *NullFloat64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nf.Valid = false
		return nil
	}

	nf.Valid = true
	return json.Unmarshal(data, &nf.Float64)
}

func (nf *NullFloat64) Value() (float64, error) {
	return nf.Float64, nil
}

// endregion

// region time

type NullTime struct {
	sql.NullTime
}

func NewNullTime(val time.Time) NullTime {
	return NullTime{
		NullTime: sql.NullTime{Time: val, Valid: !val.IsZero()},
	}
}

func (nt NullTime) MarshalJSON() ([]byte, error) {
	if nt.Valid {
		return json.Marshal(nt.Time)
	}

	return json.Marshal(nil)
}

func (nt *NullTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nt.Valid = false
		return nil
	}

	nt.Valid = true
	return json.Unmarshal(data, &nt.Time)
}

// endregion

// region struct

type NullStruct[T any] struct {
	Valid bool
	Value T
}

func NewNullStruct[T any](value T, valid bool) NullStruct[T] {
	return NullStruct[T]{
		Valid: valid,
		Value: value,
	}
}

func (ns NullStruct[T]) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.Value)
	}
	return json.Marshal(nil)
}

func (ns *NullStruct[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		ns.Valid = false
		return nil
	}
	ns.Valid = true
	return json.Unmarshal(data, &ns.Value)
}

// endregion

// region Bool

// TODO: Replace with NullBool
type NullBoolDefaultFalse struct {
	sql.NullBool
}

func (ns NullBoolDefaultFalse) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return json.Marshal(ns.Bool)
	}

	return json.Marshal(false)
}

type NullBool struct {
	sql.NullBool
}

func (nb NullBool) MarshalJSON() ([]byte, error) {
	if nb.Valid {
		return json.Marshal(nb.Bool)
	}

	return json.Marshal(nil)
}

func (nb *NullBool) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		nb.Valid = false
		return nil
	}

	nb.Valid = true
	return json.Unmarshal(data, &nb.Bool)
}

// endregion
