package api

type Account struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
