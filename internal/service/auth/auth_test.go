package auth

import (
	"errors"
	"testing"

	"sport_game2/internal/repo"
	"sport_game2/pkg/wechat"
)

type fakeUserRepo struct {
	getUser    *repo.User
	getErr     error
	createUser *repo.User
	createErr  error
}

func (f *fakeUserRepo) GetUserByOpenID(openID string) (*repo.User, error) {
	return f.getUser, f.getErr
}

func (f *fakeUserRepo) CreateUser(openID, unionID string, credits int) (*repo.User, error) {
	return f.createUser, f.createErr
}

func (f *fakeUserRepo) GetUserByID(id int64) (*repo.User, error) {
	return f.getUser, f.getErr
}

func (f *fakeUserRepo) UpdateUserProfile(id int64, nickname, avatarURL string) error {
	return nil
}

type fakeWechatClient struct {
	session *wechat.Code2SessionResp
	err     error
}

func (f *fakeWechatClient) Code2Session(code string) (*wechat.Code2SessionResp, error) {
	return f.session, f.err
}

type fakeOfficialWechatClient struct {
	tokenResp *wechat.OAuth2AccessTokenResp
	tokenErr  error
	userInfo  *wechat.UserInfoResp
	userErr   error
}

func (f *fakeOfficialWechatClient) GetOAuth2AccessToken(code string) (*wechat.OAuth2AccessTokenResp, error) {
	return f.tokenResp, f.tokenErr
}

func (f *fakeOfficialWechatClient) GetUserInfo(accessToken, openID string) (*wechat.UserInfoResp, error) {
	return f.userInfo, f.userErr
}

type fakeTokenManager struct {
	token string
	err   error
}

func (f *fakeTokenManager) Generate(userID int64, openID string) (string, error) {
	return f.token, f.err
}

func TestLoginNewUser(t *testing.T) {
	wechatClient := &fakeWechatClient{
		session: &wechat.Code2SessionResp{OpenID: "test_openid", UnionID: "test_unionid"},
	}
	userRepo := &fakeUserRepo{
		getUser: nil,
		getErr:  nil,
		createUser: &repo.User{ID: 1, OpenID: "test_openid", Nickname: "", AvatarURL: ""},
		createErr:  nil,
	}
	tokenMgr := &fakeTokenManager{token: "test_token", err: nil}

	svc := NewService(userRepo, wechatClient, nil, tokenMgr, 1000)

	resp, err := svc.Login(&LoginRequest{Code: "valid_code"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Token != "test_token" {
		t.Errorf("expected token test_token, got %s", resp.Token)
	}
	if resp.OpenID != "test_openid" {
		t.Errorf("expected openId test_openid, got %s", resp.OpenID)
	}
}

func TestLoginExistingUser(t *testing.T) {
	wechatClient := &fakeWechatClient{
		session: &wechat.Code2SessionResp{OpenID: "existing_openid"},
	}
	userRepo := &fakeUserRepo{
		getUser: &repo.User{ID: 2, OpenID: "existing_openid", Nickname: "TestUser", AvatarURL: "http://avatar"},
		getErr:  nil,
	}
	tokenMgr := &fakeTokenManager{token: "existing_token", err: nil}

	svc := NewService(userRepo, wechatClient, nil, tokenMgr, 1000)

	resp, err := svc.Login(&LoginRequest{Code: "valid_code"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Token != "existing_token" {
		t.Errorf("expected existing token, got %s", resp.Token)
	}
	if resp.WechatNickname != "TestUser" {
		t.Errorf("expected nickname TestUser, got %s", resp.WechatNickname)
	}
	if resp.WechatAvatar != "http://avatar" {
		t.Errorf("expected avatar, got %s", resp.WechatAvatar)
	}
}

func TestLoginWechatError(t *testing.T) {
	wechatClient := &fakeWechatClient{
		err: errors.New("wechat api error"),
	}
	userRepo := &fakeUserRepo{}
	tokenMgr := &fakeTokenManager{}

	svc := NewService(userRepo, wechatClient, nil, tokenMgr, 1000)

	_, err := svc.Login(&LoginRequest{Code: "bad_code"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoginCreateUserError(t *testing.T) {
	wechatClient := &fakeWechatClient{
		session: &wechat.Code2SessionResp{OpenID: "new_openid"},
	}
	userRepo := &fakeUserRepo{
		getUser:   nil,
		getErr:    nil,
		createErr: errors.New("db error"),
	}
	tokenMgr := &fakeTokenManager{}

	svc := NewService(userRepo, wechatClient, nil, tokenMgr, 1000)

	_, err := svc.Login(&LoginRequest{Code: "valid_code"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetUser(t *testing.T) {
	userRepo := &fakeUserRepo{
		getUser: &repo.User{ID: 1, OpenID: "test"},
	}
	svc := NewService(userRepo, nil, nil, nil, 0)

	user, err := svc.GetUser(1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.ID != 1 {
		t.Errorf("expected id 1, got %d", user.ID)
	}
}

func TestLoginByOfficialNewUser(t *testing.T) {
	officialClient := &fakeOfficialWechatClient{
		tokenResp: &wechat.OAuth2AccessTokenResp{OpenID: "official_openid", UnionID: "official_unionid", AccessToken: "access_token"},
		userInfo:  &wechat.UserInfoResp{OpenID: "official_openid", Nickname: "OfficialUser", HeadImgURL: "http://avatar.png"},
	}
	userRepo := &fakeUserRepo{
		getUser:    nil,
		createUser: &repo.User{ID: 10, OpenID: "official_openid"},
	}
	tokenMgr := &fakeTokenManager{token: "official_token"}

	svc := NewService(userRepo, nil, officialClient, tokenMgr, 1000)

	resp, err := svc.LoginByOfficial("valid_code")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Token != "official_token" {
		t.Errorf("expected token official_token, got %s", resp.Token)
	}
	if resp.OpenID != "official_openid" {
		t.Errorf("expected openId official_openid, got %s", resp.OpenID)
	}
	if resp.WechatNickname != "OfficialUser" {
		t.Errorf("expected nickname OfficialUser, got %s", resp.WechatNickname)
	}
	if resp.WechatAvatar != "http://avatar.png" {
		t.Errorf("expected avatar http://avatar.png, got %s", resp.WechatAvatar)
	}
}

func TestLoginByOfficialExistingUser(t *testing.T) {
	officialClient := &fakeOfficialWechatClient{
		tokenResp: &wechat.OAuth2AccessTokenResp{OpenID: "existing_official_openid", AccessToken: "access_token"},
		userInfo:  &wechat.UserInfoResp{OpenID: "existing_official_openid", Nickname: "Existing", HeadImgURL: "http://old.png"},
	}
	userRepo := &fakeUserRepo{
		getUser: &repo.User{ID: 5, OpenID: "existing_official_openid", Nickname: "Existing", AvatarURL: "http://old.png"},
	}
	tokenMgr := &fakeTokenManager{token: "existing_official_token"}

	svc := NewService(userRepo, nil, officialClient, tokenMgr, 1000)

	resp, err := svc.LoginByOfficial("valid_code")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Token != "existing_official_token" {
		t.Errorf("expected token existing_official_token, got %s", resp.Token)
	}
	if resp.NeedProfile {
		t.Errorf("expected needProfile false for existing user with profile")
	}
}

func TestLoginByOfficialOAuthError(t *testing.T) {
	officialClient := &fakeOfficialWechatClient{
		tokenErr: errors.New("oauth api error"),
	}
	userRepo := &fakeUserRepo{}
	tokenMgr := &fakeTokenManager{}

	svc := NewService(userRepo, nil, officialClient, tokenMgr, 1000)

	_, err := svc.LoginByOfficial("bad_code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoginByOfficialUserInfoError(t *testing.T) {
	officialClient := &fakeOfficialWechatClient{
		tokenResp: &wechat.OAuth2AccessTokenResp{OpenID: "official_openid", AccessToken: "access_token"},
		userErr:   errors.New("userinfo api error"),
	}
	userRepo := &fakeUserRepo{}
	tokenMgr := &fakeTokenManager{}

	svc := NewService(userRepo, nil, officialClient, tokenMgr, 1000)

	_, err := svc.LoginByOfficial("valid_code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
