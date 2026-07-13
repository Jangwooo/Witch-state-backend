package usecase

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
	"github.com/witchs-lounge_backend/internal/infrastructure/hmacauth"
	"github.com/witchs-lounge_backend/internal/infrastructure/session"
)

// SteamUseCase Steam 인증 전용 UseCase
type SteamUseCase interface {
	SignInWithSteam(ctx context.Context, steamID string, ticket string) (*entity.SessionResponse, error)
}

type steamUseCase struct {
	userRepo         repository.UserRepository
	sessionStore     session.SessionStore
	hmacCfg          *hmacauth.Config
	httpClient       *http.Client
	apiBase          string
	apiKey           string
	appID            string
	appIDs           []string
	identity         string
	requireOwnership bool
}

// NewSteamUseCase Steam UseCase 생성자
func NewSteamUseCase(userRepo repository.UserRepository, sessionStore session.SessionStore, hmacCfg *hmacauth.Config) SteamUseCase {
	// 소유권 검사에 쓸 App ID 목록을 확정한다.
	// STEAM_APP_IDS(콤마 구분) 우선. 비어 있으면 기존 STEAM_APP_ID 단일값으로 폴백(하위호환).
	appIDs := parseAppIDs(viper.GetString("STEAM_APP_IDS"))
	appID := viper.GetString("STEAM_APP_ID")
	if len(appIDs) == 0 && appID != "" {
		appIDs = []string{appID}
	}
	// 티켓 검증(AuthenticateUserTicket)은 단일 appid만 받으므로 primary를 정한다.
	// STEAM_APP_ID가 있으면 그것을, 없으면 목록의 첫 값을 primary로 사용.
	if appID == "" && len(appIDs) > 0 {
		appID = appIDs[0]
	}

	return &steamUseCase{
		userRepo:     userRepo,
		sessionStore: sessionStore,
		hmacCfg:      hmacCfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiBase:          strings.TrimRight(getEnv("STEAM_WEB_API_BASE", "https://api.steampowered.com"), "/"),
		apiKey:           viper.GetString("STEAM_WEB_API_KEY"),
		appID:            appID,
		appIDs:           appIDs,
		identity:         viper.GetString("STEAM_TICKET_IDENTITY"),
		requireOwnership: strings.EqualFold(getEnv("STEAM_REQUIRE_OWNERSHIP", "true"), "true"),
	}
}

// parseAppIDs 콤마 구분 문자열을 App ID 슬라이스로 파싱한다. 공백/빈 항목은 제거.
func parseAppIDs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var ids []string
	for _, part := range strings.Split(raw, ",") {
		if id := strings.TrimSpace(part); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// SignInWithSteam 유저 검증 및 세션 생성
func (u *steamUseCase) SignInWithSteam(ctx context.Context, steamID string, ticket string) (*entity.SessionResponse, error) {
	if steamID == "" || ticket == "" {
		return nil, newAuthError("steam_id and ticket are required")
	}
	if u.apiKey == "" || u.appID == "" {
		return nil, fmt.Errorf("steam auth config missing")
	}

	normalizedTicket, err := normalizeSteamTicket(ticket)
	if err != nil {
		return nil, newAuthError("invalid steam ticket")
	}

	params, err := u.authenticateTicket(ctx, normalizedTicket)
	if err != nil {
		return nil, err
	}

	if params.SteamID == "" {
		return nil, newAuthError("steam authentication failed")
	}

	if steamID != params.SteamID {
		return nil, newAuthError("steam_id does not match ticket")
	}

	if u.requireOwnership {
		ownsApp, err := u.checkAppOwnership(ctx, params.SteamID)
		if err != nil {
			return nil, err
		}
		if !ownsApp {
			return nil, newAuthError("steam app ownership required")
		}
	}

	profile, err := u.fetchPlayerSummary(ctx, params.SteamID)
	if err != nil {
		return nil, err
	}

	displayName := "steam_" + params.SteamID
	avatarURL := ""
	if profile != nil {
		if profile.PersonaName != "" {
			displayName = profile.PersonaName
		}
		if profile.AvatarFull != "" {
			avatarURL = profile.AvatarFull
		} else if profile.AvatarMedium != "" {
			avatarURL = profile.AvatarMedium
		} else if profile.Avatar != "" {
			avatarURL = profile.Avatar
		}
	}

	platformData := map[string]interface{}{
		"steam_id":         params.SteamID,
		"owner_steam_id":   params.OwnerSteamID,
		"vac_banned":       params.VacBanned,
		"publisher_banned": params.PublisherBanned,
	}
	if profile != nil {
		platformData["persona_name"] = profile.PersonaName
		platformData["profile_url"] = profile.ProfileURL
		platformData["avatar"] = profile.Avatar
		platformData["avatar_medium"] = profile.AvatarMedium
		platformData["avatar_full"] = profile.AvatarFull
		platformData["persona_state"] = profile.PersonaState
		platformData["community_visibility_state"] = profile.CommunityVisibilityState
		platformData["profile_state"] = profile.ProfileState
		platformData["last_logoff"] = profile.LastLogoff
		platformData["time_created"] = profile.TimeCreated
		if profile.LocCountryCode != "" {
			platformData["loc_country_code"] = profile.LocCountryCode
		}
		if profile.LocStateCode != "" {
			platformData["loc_state_code"] = profile.LocStateCode
		}
		if profile.LocCityID != 0 {
			platformData["loc_city_id"] = profile.LocCityID
		}
	}

	user, err := u.userRepo.FindByPlatformUserID(ctx, "steam", params.SteamID)
	if err != nil {
		if ent.IsNotFound(err) {
			user, err = u.userRepo.Create(ctx, &entity.CreateUserRequest{
				PlatformType:        "steam",
				PlatformUserID:      params.SteamID,
				PlatformDisplayName: displayName,
				PlatformAvatarURL:   avatarURL,
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
	} else {
		user, err = u.userRepo.UpdatePlatformProfile(ctx, user.ID, displayName, avatarURL, displayName, platformData)
		if err != nil {
			return nil, err
		}
	}

	// 차단 계정 거절. 세션/시크릿 발급 없이 즉시 종료.
	if user != nil && user.User != nil && user.IsBanned {
		return nil, ErrAccountBanned
	}

	var hmacSecret string
	if u.hmacCfg != nil && u.hmacCfg.IsIssuingSecret() {
		s, err := hmacauth.GenerateSecret()
		if err != nil {
			return nil, fmt.Errorf("hmac secret 발급 실패: %w", err)
		}
		hmacSecret = s
	}

	sid, err := u.sessionStore.CreateWithSecret(ctx, user, hmacSecret)
	if err != nil {
		return nil, err
	}
	resp := user.ToSessionResponse(sid)
	resp.HmacSecret = hmacSecret
	return resp, nil
}

type steamAuthParams struct {
	Result          string `json:"result"`
	SteamID         string `json:"steamid"`
	OwnerSteamID    string `json:"ownersteamid"`
	VacBanned       bool   `json:"vacbanned"`
	PublisherBanned bool   `json:"publisherbanned"`
}

type steamPlayerSummary struct {
	SteamID                  string `json:"steamid"`
	PersonaName              string `json:"personaname"`
	ProfileURL               string `json:"profileurl"`
	Avatar                   string `json:"avatar"`
	AvatarMedium             string `json:"avatarmedium"`
	AvatarFull               string `json:"avatarfull"`
	PersonaState             int    `json:"personastate"`
	CommunityVisibilityState int    `json:"communityvisibilitystate"`
	ProfileState             int    `json:"profilestate"`
	LastLogoff               int64  `json:"lastlogoff"`
	TimeCreated              int64  `json:"timecreated"`
	LocCountryCode           string `json:"loccountrycode"`
	LocStateCode             string `json:"locstatecode"`
	LocCityID                int    `json:"loccityid"`
}

type steamPlayerSummaryResponse struct {
	Response struct {
		Players []steamPlayerSummary `json:"players"`
	} `json:"response"`
}

type steamAuthResponse struct {
	Response struct {
		Params steamAuthParams `json:"params"`
		Error  struct {
			ErrorCode int    `json:"errorcode"`
			ErrorDesc string `json:"errordesc"`
		} `json:"error"`
	} `json:"response"`
}

func (u *steamUseCase) authenticateTicket(ctx context.Context, ticket string) (*steamAuthParams, error) {
	endpoint, err := url.Parse(u.apiBase + "/ISteamUserAuth/AuthenticateUserTicket/v1/")
	if err != nil {
		return nil, err
	}

	query := endpoint.Query()
	query.Set("key", u.apiKey)
	query.Set("appid", u.appID)
	query.Set("ticket", ticket)
	if u.identity != "" {
		query.Set("identity", u.identity)
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam ticket auth failed: status %d", resp.StatusCode)
	}

	var payload steamAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if payload.Response.Error.ErrorCode != 0 {
		return nil, newAuthError(payload.Response.Error.ErrorDesc)
	}

	params := payload.Response.Params
	if params.Result != "" && !strings.EqualFold(params.Result, "OK") {
		return nil, newAuthError(params.Result)
	}

	return &params, nil
}

func (u *steamUseCase) fetchPlayerSummary(ctx context.Context, steamID string) (*steamPlayerSummary, error) {
	endpoint, err := url.Parse(u.apiBase + "/ISteamUser/GetPlayerSummaries/v2/")
	if err != nil {
		return nil, err
	}

	query := endpoint.Query()
	query.Set("key", u.apiKey)
	query.Set("steamids", steamID)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steam player summary failed: status %d", resp.StatusCode)
	}

	var payload steamPlayerSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if len(payload.Response.Players) == 0 {
		return nil, newAuthError("steam profile not found")
	}

	return &payload.Response.Players[0], nil
}

type steamOwnership struct {
	OwnsApp      bool   `json:"ownsapp"`
	Permanent    bool   `json:"permanent"`
	Timestamp    string `json:"timestamp"`
	TimeExpires  string `json:"timeexpires"`
	OwnerSteamID string `json:"ownersteamid"`
	SiteLicense  bool   `json:"sitelicense"`
	UserCanceled bool   `json:"usercanceled"`
}

type steamOwnershipResponse struct {
	AppOwnership steamOwnership `json:"appownership"`
	Response     steamOwnership `json:"response"`
}

// checkAppOwnership 허용된 App ID 목록 중 하나라도 소유하면 true를 반환한다.
// 첫 성공에서 즉시 통과(short-circuit)하여 Steam Web API 왕복을 최소화한다.
// 목록이 비어 있으면 기존 단일 appID로 폴백한다.
func (u *steamUseCase) checkAppOwnership(ctx context.Context, steamID string) (bool, error) {
	appIDs := u.appIDs
	if len(appIDs) == 0 {
		appIDs = []string{u.appID}
	}

	var lastErr error
	for _, appID := range appIDs {
		owns, err := u.checkSingleAppOwnership(ctx, steamID, appID)
		if err != nil {
			// 한 App ID 조회가 실패해도 다음 App ID를 계속 시도한다.
			lastErr = err
			continue
		}
		if owns {
			return true, nil
		}
	}

	// 모두 미소유이고 조회 중 에러가 있었다면 그 에러를 전달(설정/네트워크 문제 구분용).
	if lastErr != nil {
		return false, lastErr
	}
	return false, nil
}

// checkSingleAppOwnership 단일 App ID에 대한 소유권을 조회한다.
func (u *steamUseCase) checkSingleAppOwnership(ctx context.Context, steamID string, appID string) (bool, error) {
	endpoint, err := url.Parse(u.apiBase + "/ISteamUser/CheckAppOwnership/v4/")
	if err != nil {
		return false, err
	}

	query := endpoint.Query()
	query.Set("key", u.apiKey)
	query.Set("steamid", steamID)
	query.Set("appid", appID)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return false, err
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("steam app ownership check failed: status %d", resp.StatusCode)
	}

	var payload steamOwnershipResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}

	ownership := payload.AppOwnership
	if ownership == (steamOwnership{}) {
		ownership = payload.Response
	}

	return ownership.OwnsApp, nil
}

func normalizeSteamTicket(ticket string) (string, error) {
	trimmed := strings.TrimSpace(ticket)
	if trimmed == "" {
		return "", fmt.Errorf("empty ticket")
	}

	if _, err := hex.DecodeString(trimmed); err == nil {
		return strings.ToLower(trimmed), nil
	}

	for _, enc := range []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
	} {
		if decoded, err := enc(trimmed); err == nil {
			return hex.EncodeToString(decoded), nil
		}
	}

	return "", fmt.Errorf("invalid ticket encoding")
}
