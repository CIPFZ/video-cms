package service

import (
	"context"

	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"
)

// IOperationLogService 定义了操作日志服务的接口
type IOperationLogService interface {
	// GetOperationLogList 分页获取操作日志列表
	GetOperationLogList(ctx context.Context, req dto.SearchOperationLogReq) ([]model.SysOperationLog, int64, error)
	// DeleteOperationLogByIds 批量删除操作日志
	DeleteOperationLogByIds(ctx context.Context, ids []uint) error
}

// OperationLogService 实现了 IOperationLogService 接口。
// 它使用一个带缓冲的 channel 和一个后台 goroutine 来实现操作日志的异步、批量写入，
// 从而避免阻塞 HTTP 请求的正常流程。
type OperationLogService struct {
	opLogRepo repository.IOperationLogRepository // 数据仓库依赖
}

// NewOperationLogService 创建并启动一个新的 OperationLogService 实例。
// 它会立即启动一个后台 worker goroutine 来消费日志。
func NewOperationLogService(opLogRepo repository.IOperationLogRepository) IOperationLogService {
	s := &OperationLogService{
		opLogRepo: opLogRepo,
	}
	return s
}

// GetOperationLogList 是一个同步方法，直接调用数据仓库来分页获取操作日志。
func (s *OperationLogService) GetOperationLogList(ctx context.Context, req dto.SearchOperationLogReq) ([]model.SysOperationLog, int64, error) {
	return s.opLogRepo.GetList(ctx, req)
}

// DeleteOperationLogByIds 是一个同步方法，直接调用数据仓库来批量删除操作日志。
func (s *OperationLogService) DeleteOperationLogByIds(ctx context.Context, ids []uint) error {
	return s.opLogRepo.DeleteByIds(ctx, ids)
}
