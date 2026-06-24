package auth

import (
	"fmt"

	"sport_game2/internal/repo"
	"sport_game2/pkg/wechat"
)

type userStore interface {
	GetUserByOpenID(openID string) (*repo.User, error)
	CreateUser(openID, unionID string, initialCredits int) (*repo.User, error)
	GetUserByID(id int64) (*repo.User, error)
	UpdateUserProfile(id int64, nickname, avatarURL string) error
}

type tokenManager interface {
	Generate(userID int64, openID string) (string, error)
}

type wechatClient interface {
	Code2Session(code string) (*wechat.Code2SessionResp, error)
}

type Service struct {
	userRepo       userStore
	wechat         wechatClient
	tokenManager   tokenManager
	initialCredits int
}

func NewService(userRepo userStore, wechat wechatClient, tokenManager tokenManager, initialCredits int) *Service {
	return &Service{
		userRepo:       userRepo,
		wechat:         wechat,
		tokenManager:   tokenManager,
		initialCredits: initialCredits,
	}
}

type LoginRequest struct {
	AppID  string `json:"appid"`
	Code   string `json:"code"`
	State  string `json:"state"`
}

type LoginResponse struct {
	Token          string `json:"token"`
	WechatNickname string `json:"wechatNickname"`
	WechatAvatar   string `json:"wechatAvatar"`
	OpenID         string `json:"openId"`
	NeedProfile    bool   `json:"needProfile"`
}

type Shop struct {
	MerchantID string `json:"merchantId"`
	ShopName   string `json:"shopName"`
}

func (s *Service) Login(req *LoginRequest) (*LoginResponse, error) {
	session, err := s.wechat.Code2Session(req.Code)
	if err != nil {
		return nil, fmt.Errorf("wechat login: %w", err)
	}

	user, err := s.userRepo.GetUserByOpenID(session.OpenID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if user == nil {
		user, err = s.userRepo.CreateUser(session.OpenID, session.UnionID, s.initialCredits)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
	}

	token, err := s.tokenManager.Generate(user.ID, user.OpenID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &LoginResponse{
		Token:          token,
		WechatNickname: user.Nickname,
		WechatAvatar:   user.AvatarURL,
		OpenID:         session.OpenID,
		NeedProfile:    user.Nickname == "" && user.AvatarURL == "",
	}, nil
}

func (s *Service) GetUser(userID int64) (*repo.User, error) {
	return s.userRepo.GetUserByID(userID)
}

func (s *Service) UpdateProfile(userID int64, nickname, avatarURL string) error {
	return s.userRepo.UpdateUserProfile(userID, nickname, avatarURL)
}

type DevLoginRequest struct {
	UserNum int `json:"userNum"`
}

func (s *Service) DevLogin(req *DevLoginRequest) (*LoginResponse, error) {
	if req.UserNum <= 0 {
		req.UserNum = 1
	}
	openID := fmt.Sprintf("dev_user_%d", req.UserNum)

	user, err := s.userRepo.GetUserByOpenID(openID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		user, err = s.userRepo.CreateUser(openID, "", s.initialCredits)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
	}

	token, err := s.tokenManager.Generate(user.ID, user.OpenID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &LoginResponse{
		Token:          token,
		WechatNickname: user.Nickname,
		WechatAvatar:   user.AvatarURL,
		OpenID:         openID,
		NeedProfile:    user.Nickname == "" && user.AvatarURL == "",
	}, nil
}
