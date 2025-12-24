package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

type PrescriptionResponse struct {
	Data []Prescription `json:"data"`
}
type DmedResponse struct {
	Prescriptions []string `json:"prescriptions"`
	Count         []int    `json:"count"`
	PerDay        []int    `json:"per_day"`
}

type Prescription struct {
	ID               int             `json:"id"`
	Number           string          `json:"number"`
	SafeCode         string          `json:"safe_code"`
	Status           string          `json:"status"`
	Type             string          `json:"type"`
	DistributionType string          `json:"distribution_type"`
	ExpirationDate   string          `json:"expiration_date"`
	Price            *any            `json:"price"`
	CreatedAt        string          `json:"created_at"`
	DrugAppointment  DrugAppointment `json:"drug_appointment"`
}

type DrugAppointment struct {
	ID                        int          `json:"id"`
	SingleDose                string       `json:"single_dose"`
	AdministrationFrequency   int          `json:"administration_frequency"`
	AdministrationFreqMeasure string       `json:"administration_frequency_measure"`
	CountDays                 int          `json:"count_days"`
	TotalAmount               int          `json:"total_amount"`
	Medications               []Medication `json:"medication"` // ← faqat array
	ATCs                      []ATC        `json:"atcs"`
	RemainingCount            int          `json:"remaining_count"`
	DosageForm                TitleOnly    `json:"dosage_form"`
	DrugAdministrationRoute   TitleOnly    `json:"drug_administration_route"`
	DrugAdministrationTimes   []any        `json:"drug_administration_times"`
}

type Medication struct {
	ID              int             `json:"id"`
	Title           string          `json:"title"`
	TitleEn         string          `json:"title_en"`
	Number          string          `json:"number"`
	SubstanceDosage SubstanceDosage `json:"substance_dosage"`
}

type SubstanceDosage struct {
	ID              int       `json:"id"`
	Dosage          float64   `json:"dosage"`
	MeasurementUnit TitleOnly `json:"measurement_unit"`
}

type ATC struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Code   string `json:"code"`
	IsLeaf bool   `json:"is_leaf"`
}

type TitleOnly struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

func (d *DrugAppointment) UnmarshalJSON(data []byte) error {
	type Alias DrugAppointment

	aux := &struct {
		*Alias
		Medication json.RawMessage `json:"medication"`
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Try array
	var meds []Medication
	if err := json.Unmarshal(aux.Medication, &meds); err == nil {
		d.Medications = meds
		return nil
	}

	// Try object
	var single Medication
	if err := json.Unmarshal(aux.Medication, &single); err == nil {
		d.Medications = []Medication{single}
		return nil
	}

	return fmt.Errorf("invalid medication format")
}

type DmedRequestLog struct {
	Id        int        `gorm:"id" json:"id"`
	Payload   string     `gorm:"payload" json:"payload"`
	Method    string     `gorm:"method" json:"method"`
	Response  string     `gorm:"response" json:"response"`
	Status    int        `gorm:"status" json:"status"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

type DmedGerPrescripReq struct {
	PatientId string `json:"patient_id"`
	SafeCode  string `json:"safe_code"`
	Url       string `json:"url"`
}

type DmedGiveReceiptReq struct {
	DrugAmount       int    `json:"drug_amount"`
	Price            int    `json:"price"`
	IssuedByFullName string `json:"issued_by_full_name"`
	MarkingCode      string `json:"marking_code,omitempty"`
	Gtin             string `json:"gtin,omitempty"`
	SerialNumber     string `json:"serial_number,omitempty"`
}

type DmedGeneral[T any] struct {
	Request T
}

type DmedResponseWrapper[T any] struct {
	Data    T      `json:"data"`
	Message string `json:"message"`
}

func (p *DmedResponseWrapper[T]) FormatResponseError(code int, message string) {
	p.Message = message
}

type DmedResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
