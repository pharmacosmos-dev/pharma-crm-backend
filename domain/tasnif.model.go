package domain

type TasnifResponse struct {
	MxikCode  string         `json:"mxikCode"`
	GroupName string         `json:"groupName"`
	GroupCode string         `json:"groupCode"`
	ClassName string         `json:"className"`
	ClassCode string         `json:"classCode"`
	MxikName  string         `json:"mxikName"`
	Packages  []PackageCodes `json:"packages"`
}

type PackageCodes struct {
	Code        int     `json:"code"`
	MxikCode    string  `json:"mxikCode"`
	Name        string  `json:"name"`
	ParentValue float64 `json:"parentValue"`
}
