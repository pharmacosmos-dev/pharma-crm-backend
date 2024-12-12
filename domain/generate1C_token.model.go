package domain

type Generate1CTokenRequest struct {
	Password string `json:"password" validate:"required"`
}

type Generate1CTokenResponse struct {
	Token string `json:"token"`
}
