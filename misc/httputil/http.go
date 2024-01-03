package httputil

import (
	"io"
	"net/http"

	"GoMusic/misc/log"
)

const (
	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36"
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
	//req.Header.Add("User-Agent", UserAgent)
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
