package repository

import (
	"context"
	"errors"
	"github.com/CIPFZ/video-cms/internal/modules/common"
	"time"

	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type INoticeRepository interface {
	CreateWithReceivers(ctx context.Context, notice *model.SysNotice, userIDs []uint) error
	GetNoticeList(ctx context.Context, req dto.SearchNoticeReq) ([]dto.NoticeListItem, int64, error)
	GetMyNotices(ctx context.Context, userID uint, page, pageSize int, popupOnly ...bool) ([]dto.MyNoticeItem, int64, error)
	MarkRead(ctx context.Context, noticeID uint, userID uint, readAt time.Time) error
	ListAllActiveUserIDs(ctx context.Context) ([]uint, error)
	ListUserIDsByAuthorityIDs(ctx context.Context, authorityIDs []uint) ([]uint, error)
	ListExistingUserIDs(ctx context.Context, userIDs []uint) ([]uint, error)
	ValidateRoles(ctx context.Context, ids []uint) error
}

type NoticeRepository struct {
	db *gorm.DB
}

func NewNoticeRepository(db *gorm.DB) INoticeRepository {
	return &NoticeRepository{db: db}
}

func (r *NoticeRepository) CreateWithReceivers(ctx context.Context, notice *model.SysNotice, userIDs []uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(notice).Error; err != nil {
			return err
		}
		if len(userIDs) == 0 {
			return nil
		}
		receivers := make([]model.SysNoticeReceiver, 0, len(userIDs))
		for _, userID := range userIDs {
			receivers = append(receivers, model.SysNoticeReceiver{NoticeID: notice.ID, UserID: userID})
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&receivers, 500).Error
	})
}

func (r *NoticeRepository) GetNoticeList(ctx context.Context, req dto.SearchNoticeReq) ([]dto.NoticeListItem, int64, error) {
	req.Normalize()

	base := r.db.WithContext(ctx).Model(&model.SysNotice{})
	if req.Title != "" {
		base = base.Where("title LIKE ?", "%"+req.Title+"%")
	}
	if req.TargetType != "" {
		base = base.Where("target_type = ?", req.TargetType)
	}
	if req.Level != "" {
		base = base.Where("level = ?", req.Level)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []dto.NoticeListItem
	err := base.Select(`
		sys_notices.id,
		sys_notices.created_at,
		sys_notices.title,
		sys_notices.content,
		sys_notices.level,
		sys_notices.target_type,
 sys_notices.target_ids,
		sys_notices.is_popup,
		sys_notices.need_confirm,
		sys_notices.start_at,
		sys_notices.end_at,
		sys_notices.created_by,
		COUNT(DISTINCT nr.id) AS receiver_count,
		COUNT(DISTINCT CASE WHEN nr.read_at IS NOT NULL THEN nr.id END) AS read_count
	`).
		Joins("LEFT JOIN sys_notice_receivers nr ON nr.notice_id = sys_notices.id AND nr.deleted_at IS NULL").
		Group("sys_notices.id").
		Order("sys_notices.id DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Scan(&list).Error
	return list, total, err
}

func (r *NoticeRepository) GetMyNotices(ctx context.Context, userID uint, page, pageSize int, popupOnly ...bool) ([]dto.MyNoticeItem, int64, error) {
	info := common.PageInfo{Page: page, PageSize: pageSize}
	info.Normalize()
	page, pageSize = info.Page, info.PageSize

	now := time.Now()

	base := r.db.WithContext(ctx).Table("sys_notice_receivers AS nr").
		Joins("JOIN sys_notices n ON n.id = nr.notice_id").
		Where("nr.user_id = ?", userID).
		Where("n.deleted_at IS NULL AND nr.deleted_at IS NULL").
		Where("(n.start_at IS NULL OR n.start_at <= ?)", now).
		Where("(n.end_at IS NULL OR n.end_at > ?)", now)

	if len(popupOnly) > 0 && popupOnly[0] {
		base = base.Where("n.is_popup = ? AND nr.read_at IS NULL", true)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []dto.MyNoticeItem
	err := base.Select(`
		n.id,
		n.created_at,
		n.title,
		n.content,
		n.level,
		n.is_popup,
		n.need_confirm,
		n.start_at,
		n.end_at,
		nr.read_at
	`).
		Order("CASE WHEN nr.read_at IS NULL THEN 0 ELSE 1 END ASC, n.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&list).Error
	return list, total, err
}

func (r *NoticeRepository) MarkRead(ctx context.Context, noticeID uint, userID uint, readAt time.Time) error {
	active := r.db.WithContext(ctx).Model(&model.SysNotice{}).Select("id").Where("id = ?", noticeID).
		Where("(start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at > ?)", readAt, readAt)
	base := r.db.WithContext(ctx).Model(&model.SysNoticeReceiver{}).Where("notice_id IN (?) AND user_id = ?", active, userID)
	// COALESCE keeps the first acknowledgement while RowsAffected is allowed to be
	// zero on MySQL for an already-read row. Recheck membership, never fake success.
	result := base.Update("read_at", gorm.Expr("COALESCE(read_at, ?)", readAt))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	var count int64
	if err := base.Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("notice.unavailable")
	}
	return nil
}

func (r *NoticeRepository) ListAllActiveUserIDs(ctx context.Context) ([]uint, error) {
	var ids []uint
	err := r.db.WithContext(ctx).Model(&model.SysUser{}).Where("status = ?", model.UserActive).Pluck("id", &ids).Error
	return ids, err
}

func (r *NoticeRepository) ListUserIDsByAuthorityIDs(ctx context.Context, authorityIDs []uint) ([]uint, error) {
	if len(authorityIDs) == 0 {
		return []uint{}, nil
	}
	var ids []uint
	err := r.db.WithContext(ctx).
		Table("sys_user_authorities AS ua").
		Distinct("ua.user_id").
		Joins("JOIN sys_users u ON u.id = ua.user_id AND u.deleted_at IS NULL").
		Joins("JOIN sys_authorities a ON a.authority_id = ua.authority_id AND a.deleted_at IS NULL").
		Where("ua.authority_id IN ?", authorityIDs).
		Where("u.status = ?", model.UserActive).
		Pluck("ua.user_id", &ids).Error
	return ids, err
}

func (r *NoticeRepository) ListExistingUserIDs(ctx context.Context, userIDs []uint) ([]uint, error) {
	if len(userIDs) == 0 {
		return []uint{}, nil
	}
	var ids []uint
	err := r.db.WithContext(ctx).Model(&model.SysUser{}).
		Where("id IN ?", userIDs).
		Where("status = ?", model.UserActive).
		Pluck("id", &ids).Error
	return ids, err
}

func (r *NoticeRepository) ValidateRoles(ctx context.Context, ids []uint) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.SysAuthority{}).Where("authority_id IN ? AND deleted_at IS NULL", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return errors.New("notice.invalidTargets")
	}
	return nil
}
