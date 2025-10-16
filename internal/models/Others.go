package models

type CreateShortLinkResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Link    string `json:"link"`
}