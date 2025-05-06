package domain

type TasnifResponse struct {
	MxikCode  string        `json:"mxikCode"`
	GroupName string        `json:"groupName"`
	GroupCode string        `json:"groupCode"`
	ClassName string        `json:"className"`
	ClassCode string        `json:"classCode"`
	MxikName  string        `json:"mxikName"`
	Packages  []MxikPackage `json:"packages"`
}

type PackageCodes struct {
	Code        int     `json:"code"`
	MxikCode    string  `json:"mxikCode"`
	Name        string  `json:"name"`
	ParentValue float64 `json:"parentValue"`
}

type MxikPackage struct {
	MXIK      string  `gorm:"mxik" json:"mxikCode"`
	UnitCode  int     `gorm:"unit_code" json:"code"`
	UnitLabel string  `gorm:"unit_label" json:"name"`
	UnitCount float64 `gorm:"unit_count" json:"parentValue"`
}
