package httputil

import (
	"fmt"
	"io"
	"net/http"

	"GoMusic/misc/log"
	"GoMusic/misc/models"
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
	req.Header.Add(models.ContentType, "application/x-www-form-urlencoded")
	return client.Do(req)
}

// GetRedirectLocation 获取短链的跳转地址，并校验响应状态和 Location。
func GetRedirectLocation(link string) (string, error) {
	rsp, err := clientNoRedirect.Get(link)
	if err != nil {
		log.Errorf("http Get error: %+v", err)
		return "", err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode < http.StatusMultipleChoices || rsp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("短链请求返回 %s，未提供跳转地址", rsp.Status)
	}
	location := rsp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("短链请求返回 %s，但 Location 为空", rsp.Status)
	}
	return location, nil
}

func Get(link string, data io.Reader) (*http.Response, error) {
	return GetWithHeaders(link, data, nil)
}

// GetWithHeaders 发送带自定义请求头的 GET 请求。
func GetWithHeaders(link string, data io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, link, data)
	if err != nil {
		log.Errorf("http NewRequest error: %+v", err)
		return nil, err
	}
	req.Header.Add(models.ContentType, "application/x-www-form-urlencoded")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return client.Do(req)
}
