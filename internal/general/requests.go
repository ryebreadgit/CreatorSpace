package general

import (
	"fmt"
	"io"
	"net/http"
)

func GetAPIResponse(filepath string, url string) (string, error) {

	// Get the data
	resp, err := http.Get(url)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("invalid response code %d", resp.StatusCode)
	} else {
		responseData, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(responseData), nil
	}
}
