package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	tokenCore "github.com/CIPFZ/video-cms/internal/core/token"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"
	"gorm.io/gorm"
)

type IApiTokenService interface {
	CreateApiToken(ctx context.Context, createdBy uint, req dto.CreateApiTokenReq) (*dto.ApiTokenSecretResponse, error)
	GetApiTokenList(ctx context.Context, req dto.SearchApiTokenReq) ([]dto.ApiTokenResponse, int64, error)
	GetApiTokenDetail(ctx context.Context, id uint) (*dto.ApiTokenResponse, error)
	UpdateApiToken(ctx context.Context, req dto.UpdateApiTokenReq) error
	DeleteApiToken(ctx context.Context, req dto.DeleteApiTokenReq) error
	ResetApiToken(ctx context.Context, id uint) (*dto.ApiTokenSecretResponse, error)
	EnableApiToken(ctx context.Context, id uint) error
	DisableApiToken(ctx context.Context, id uint) error
}

type ApiTokenService struct {
	tokenRepo repository.IApiTokenRepository
}

func NewApiTokenService(tokenRepo repository.IApiTokenRepository) IApiTokenService {
	return &ApiTokenService{
		tokenRepo: tokenRepo,
	}
}

func (s *ApiTokenService) CreateApiToken(ctx context.Context, createdBy uint, req dto.CreateApiTokenReq) (*dto.ApiTokenSecretResponse, error) {
	if strings.TrimSpace(req.Name) == "" || req.MaxConcurrency < 1 || req.MaxConcurrency > 1000 || utf8.RuneCountInString(req.Name) > 100 || utf8.RuneCountInString(req.Description) > 255 {
		return nil, errors.New("validation.invalid")
	}
	expiresAt, err := parseExpiresAt(req.ExpiresAt, req.NeverExpire)
	if err != nil {
		return nil, err
	}

	apis, err := s.resolveApis(ctx, req.ApiIds)
	if err != nil {
		return nil, err
	}

	rawToken := tokenCore.GenerateRawToken()
	token := &model.SysApiToken{
		TokenHash:      tokenCore.HashToken(rawToken),
		TokenPrefix:    buildTokenPrefix(rawToken),
		Name:           strings.TrimSpace(req.Name),
		Description:    strings.TrimSpace(req.Description),
		ExpiresAt:      expiresAt,
		MaxConcurrency: normalizeMaxConcurrency(req.MaxConcurrency),
		Enabled:        true,
		CreatedBy:      createdBy,
		Apis:           apis,
	}

	if err := s.tokenRepo.Create(ctx, token); err != nil {
		return nil, err
	}

	return buildSecretResponse(token, rawToken), nil
}

func (s *ApiTokenService) GetApiTokenList(ctx context.Context, req dto.SearchApiTokenReq) ([]dto.ApiTokenResponse, int64, error) {
	list, total, err := s.tokenRepo.GetList(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	resp := make([]dto.ApiTokenResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, buildApiTokenResponse(&item))
	}
	return resp, total, nil
}

func (s *ApiTokenService) GetApiTokenDetail(ctx context.Context, id uint) (*dto.ApiTokenResponse, error) {
	token, err := s.tokenRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := buildApiTokenResponse(token)
	return &resp, nil
}

func (s *ApiTokenService) UpdateApiToken(ctx context.Context, req dto.UpdateApiTokenReq) error {
	token, err := s.tokenRepo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}

	if strings.TrimSpace(req.Name) == "" || req.MaxConcurrency < 1 || req.MaxConcurrency > 1000 || utf8.RuneCountInString(req.Name) > 100 || utf8.RuneCountInString(req.Description) > 255 {
		return errors.New("validation.invalid")
	}
	expiresAt, err := parseExpiresAt(req.ExpiresAt, req.NeverExpire)
	if err != nil {
		return err
	}

	apis, err := s.resolveApis(ctx, req.ApiIds)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"name":            strings.TrimSpace(req.Name),
		"description":     strings.TrimSpace(req.Description),
		"expires_at":      expiresAt,
		"max_concurrency": normalizeMaxConcurrency(req.MaxConcurrency),
	}
	return s.tokenRepo.UpdateWithAPIs(ctx, token, updates, apis)
}

func (s *ApiTokenService) DeleteApiToken(ctx context.Context, req dto.DeleteApiTokenReq) error {
	ids := make([]uint, 0, len(req.IDs)+1)
	if req.ID != 0 {
		ids = append(ids, req.ID)
	}
	ids = append(ids, req.IDs...)
	if len(ids) == 0 {
		return errors.New("未选择要删除的 token")
	}
	return s.tokenRepo.DeleteByIDs(ctx, ids)
}

func (s *ApiTokenService) ResetApiToken(ctx context.Context, id uint) (*dto.ApiTokenSecretResponse, error) {
	token, err := s.tokenRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	rawToken := tokenCore.GenerateRawToken()
	updates := map[string]interface{}{
		"token_hash":   tokenCore.HashToken(rawToken),
		"token_prefix": buildTokenPrefix(rawToken),
	}
	if err := s.tokenRepo.UpdateColumns(ctx, id, updates); err != nil {
		return nil, err
	}

	token.TokenHash = updates["token_hash"].(string)
	token.TokenPrefix = updates["token_prefix"].(string)
	return buildSecretResponse(token, rawToken), nil
}

func (s *ApiTokenService) EnableApiToken(ctx context.Context, id uint) error {
	return s.tokenRepo.UpdateColumns(ctx, id, map[string]interface{}{"enabled": true})
}

func (s *ApiTokenService) DisableApiToken(ctx context.Context, id uint) error {
	return s.tokenRepo.UpdateColumns(ctx, id, map[string]interface{}{"enabled": false})
}

func (s *ApiTokenService) resolveApis(ctx context.Context, apiIDs []uint) ([]model.SysApi, error) {
	if len(apiIDs) == 0 {
		return nil, errors.New("请至少选择一个 API")
	}
	unique := make([]uint, 0, len(apiIDs))
	seen := map[uint]bool{}
	for _, id := range apiIDs {
		if !seen[id] {
			unique = append(unique, id)
			seen[id] = true
		}
	}
	apiIDs = unique
	apis, err := s.tokenRepo.LoadApisByIDs(ctx, apiIDs)
	if err != nil {
		return nil, err
	}
	if len(apis) != len(apiIDs) {
		return nil, errors.New("存在无效的 API 选择")
	}
	return apis, nil
}

func parseExpiresAt(raw *string, neverExpire bool) (*time.Time, error) {
	if neverExpire || raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, errors.New("token.expiryRequired")
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
	if err != nil {
		return nil, errors.New("过期时间格式错误")
	}
	if parsed.Before(time.Now()) {
		return nil, errors.New("过期时间必须晚于当前时间")
	}
	return &parsed, nil
}

func normalizeMaxConcurrency(value int) int {
	if value <= 0 {
		return 5
	}
	return value
}

func buildTokenPrefix(raw string) string {
	if len(raw) <= 8 {
		return raw
	}
	return raw[:8]
}

func buildApiTokenResponse(token *model.SysApiToken) dto.ApiTokenResponse {
	resp := dto.ApiTokenResponse{
		ID:             token.ID,
		TokenPrefix:    token.TokenPrefix,
		Name:           token.Name,
		Description:    token.Description,
		NeverExpire:    token.ExpiresAt == nil,
		MaxConcurrency: token.MaxConcurrency,
		Enabled:        token.Enabled,
		CreatedAt:      token.CreatedAt.Format(time.RFC3339),
		CreatedBy:      token.CreatedBy,
		Apis:           buildApiItems(token.Apis),
	}
	if token.ExpiresAt != nil {
		value := token.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &value
	}
	if token.LastUsedAt != nil {
		value := token.LastUsedAt.Format(time.RFC3339)
		resp.LastUsedAt = &value
	}
	return resp
}

func buildSecretResponse(token *model.SysApiToken, rawToken string) *dto.ApiTokenSecretResponse {
	resp := buildApiTokenResponse(token)
	return &dto.ApiTokenSecretResponse{
		ApiTokenResponse: resp,
		Token:            rawToken,
	}
}

func buildApiItems(apis []model.SysApi) []dto.ApiSimpleItem {
	items := make([]dto.ApiSimpleItem, 0, len(apis))
	for _, api := range apis {
		items = append(items, dto.ApiSimpleItem{
			ID:          api.ID,
			Path:        api.Path,
			Method:      api.Method,
			ApiGroup:    api.ApiGroup,
			Description: api.Description,
		})
	}
	return items
}

var _ = gorm.ErrRecordNotFound
