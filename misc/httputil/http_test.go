package httputil

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type observedBody struct {
	io.Reader
	closed bool
}

func (b *observedBody) Close() error { b.closed = true; return nil }
func TestRedirectHostValidationAndBodyClosure(t *testing.T) {
	for _, location := range []string{"http://127.0.0.1/private", "https://evil.test/x", "https://qishui.douyin.com/s/x/"} {
		calls := 0
		body := &observedBody{Reader: strings.NewReader("redirect")}
		client := NewClient(&http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 302, Body: body, Header: http.Header{"Location": []string{location}}, Request: r}, nil
		})})
		_, _, err := client.Do(context.Background(), "GET", "https://c6.y.qq.com/base/fcgi-bin/u?__=x", nil, nil)
		if err == nil || calls != 1 || !body.closed {
			t.Fatalf("redirect accepted/leaked: %s err=%v calls=%d closed=%v", location, err, calls, body.closed)
		}
	}
}
func TestResponseLimitAndHTTPStatus(t *testing.T) {
	for _, tt := range []struct {
		status int
		body   string
	}{{200, strings.Repeat("x", MaxResponseBytes+1)}, {403, "forbidden"}} {
		body := &observedBody{Reader: strings.NewReader(tt.body)}
		client := NewClient(&http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: tt.status, Body: body, Header: make(http.Header), Request: r}, nil
		})})
		if _, _, err := client.Do(context.Background(), "GET", "https://music.163.com/api/test", nil, nil); err == nil {
			t.Fatal("bad response accepted")
		}
		if !body.closed {
			t.Fatal("body leaked")
		}
	}
}
func TestRelativeRedirect(t *testing.T) {
	calls := 0
	client := NewClient(&http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return &http.Response{StatusCode: 302, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{"Location": []string{"/n/ryqq/playlist/1"}}, Request: r}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header), Request: r}, nil
	})})
	_, final, err := client.Do(context.Background(), "GET", "https://c6.y.qq.com/start", nil, nil)
	if err != nil || final.Path != "/n/ryqq/playlist/1" {
		t.Fatalf("relative redirect failed: %v %v", final, err)
	}
}

type unreadBody struct {
	read   bool
	closed bool
}

func (b *unreadBody) Read(_ []byte) (int, error) { b.read = true; return 0, io.EOF }
func (b *unreadBody) Close() error               { b.closed = true; return nil }
func TestResolveClosesWithoutDownloadingSharePage(t *testing.T) {
	body := &unreadBody{}
	client := NewClient(&http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header), Request: r}, nil
	})})
	final, err := client.Resolve(context.Background(), "https://y.qq.com/n/ryqq/playlist/1", nil)
	if err != nil || final == nil || body.read || !body.closed {
		t.Fatalf("err=%v read=%v closed=%v", err, body.read, body.closed)
	}
}
