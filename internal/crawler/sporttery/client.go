package sporttery

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	wafMarkers = []string{"WAF拦截", "禁止访问", "web应用防护", "请求已中断"}
)

type ErrWAFBlocked struct {
	URL string
}

func (e *ErrWAFBlocked) Error() string {
	return fmt.Sprintf("blocked by WAF for %s", e.URL)
}

type Client struct {
	httpClient *http.Client
	userAgent  string
	referer    string
}

func NewClient(userAgent, referer string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:    90 * time.Second,
				DisableCompression:  false,
			},
		},
		userAgent: userAgent,
		referer:   referer,
	}
}

func (c *Client) Get(url string) ([]byte, error) {
	return c.doRequest("GET", url, nil)
}

func (c *Client) doRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", c.referer)
	req.Header.Set("Origin", c.referer)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, &ErrWAFBlocked{URL: url}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d for %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if isWAFBlocked(data) {
		return nil, &ErrWAFBlocked{URL: url}
	}

	return data, nil
}

func isWAFBlocked(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	s := string(body)
	for _, marker := range wafMarkers {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}
