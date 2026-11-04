package repository

import (
	"context"
	"errors"
	"github.com/CIPFZ/video-cms/internal/core/claims"
	tokenCore "github.com/CIPFZ/video-cms/internal/core/token"
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"gorm.io/gorm"
)

type IApiTokenRepository interface {
	Create(ctx context.Context, token *model.SysApiToken) error
	GetList(ctx context.Context, req dto.SearchApiTokenReq) ([]model.SysApiToken, int64, error)
	FindByID(ctx context.Context, id uint) (*model.SysApiToken, error)
	FindByHash(ctx context.Context, hash string) (*model.SysApiToken, error)
	LoadApisByIDs(ctx context.Context, ids []uint) ([]model.SysApi, error)
	UpdateWithAPIs(ctx context.Context, token *model.SysApiToken, updates map[string]interface{}, apis []model.SysApi) error
	UpdateColumns(ctx context.Context, id uint, updates map[string]interface{}) error
	DeleteByIDs(ctx context.Context, ids []uint) error
	TouchLastUsedAt(ctx context.Context, id uint, usedAt time.Time) error
}

type ApiTokenRepository struct {
	db     *gorm.DB
	prefix string
}

func NewApiTokenRepository(db *gorm.DB, prefix ...string) IApiTokenRepository {
	p := "/api/v1"
	if len(prefix) > 0 {
		p = prefix[0]
	}
	return &ApiTokenRepository{db: db, prefix: p}
}

func (r *ApiTokenRepository) Create(ctx context.Context, token *model.SysApiToken) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		if err := validateTokenAPIs(tx, token.Apis, r.prefix); err != nil {
			return err
		}
		return tx.Omit("Apis.*").Create(token).Error
	})
}

func (r *ApiTokenRepository) GetList(ctx context.Context, req dto.SearchApiTokenReq) ([]model.SysApiToken, int64, error) {
	var list []model.SysApiToken
	var total int64

	base := r.db.WithContext(ctx).Model(&model.SysApiToken{})
	if req.Name != "" {
		base = base.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Enabled != nil {
		base = base.Where("enabled = ?", *req.Enabled)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.Preload("Apis").
		Scopes(req.Paginate()).
		Order("id desc").
		Find(&list).Error

	return list, total, err
}

func (r *ApiTokenRepository) FindByID(ctx context.Context, id uint) (*model.SysApiToken, error) {
	var token model.SysApiToken
	err := r.db.WithContext(ctx).Preload("Apis").First(&token, id).Error
	return &token, err
}

func (r *ApiTokenRepository) FindByHash(ctx context.Context, hash string) (*model.SysApiToken, error) {
	var token model.SysApiToken
	err := r.db.WithContext(ctx).Preload("Apis").Where("token_hash = ?", hash).First(&token).Error
	return &token, err
}

func (r *ApiTokenRepository) LoadApisByIDs(ctx context.Context, ids []uint) ([]model.SysApi, error) {
	var apis []model.SysApi
	if err := r.db.WithContext(ctx).Find(&apis, ids).Error; err != nil {
		return nil, err
	}
	return apis, nil
}

func (r *ApiTokenRepository) UpdateWithAPIs(ctx context.Context, token *model.SysApiToken, updates map[string]interface{}, apis []model.SysApi) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		if err := validateTokenAPIs(tx, apis, r.prefix); err != nil {
			return err
		}
		var current model.SysApiToken
		if err := tx.First(&current, token.ID).Error; err != nil {
			return err
		}
		token = &current
		if len(updates) > 0 {
			if err := tx.Model(token).Updates(updates).Error; err != nil {
				return err
			}
		}
		return tx.Model(token).Omit("Apis.*").Association("Apis").Replace(apis)
	})
}

func (r *ApiTokenRepository) UpdateColumns(ctx context.Context, id uint, updates map[string]interface{}) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		var current model.SysApiToken
		if err := tx.First(&current, id).Error; err != nil {
			return err
		}
		return tx.Model(&current).Updates(updates).Error
	})
}

func (r *ApiTokenRepository) DeleteByIDs(ctx context.Context, ids []uint) error {
	return claims.PolicyTransaction(ctx, r.db, func(tx *gorm.DB) error {
		if err := tx.Where("api_token_id IN ?", ids).Delete(&model.SysApiTokenApi{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.SysApiToken{}, ids).Error
	})
}

func (r *ApiTokenRepository) TouchLastUsedAt(ctx context.Context, id uint, usedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.SysApiToken{}).
		Where("id = ?", id).
		Update("last_used_at", usedAt).
		Error
}

func validateTokenAPIs(tx *gorm.DB, apis []model.SysApi, prefix string) error {
	for _, api := range apis {
		var current model.SysApi
		if err := tx.First(&current, api.ID).Error; err != nil {
			return err
		}
		if !tokenCore.AllowsEndpoint(prefix, current.Method, current.Path) {
			return errors.New("token.apiNotExposed")
		}
	}
	return nil
}
