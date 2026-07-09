package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/witchs-lounge_backend/ent"
	"github.com/witchs-lounge_backend/ent/consentlog"
	"github.com/witchs-lounge_backend/internal/domain/entity"
	"github.com/witchs-lounge_backend/internal/domain/repository"
)

type consentLogRepository struct {
	client *ent.Client
}

func NewConsentLogRepository(client *ent.Client) repository.ConsentLogRepository {
	return &consentLogRepository{client: client}
}

func (r *consentLogRepository) CreateWithClientConsentID(ctx context.Context, userID *uuid.UUID, req *entity.CreateConsentLogRequest) (*entity.ConsentLog, error) {
	// client_consent_id / client_id 는 DB 에서 uuid 타입. 문자열 페이로드를 파싱한다.
	// 파싱 실패는 결정론적 형식 오류 → 호출측(usecase)이 invalid_payload 로 분류한다.
	consentID, err := uuid.Parse(req.ClientConsentID)
	if err != nil {
		return nil, err
	}
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return nil, err
	}

	create := r.client.ConsentLog.Create().
		SetClientConsentID(consentID).
		SetClientID(clientID).
		SetNillableUserID(userID). // nil 이면 user_id NULL (로그인 전 발생)
		SetConsentType(consentlog.ConsentType(req.ConsentType)).
		SetPolicyVersion(req.PolicyVersion).
		SetGranted(req.Granted).
		SetNickname(req.Nickname).
		SetConsentedAt(req.ConsentedAtTime()).
		SetClientVersion(req.ClientVersion).
		SetPlatform(req.Platform).
		SetLocale(req.Locale).
		SetBuildMode(req.BuildMode)

	logEnt, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return entity.NewConsentLog(logEnt), nil
}

func (r *consentLogRepository) FindByClientConsentID(ctx context.Context, clientConsentID string) (*entity.ConsentLog, error) {
	consentID, err := uuid.Parse(clientConsentID)
	if err != nil {
		// 형식 오류면 "존재하지 않음"과 동일 취급 — 멱등 조회에서 nil 반환.
		return nil, err
	}
	logEnt, err := r.client.ConsentLog.Query().
		Where(consentlog.ClientConsentIDEQ(consentID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return entity.NewConsentLog(logEnt), nil
}
