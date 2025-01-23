package httputil

import (
	"io"
	"net/http"

	"GoMusic/misc/log"
)

const (
	user_agent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0"
	referer    = "https://github.com/"
)

var client *http.Client
var clientNoRedirect *http.Client

func init() {
	client = &http.Client{}
	clientNoRedirect = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 返回此错误阻止重定向
		},
	}
}

func Post(link string, data io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", link, data)
	if err != nil {
		log.Errorf("http NewRequest error: %+v", err)
		return nil, err
	}
	req.Header.Add("User-Agent", user_agent)
	req.Header.Add("Referer", referer)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return client.Do(req)
}

func GetRedirectLocation(link string) (string, error) {
	resp, err := clientNoRedirect.Get(link)
	if err != nil {
		return "", err
	}
	return resp.Header.Get("Location"), nil
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
