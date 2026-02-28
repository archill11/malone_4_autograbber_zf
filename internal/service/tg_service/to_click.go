package tg_service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CreateLinkResponse struct {
	Data struct{
		Attributes struct{
			FullUrl string `json:"full_url"`
		} `json:"attributes"`
	} `json:"data"`

	CreateLinkErrResponse
}

type CreateLinkErrResponse struct {
	Error    string `json:"error"`
}

func ToClick_CreateShortLink(originalURL string, apiKey string) (string, error) {

	jsonStr := `{"data":{"type":"link", "attributes":{"web_url":"` + originalURL + `"}}}`

	jsonData := []byte(jsonStr)

	req, err := http.NewRequest(
		"POST",
		"https://to.click/api/v1/links",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("CreateLink NewRequest err: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", fmt.Sprintf("%v", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("CreateLink Do req err: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("CreateLink ReadAll err: %v", err)
	}

	var result CreateLinkResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("CreateLink Unmarshal err: %v", err)
	}

	if result.CreateLinkErrResponse.Error != "" {
		return "", fmt.Errorf("CreateLink Resp err: %+v", result.CreateLinkErrResponse.Error)
	}

	return result.Data.Attributes.FullUrl, nil
}