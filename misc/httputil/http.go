package httputil

import (
	"io"
	"net/http"

	"GoMusic/misc/log"
	"GoMusic/misc/models"
)

const (
	user_agent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0"
	referer    = "https://github.com/"
)
// not allow redirect client
var client *http.Client

// allow redirect client
var clientNoRedirect *http.Client

func init() {
	client = &http.Client{}
	clientNoRedirect = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // return this error to prevent redirect
		},
	}
}

// Post ...
func Post(link string, data io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(models.POST, link, data)
	if err != nil {
		log.Errorf("http NewRequest error: %+v", err)
		return nil, err
	}
	req.Header.Add("User-Agent", user_agent)
	req.Header.Add("Referer", referer)
	req.Header.Add(models.ContentType, "application/x-www-form-urlencoded")
	return client.Do(req)
}

// GetRedirectLocation ...
func GetRedirectLocation(link string) (string, error) {
	rsp, err := clientNoRedirect.Get(link)
	if err != nil {
		log.Errorf("http Get error: %+v", err)
		return "", err
	}
	return rsp.Header.Get("Location"), nil
}
func Get(link string, data io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("GET", link, data)
	if err != nil {
		log.Errorf("http NewRequest error: %+v", err)
		return nil, err
	}
	req.Header.Add("User-Agent", user_agent)
	req.Header.Add("Referer", referer)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return client.Do(req)
}
