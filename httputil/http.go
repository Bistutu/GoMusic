package httputil

import (
	"io"
	"net/http"
)

const (
	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"
)

var client *http.Client

func init() {
	client = &http.Client{}
}

func Post(link string, data io.Reader) (*http.Response, error) {
	req, _ := http.NewRequest("POST", link, data)
	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return client.Do(req)
}
