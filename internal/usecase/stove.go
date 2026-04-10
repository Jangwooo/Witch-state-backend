package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

// StoveUseCase Stove 인증 전용 UseCase
type StoveUseCase interface {
	SignInWithStove(ctx context.Context, accessToken string) (*entity.SessionResponse, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.UserResponse, error)
}

type stoveUseCase struct {
	userRepo     repository.UserRepository
	sessionStore session.SessionStore
	httpClient   *http.Client
	apiBase      string
	serviceID    string
	clientID     string
	clientSecret string
	tokenMu      sync.Mutex
	tokenValue   string
	tokenExpiry  time.Time
}

// NewStoveUseCase Stove UseCase 생성자
func NewStoveUseCase(userRepo repository.UserRepository, sessionStore session.SessionStore) StoveUseCase {
	return &stoveUseCase{
		userRepo:     userRepo,
		sessionStore: sessionStore,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiBase:      strings.TrimRight(getEnv("STOVE_API_BASE", "https://api.onstove.com"), "/"),
		serviceID:    viper.GetString("STOVE_SERVICE_ID"),
		clientID:     viper.GetString("STOVE_CLIENT_ID"),
		clientSecret: viper.GetString("STOVE_CLIENT_SECRET"),
	}
}

// SignInWithStove 유저 검증 및 세션 생성
func (u *stoveUseCase) SignInWithStove(ctx context.Context, accessToken string) (*entity.SessionResponse, error) {
	if accessToken == "" {
		return nil, newAuthError("access_token is required")
	}
	if u.serviceID == "" || u.clientID == "" || u.clientSecret == "" {
		return nil, fmt.Errorf("stove auth config missing")
	}

	verifyResp, err := u.verifyAccessToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	memberNo := strconv.FormatInt(verifyResp.Value.MemberNo, 10)

	// STOVE API에서 사용자 닉네임 조회
	memberInfo, err := u.getMemberInfo(ctx, verifyResp.Value.MemberNo)
	displayName := "stove_" + memberNo
	if err == nil && memberInfo.Value.Nickname != "" {
		displayName = memberInfo.Value.Nickname
	}

	user, err := u.userRepo.FindByPlatformUserID(ctx, "stove", memberNo)
	if err != nil {
		if ent.IsNotFound(err) {
			platformData := map[string]interface{}{
				"stove_member_no": verifyResp.Value.MemberNo,
			}
			if verifyResp.Value.Guid != 0 {
				platformData["stove_guid"] = verifyResp.Value.Guid
			}

			user, err = u.userRepo.Create(ctx, &entity.CreateUserRequest{
				PlatformType:        "stove",
				PlatformUserID:      memberNo,
				PlatformDisplayName: displayName,
				IsVerified:          true,
				Nickname:            displayName,
				PlatformData:        platformData,
			})

			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	sid, err := u.sessionStore.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	return user.ToSessionResponse(sid), nil
}

// FindByID ID로 사용자 조회
func (u *stoveUseCase) FindByID(ctx context.Context, id uuid.UUID) (*entity.UserResponse, error) {
	user, err := u.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user.ToResponse(), nil
}

type stoveTokenVerifyResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Value   struct {
		MemberNo int64 `json:"member_no"`
		Guid     int64 `json:"guid"`
	} `json:"value"`
}

type stoveMemberInfoResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Value   struct {
		MemberNo int64  `json:"member_no"`
		Nickname string `json:"nickname"`
	} `json:"value"`
}

func (u *stoveUseCase) verifyAccessToken(ctx context.Context, accessToken string) (*stoveTokenVerifyResponse, error) {
	payload := struct {
		AccessToken string `json:"access_token"`
	}{
		AccessToken: accessToken,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	serverToken, err := u.getServerAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/member/v3.0/%s/token/verify", u.apiBase, u.serviceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+serverToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stove token verify failed: status %d", resp.StatusCode)
	}

	var verifyResp stoveTokenVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return nil, err
	}

	if verifyResp.Code != 0 {
		return nil, newAuthError(verifyResp.Message)
	}

	if verifyResp.Value.MemberNo == 0 {
		return nil, newAuthError("invalid stove token")
	}

	return &verifyResp, nil
}

func (u *stoveUseCase) getMemberInfo(ctx context.Context, memberNo int64) (*stoveMemberInfoResponse, error) {
	serverToken, err := u.getServerAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/member/v3.0/%s?member_no=%d", u.apiBase, u.serviceID, memberNo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+serverToken)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stove member info failed: status %d", resp.StatusCode)
	}

	var memberResp stoveMemberInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&memberResp); err != nil {
		return nil, err
	}

	if memberResp.Code != 0 {
		return nil, fmt.Errorf("stove member info failed: %s", memberResp.Message)
	}

	return &memberResp, nil
}

type stoveServerTokenResponse struct {
	Code         int    `json:"code"`
	Message      string `json:"message"`
	ResponseData struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	} `json:"response_data"`
}

func (u *stoveUseCase) getServerAccessToken(ctx context.Context) (string, error) {
	u.tokenMu.Lock()
	defer u.tokenMu.Unlock()

	if u.tokenValue != "" && time.Now().Before(u.tokenExpiry) {
		return u.tokenValue, nil
	}

	payload := struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		ServiceID    string `json:"service_id"`
	}{
		ClientID:     u.clientID,
		ClientSecret: u.clientSecret,
		ServiceID:    u.serviceID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/auth/v5/server_token", u.apiBase)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("stove server token failed: status %d", resp.StatusCode)
	}

	var tokenResp stoveServerTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	if tokenResp.Code != 0 {
		return "", fmt.Errorf("stove server token failed: %s", tokenResp.Message)
	}

	if tokenResp.ResponseData.AccessToken == "" {
		return "", fmt.Errorf("stove server token missing access_token")
	}

	expiresIn := time.Duration(tokenResp.ResponseData.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = 30 * 24 * time.Hour
	}

	refreshBefore := 5 * time.Minute
	expiry := time.Now().Add(expiresIn - refreshBefore)
	if expiresIn <= refreshBefore {
		expiry = time.Now().Add(expiresIn)
	}

	u.tokenValue = tokenResp.ResponseData.AccessToken
	u.tokenExpiry = expiry

	return u.tokenValue, nil
}
