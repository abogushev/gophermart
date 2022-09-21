package api

type Account struct {
	Current   int64 `json:"current"`
	Withdrawn int64 `json:"withdrawn"`
}
