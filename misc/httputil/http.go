package httputil

import (
	"GoMusic/internal/musicurl"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const MaxResponseBytes = 16 << 20

type StatusError struct{ Code int }

func (e *StatusError) Error() string { return fmt.Sprintf("音乐平台返回 HTTP %d", e.Code) }

type Client struct{ http *http.Client }

func NewClient(client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	copy := *client
	if copy.Timeout <= 0 {
		copy.Timeout = 12 * time.Second
	}
	copy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 6 {
			return fmt.Errorf("音乐分享链接跳转次数过多")
		}
		provider, err := musicurl.Validate(req.URL)
		if err != nil {
			return err
		}
		initial, err := musicurl.Validate(via[0].URL)
		if err != nil || initial != provider {
			return fmt.Errorf("音乐分享链接跳转到了其他平台")
		}
		return nil
	}
	return &Client{http: &copy}
}

func (c *Client) request(ctx context.Context, method, address string, body io.Reader, headers map[string]string) (*http.Response, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	if _, err = musicurl.Validate(u); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 GoMusic/2.0")
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	return c.http.Do(req)
}

// Resolve reads only redirect headers, avoiding a full share-page download just to extract its ID.
func (c *Client) Resolve(ctx context.Context, address string, headers map[string]string) (*url.URL, error) {
	resp, err := c.request(ctx, http.MethodGet, address, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{Code: resp.StatusCode}
	}
	return resp.Request.URL, nil
}

// Do applies a deadline, bounds the response size, and closes every response body.
func (c *Client) Do(ctx context.Context, method, address string, body io.Reader, headers map[string]string) ([]byte, *url.URL, error) {
	data, _, finalURL, err := c.DoWithCookies(ctx, method, address, body, headers)
	return data, finalURL, err
}

// DoWithCookies returns cookies set by a successful response while retaining
// the same host, timeout, redirect, body-size, and body-close protections as Do.
func (c *Client) DoWithCookies(ctx context.Context, method, address string, body io.Reader, headers map[string]string) ([]byte, []*http.Cookie, *url.URL, error) {
	resp, err := c.request(ctx, method, address, body, headers)
	if err != nil {
		return nil, nil, nil, err
	}
	defer resp.Body.Close()
	cookies := resp.Cookies()
	if resp.StatusCode != http.StatusOK {
		return nil, cookies, resp.Request.URL, &StatusError{Code: resp.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	if err != nil {
		return nil, cookies, resp.Request.URL, err
	}
	if len(data) > MaxResponseBytes {
		return nil, cookies, resp.Request.URL, fmt.Errorf("音乐平台响应超过大小限制")
	}
	return data, cookies, resp.Request.URL, nil
}
