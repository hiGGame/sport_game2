package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type OfficialClient struct {
	appID      string
	secret     string
	httpClient *http.Client
}

func NewOfficialClient(appID, secret string) *OfficialClient {
	return &OfficialClient{
		appID:      appID,
		secret:     secret,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type OAuth2AccessTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type UserInfoResp struct {
	OpenID     string `json:"openid"`
	Nickname   string `json:"nickname"`
	Sex        int    `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	HeadImgURL string `json:"headimgurl"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func (c *OfficialClient) GetOAuth2AccessToken(code string) (*OAuth2AccessTokenResp, error) {
	if c.appID == "" || c.secret == "" {
		return nil, fmt.Errorf("wechat official appid or secret not configured")
	}

	u := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		url.QueryEscape(c.appID), url.QueryEscape(c.secret), url.QueryEscape(code))

	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("wechat oauth2 access_token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read wechat response: %w", err)
	}

	var result OAuth2AccessTokenResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse wechat response: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error: %d %s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}

func (c *OfficialClient) GetUserInfo(accessToken, openID string) (*UserInfoResp, error) {
	if accessToken == "" || openID == "" {
		return nil, fmt.Errorf("access_token or openid is empty")
	}

	u := fmt.Sprintf("https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		url.QueryEscape(accessToken), url.QueryEscape(openID))

	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("wechat userinfo: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read wechat response: %w", err)
	}

	var result UserInfoResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse wechat response: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wechat error: %d %s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}
