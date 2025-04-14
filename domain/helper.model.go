package domain

import "time"

// Product Measurenment
type ProductMeasurement struct {
	ID         *string    `gorm:"id" json:"id"`
	MxikCode   string     `gorm:"mxik_code" json:"mxik_code"`
	MxikNameUz string     `gorm:"mxik_name_uz" json:"mxik_name_uz"`
	MxikNameRu string     `gorm:"mxik_name_ru" json:"mxik_name_ru"`
	UnitName   string     `gorm:"unit_name" json:"unit_name"`
	UnitCode   string     `gorm:"unit_code" json:"unit_code"`
	CreatedAt  *time.Time `gorm:"created_at" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"updated_at" json:"updated_at"`
}

type SoliqResponse struct {
	Success string              `json:"success"`
	Code    int                 `json:"code"`
	Reason  string              `json:"reason"`
	Data    []SoliqIKPUResponse `json:"data"`
}

type SoliqIKPUResponse struct {
	MxikCode string `json:"mxikCode"`
	Name     string `json:"name"`
	Units    string `json:"units"`
}

// {
//     "info": {
//         "dateTime": "20250414170342",
//         "qrCodeURL": "https://ofd.soliq.uz/check?t=VG343420000976&r=1205&c=20250414170342&s=127401140452",
//         "fiscalSign": "127401140452",
//         "receiptSeq": "1205",
//         "terminalId": "VG343420000976"
//     },
//     "error": false,
//     "qrPath": "C:\\Users\\user\\.EPOS\\qrs\\1205.bmp",
//     "paycheck": "",
//     "virtualNumber": "7645"
// }

// Epost response data
type EposResponseInfo struct {
	Error         bool                  `json:"error"`
	QrPath        string                `json:"qrPath"`
	Paycheck      string                `json:"paycheck"`
	VirtualNumber string                `json:"virtualNumber"`
	Info          EposResponseInfoParam `json:"info"`
}

type EposResponseInfoParam struct {
	Datetime   string `json:"dateTime"`
	QrCodeURL  string `json:"qrCodeURL"`
	FiscalSign string `json:"fiscalSign"`
	ReceiptSeq string `json:"receiptSeq"`
	TerminalId string `json:"terminalId"`
}
