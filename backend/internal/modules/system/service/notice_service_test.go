package service

import (
	"context"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"
	"testing"
	"time"
)

func TestNoticeRecipientsAndReadBoundaries(t *testing.T) {
	db := newApiTokenTestDB(t)
	if err := db.AutoMigrate(&model.SysUser{}, &model.SysAuthority{}, &model.SysUserAuthority{}, &model.SysNotice{}, &model.SysNoticeReceiver{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	role := model.SysAuthority{AuthorityId: 888, AuthorityName: "reader"}
	db.Create(&role)
	users := []model.SysUser{{Username: "reader", Status: 1}, {Username: "outsider", Status: 1}, {Username: "disabled", Status: 0}, {Username: "deleted", Status: 1}}
	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatal(err)
		}
		db.Create(&model.SysUserAuthority{UserId: users[i].ID, AuthorityId: 888})
	}
	db.Model(&users[2]).Update("status", 0)
	db.Where("user_id = ?", users[1].ID).Delete(&model.SysUserAuthority{})
	db.Delete(&users[3])
	repo := repository.NewNoticeRepository(db)
	service := NewNoticeService(repo)
	req := dto.CreateNoticeReq{Title: "Role notice", Content: "private content", TargetType: model.NoticeTargetRoles, TargetIDs: []uint{888}, NeedConfirm: true}
	if err := service.CreateNotice(ctx, req, 1); err != nil {
		t.Fatal(err)
	}
	list, _, err := service.GetNoticeList(ctx, dto.SearchNoticeReq{})
	if err != nil || len(list) != 1 {
		t.Fatalf("list %v %v", list, err)
	}
	id := list[0].ID
	t.Run("active recipients only", func(t *testing.T) {
		if list[0].ReceiverCount != 1 {
			t.Fatalf("receivers = %d, want 1", list[0].ReceiverCount)
		}
	})
	t.Run("outsider cannot read", func(t *testing.T) {
		if err := service.MarkRead(ctx, id, users[1].ID); err == nil {
			t.Fatal("nonrecipient reported success")
		}
		items, total, err := service.GetMyNotices(ctx, users[1].ID, 1, 10)
		if err != nil || total != 0 || len(items) != 0 {
			t.Fatalf("leaked notice %v %v", items, err)
		}
	})
	t.Run("future notice cannot be confirmed", func(t *testing.T) {
		future := time.Now().Add(time.Hour)
		db.Model(&model.SysNotice{}).Where("id = ?", id).Update("start_at", future)
		if err := service.MarkRead(ctx, id, users[0].ID); err == nil {
			t.Error("future notice confirmed")
		}
		db.Model(&model.SysNotice{}).Where("id = ?", id).Update("start_at", nil)
	})
	t.Run("read idempotent", func(t *testing.T) {
		if err := service.MarkRead(ctx, id, users[0].ID); err != nil {
			t.Fatal(err)
		}
		var before, after model.SysNoticeReceiver
		db.Where("notice_id = ? AND user_id = ?", id, users[0].ID).First(&before)
		service.MarkRead(ctx, id, users[0].ID)
		db.First(&after, before.ID)
		if before.ReadAt == nil || after.ReadAt == nil || !before.ReadAt.Equal(*after.ReadAt) {
			t.Fatal("first read timestamp changed")
		}
	})
	t.Run("snapshot survives membership changes", func(t *testing.T) {
		db.Where("user_id = ?", users[0].ID).Delete(&model.SysUserAuthority{})
		db.Create(&model.SysUserAuthority{UserId: users[1].ID, AuthorityId: 888})
		_, old, _ := service.GetMyNotices(ctx, users[0].ID, 1, 10)
		_, newCount, _ := service.GetMyNotices(ctx, users[1].ID, 1, 10)
		if old != 1 || newCount != 0 {
			t.Fatalf("snapshot changed: %d %d", old, newCount)
		}
	})
	t.Run("expired or deleted cannot read", func(t *testing.T) {
		past := time.Now().Add(-time.Hour)
		db.Model(&model.SysNotice{}).Where("id = ?", id).Update("end_at", past)
		if err := service.MarkRead(ctx, id, users[0].ID); err == nil {
			t.Error("expired read allowed")
		}
		db.Delete(&model.SysNotice{}, id)
		if err := service.MarkRead(ctx, id, users[0].ID); err == nil {
			t.Error("deleted read allowed")
		}
	})
}

func TestNoticeAllTargetsValidationAndDeletedReceiver(t *testing.T) {
	database := newApiTokenTestDB(t)
	if err := database.AutoMigrate(&model.SysUser{}, &model.SysAuthority{}, &model.SysNotice{}, &model.SysNoticeReceiver{}); err != nil {
		t.Fatal(err)
	}
	first := model.SysUser{Username: "first", Status: 1}
	second := model.SysUser{Username: "second", Status: 1}
	disabled := model.SysUser{Username: "inactive", Status: 1}
	for _, u := range []*model.SysUser{&first, &second, &disabled} {
		if err := database.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	database.Model(&disabled).Update("status", 0)
	service := NewNoticeService(repository.NewNoticeRepository(database))
	ctx := context.Background()
	req := dto.CreateNoticeReq{Title: "Everyone", Content: "all active users", TargetType: model.NoticeTargetAll, TargetIDs: []uint{999999}}
	if err := service.CreateNotice(ctx, req, first.ID); err != nil {
		t.Fatal(err)
	}
	list, _, err := service.GetNoticeList(ctx, dto.SearchNoticeReq{})
	if err != nil || len(list) != 1 || list[0].ReceiverCount != 2 || len(list[0].TargetIDs) != 0 {
		t.Fatalf("all target snapshot: %v %v", list, err)
	}
	id := list[0].ID
	database.Where("notice_id = ? AND user_id = ?", id, first.ID).Delete(&model.SysNoticeReceiver{})
	items, total, err := service.GetMyNotices(ctx, first.ID, 1, 10)
	if err != nil || total != 0 || len(items) != 0 {
		t.Fatalf("deleted recipient leaked: %v %v", items, err)
	}
	if err := service.MarkRead(ctx, id, first.ID); err == nil {
		t.Fatal("deleted recipient acknowledged")
	}
	equal := time.Now().Add(time.Hour)
	req.StartAt = &equal
	req.EndAt = &equal
	if err := service.CreateNotice(ctx, req, first.ID); err == nil {
		t.Fatal("zero-length validity accepted")
	}
}
