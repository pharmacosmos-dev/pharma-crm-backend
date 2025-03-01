package domain

type PackageCodeResponse struct {
	Error     bool       `json:"error"`
	ClassCode string     `json:"classCode"`
	Packages  []Packages `json:"packages"`
}

type Packages struct {
	Code    string `json:"code"`
	NameLat string `json:"nameLat"`
}
