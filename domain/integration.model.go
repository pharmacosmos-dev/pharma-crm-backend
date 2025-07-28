package domain

import "time"

type Request1C struct {
	ID        *string    `gorm:"type:uuid;default:gen_random_uuid()" json:"id"`
	Method    string     `gorm:"method" json:"method"`
	Payload   []byte     `gorm:"payload" json:"payload"`
	Response  []byte     `gorm:"response" json:"response"`
	Action    string     `gorm:"action" json:"action"`
	DocDate   string     `gorm:"doc_date" json:"doc_date"`
	DocNum    string     `gorm:"doc_num" json:"doc_num"`
	Status    string     `gorm:"status" json:"status"`
	CreatedAt *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt *time.Time `gorm:"updated_at" json:"updated_at"`
}

type PrescriptionResponse struct {
	Data []Prescription `json:"data"`
}

type Prescription struct {
	ID               int             `json:"id"`
	Number           string          `json:"number"`
	SafeCode         string          `json:"safe_code"`
	Status           string          `json:"status"`
	Type             string          `json:"type"`
	DistributionType string          `json:"distribution_type"`
	ExpirationDate   string          `json:"expiration_date"`
	Price            *float64        `json:"price"`
	CreatedAt        string          `json:"created_at"`
	DrugAppointment  DrugAppointment `json:"drug_appointment"`
}

type DrugAppointment struct {
	ID                        int        `json:"id"`
	SingleDose                string     `json:"single_dose"`
	AdministrationFrequency   int        `json:"administration_frequency"`
	AdministrationFreqMeasure string     `json:"administration_frequency_measure"`
	CountDays                 int        `json:"count_days"`
	TotalAmount               int        `json:"total_amount"`
	Medication                Medication `json:"medication"`
	ATCs                      []ATC      `json:"atcs"`
	RemainingCount            int        `json:"remaining_count"`
	DosageForm                TitleOnly  `json:"dosage_form"`
	DrugAdministrationRoute   TitleOnly  `json:"drug_administration_route"`
	DrugAdministrationTimes   []any      `json:"drug_administration_times"` // or create struct if needed
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
