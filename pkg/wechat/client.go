package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	appID     string
	secret    string
	httpClient *http.Client
}

func NewClient(appID, secret string) *Client {
	return &Client{
		appID:     appID,
		secret:    secret,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type Code2SessionResp struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func (c *Client) Code2Session(code string) (*Code2SessionResp, error) {
	if c.appID == "" || c.secret == "" {
		return nil, fmt.Errorf("wechat appid or secret not configured")
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		c.appID, c.secret, code)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("wechat code2session: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read wechat response: %w", err)
	}

	var result Code2SessionResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse wechat response: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error: %d %s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}
