package repository

import (
	"context"

	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"

	"gorm.io/gorm"
)

// IOperationLogRepository 定义了操作日志数据仓库的接口
type IOperationLogRepository interface {
	// DeleteByIds 根据 ID 列表批量删除操作日志
	DeleteByIds(ctx context.Context, ids []uint) error
	// GetList 根据查询条件分页获取操作日志列表
	GetList(ctx context.Context, req dto.SearchOperationLogReq) ([]model.SysOperationLog, int64, error)
}

// OperationLogRepository 是 IOperationLogRepository 的 GORM 实现
type OperationLogRepository struct {
	db *gorm.DB
}

// NewOperationLogRepository 创建一个新的 OperationLogRepository 实例
func NewOperationLogRepository(db *gorm.DB) IOperationLogRepository {
	return &OperationLogRepository{db: db}
}

// DeleteByIds 根据提供的主键 ID 列表，批量硬删除操作日志记录。
// 使用 Unscoped() 确保执行的是物理删除，而不是软删除。
func (r *OperationLogRepository) DeleteByIds(ctx context.Context, ids []uint) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&model.SysOperationLog{}, ids).Error
}

// GetList 根据复杂的查询条件进行分页和筛选，获取操作日志列表。
func (r *OperationLogRepository) GetList(ctx context.Context, req dto.SearchOperationLogReq) ([]model.SysOperationLog, int64, error) {
	var list []model.SysOperationLog
	var total int64
	db := r.db.WithContext(ctx).Model(&model.SysOperationLog{})

	// --- 动态构建查询条件 ---
	if req.Method != "" {
		db = db.Where("method = ?", req.Method)
	}
	if req.Path != "" {
		db = db.Where("path LIKE ?", "%"+req.Path+"%")
	}
	if req.Ip != "" {
		db = db.Where("ip LIKE ?", "%"+req.Ip+"%")
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	if req.UserID != 0 {
		db = db.Where("user_id = ?", req.UserID)
	}
	if req.TraceID != "" {
		db = db.Where("trace_id = ?", req.TraceID)
	}
	if req.StartDate != nil && req.EndDate != nil {
		db = db.Where("created_at BETWEEN ? AND ?", req.StartDate, req.EndDate)
	}

	// --- 执行查询 ---

	// 1. 首先执行 Count 查询获取总记录数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 2. 然后执行分页查询获取当前页的数据
	err := db.Scopes(req.Paginate()).
		Preload("User").  // 使用 Preload 预加载关联的 User 信息
		Order("id desc"). // 按 ID 降序排序，最新的日志在前面
		Find(&list).Error

	return list, total, err
}
