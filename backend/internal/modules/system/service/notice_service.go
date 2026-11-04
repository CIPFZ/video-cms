package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"
)

type INoticeService interface {
	CreateNotice(ctx context.Context, req dto.CreateNoticeReq, creatorID uint) error
	GetNoticeList(ctx context.Context, req dto.SearchNoticeReq) ([]dto.NoticeListItem, int64, error)
	GetMyNotices(ctx context.Context, userID uint, page, pageSize int, popupOnly ...bool) ([]dto.MyNoticeItem, int64, error)
	MarkRead(ctx context.Context, noticeID uint, userID uint) error
}

type NoticeService struct {
	noticeRepo repository.INoticeRepository
}

func NewNoticeService(noticeRepo repository.INoticeRepository) INoticeService {
	return &NoticeService{noticeRepo: noticeRepo}
}

func (s *NoticeService) CreateNotice(ctx context.Context, req dto.CreateNoticeReq, creatorID uint) error {
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if utf8.RuneCountInString(req.Title) < 2 || utf8.RuneCountInString(req.Title) > 128 || req.Content == "" || utf8.RuneCountInString(req.Content) > 5000 {
		return errors.New("validation.invalid")
	}
	if req.Level == "" {
		req.Level = model.NoticeLevelInfo
	}
	if req.EndAt != nil && req.StartAt != nil && !req.EndAt.After(*req.StartAt) {
		return errors.New("endAt must be greater than startAt")
	}

	if req.EndAt != nil && !req.EndAt.After(time.Now()) {
		return errors.New("notice.expiredEnd")
	}
	req.TargetIDs = uniqueNoticeIDs(req.TargetIDs)
	if req.TargetType == model.NoticeTargetAll {
		req.TargetIDs = nil
	}
	userIDs, err := s.resolveTargetUsers(ctx, req.TargetType, req.TargetIDs)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return errors.New("no valid receiver found")
	}

	notice := &model.SysNotice{
		Title:       req.Title,
		Content:     req.Content,
		Level:       req.Level,
		TargetType:  req.TargetType,
		TargetIDs:   req.TargetIDs,
		IsPopup:     req.IsPopup,
		NeedConfirm: req.NeedConfirm,
		StartAt:     req.StartAt,
		EndAt:       req.EndAt,
		CreatedBy:   creatorID,
	}
	return s.noticeRepo.CreateWithReceivers(ctx, notice, userIDs)
}

func (s *NoticeService) GetNoticeList(ctx context.Context, req dto.SearchNoticeReq) ([]dto.NoticeListItem, int64, error) {
	return s.noticeRepo.GetNoticeList(ctx, req)
}

func (s *NoticeService) GetMyNotices(ctx context.Context, userID uint, page, pageSize int, popupOnly ...bool) ([]dto.MyNoticeItem, int64, error) {
	return s.noticeRepo.GetMyNotices(ctx, userID, page, pageSize, popupOnly...)
}

func (s *NoticeService) MarkRead(ctx context.Context, noticeID uint, userID uint) error {
	return s.noticeRepo.MarkRead(ctx, noticeID, userID, time.Now())
}

func (s *NoticeService) resolveTargetUsers(ctx context.Context, targetType model.NoticeTargetType, targetIDs []uint) ([]uint, error) {
	var (
		ids []uint
		err error
	)

	switch targetType {
	case model.NoticeTargetAll:
		ids, err = s.noticeRepo.ListAllActiveUserIDs(ctx)
	case model.NoticeTargetRoles:
		if len(targetIDs) == 0 {
			return nil, errors.New("notice.invalidTargets")
		}
		if err := s.noticeRepo.ValidateRoles(ctx, targetIDs); err != nil {
			return nil, err
		}
		ids, err = s.noticeRepo.ListUserIDsByAuthorityIDs(ctx, targetIDs)
	case model.NoticeTargetUsers:
		ids, err = s.noticeRepo.ListExistingUserIDs(ctx, targetIDs)
		if err == nil && (len(ids) != len(targetIDs) || len(ids) == 0) {
			return nil, errors.New("notice.invalidTargets")
		}
	default:
		return nil, errors.New("invalid targetType")
	}
	if err != nil {
		return nil, err
	}

	unique := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		unique[id] = struct{}{}
	}
	result := make([]uint, 0, len(unique))
	for id := range unique {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func uniqueNoticeIDs(ids []uint) []uint {
	seen := map[uint]bool{}
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
