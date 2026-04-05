package tg_service

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func (srv *TgService) DownloadFile(filepath, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("DownloadFile Create filepath-%s err: %v", filepath, err)
	}
	defer out.Close()
	// Get the data
	resp, err := srv.MyHttpGet(url)
	if err != nil {
		return fmt.Errorf("DownloadFile Get url-%s err: %v", url, err)
	}
	defer resp.Body.Close()
	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DownloadFile Get url-%s err: bad status: %s", url, resp.Status)
	}
	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("DownloadFile Copy err: %v", err)
	}
	return nil
}