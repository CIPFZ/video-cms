package service

import (
	"context"
	"errors"
	"gorm.io/gorm"

	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"
)

// IApiService 定义了 API 管理的服务层接口
type IApiService interface {
	// GetApiList 获取 API 列表
	GetApiList(ctx context.Context, req dto.SearchApiReq) ([]model.SysApi, int64, error)
	// CreateApi 创建一个新的 API
	CreateApi(ctx context.Context, req dto.CreateApiReq) error
	// UpdateApi 更新一个已存在的 API
	UpdateApi(ctx context.Context, req dto.UpdateApiReq) error
	// DeleteApi 删除一个或多个 API
	DeleteApi(ctx context.Context, req dto.DeleteApiReq) error
}

// ApiService 是 IApiService 的实现
type ApiService struct {
	apiRepo repository.IApiRepository // 依赖注入 ApiRepository
}

// NewApiService 创建一个新的 ApiService 实例
func NewApiService(apiRepo repository.IApiRepository) IApiService {
	return &ApiService{
		apiRepo: apiRepo,
	}
}

// GetApiList 根据查询条件分页获取 API 列表
func (s *ApiService) GetApiList(ctx context.Context, req dto.SearchApiReq) ([]model.SysApi, int64, error) {
	return s.apiRepo.GetList(ctx, req)
}

// CreateApi 创建一个新的 API 记录
func (s *ApiService) CreateApi(ctx context.Context, req dto.CreateApiReq) error {
	// 1. 检查是否存在具有相同路径和方法的 API
	_, err := s.apiRepo.FindByPathMethod(ctx, req.Path, req.Method)
	if err == nil {
		return errors.New("存在相同路径和方法的API")
	}
	// 如果错误不是 "记录未找到"，则返回该错误
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// 2. 创建 API 实例并保存到数据库
	api := model.SysApi{
		Path:        req.Path,
		Description: req.Description,
		ApiGroup:    req.ApiGroup,
		Method:      req.Method,
	}
	return s.apiRepo.Create(ctx, &api)
}

// UpdateApi 更新一个已存在的 API
func (s *ApiService) UpdateApi(ctx context.Context, req dto.UpdateApiReq) error {
	// 1. 查找旧的 API 数据
	oldApi, err := s.apiRepo.FindById(ctx, req.ID)
	if err != nil {
		return errors.New("API不存在")
	}

	// 2. 如果路径或方法已更改，请检查新组合是否已存在
	if oldApi.Path != req.Path || oldApi.Method != req.Method {
		if _, err := s.apiRepo.FindByPathMethod(ctx, req.Path, req.Method); err == nil {
			return errors.New("修改后的API路径和方法已存在")
		}
	}

	// 3. 更新 API 数据
	newApi := model.SysApi{
		Path:        req.Path,
		Description: req.Description,
		ApiGroup:    req.ApiGroup,
		Method:      req.Method,
	}
	newApi.ID = req.ID

	// 调用仓库层的方法来更新 API 并同步 Casbin 策略
	return s.apiRepo.UpdateWithSyncCasbin(ctx, oldApi, newApi)
}

// DeleteApi 删除一个或多个 API
func (s *ApiService) DeleteApi(ctx context.Context, req dto.DeleteApiReq) error {
	var ids []uint
	if req.ID != 0 {
		ids = append(ids, req.ID)
	}
	if len(req.IDs) > 0 {
		ids = append(ids, req.IDs...)
	}
	if len(ids) == 0 {
		return errors.New("未选择数据")
	}

	// 调用仓库层的方法来删除 API 并同步 Casbin 策略
	return s.apiRepo.DeleteWithSyncCasbin(ctx, ids)
}
