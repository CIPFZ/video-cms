package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
	"time"
	// Schedules use Beijing time even where the host has no zoneinfo database.
	_ "time/tzdata"
)

var errBusy = errors.New("来源已有任务正在运行")

const pageRetryAttempts = 3

// DefaultFullCron runs the catalogue calibration monthly; the first full
// collection after deployment is started manually from the schedule page.
const DefaultFullCron = "30 3 1 * *"

func leaseToken() string { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func (s *Service) withSource(ctx context.Context, id uint, fn func(context.Context, Source) error) error {
	token := leaseToken()
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&Source{}).Where("id = ? AND (lease_until IS NULL OR lease_until < ?)", id, now).Updates(map[string]any{"lease_token": token, "lease_until": now.Add(12 * time.Minute)})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errBusy
	}
	defer s.db.Model(&Source{}).Where("id = ? AND lease_token = ?", id, token).Updates(map[string]any{"lease_token": "", "lease_until": nil})
	src, err := s.GetSource(ctx, id)
	if err != nil {
		return err
	}
	bounded, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	return fn(bounded, src)
}
func seedSchedules(tx *gorm.DB, src Source) error {
	now := time.Now()
	builtin := IsBuiltinSource(src.APIURL)
	for kind, spec := range map[string]string{"health": "*/20 * * * *", "incremental": "15 */3 * * *", "full": DefaultFullCron} {
		schedule, _ := cron.ParseStandard("CRON_TZ=Asia/Shanghai " + spec)
		next := schedule.Next(now)
		// Built-in sites start collecting on a fresh install: the first incremental run is due now.
		if builtin && kind == "incremental" {
			next = now
		}
		row := CollectionSchedule{SourceID: src.ID, Kind: kind, Cron: spec, Enabled: true, NextRunAt: next}
		if e := tx.Where("source_id = ? AND kind = ?", src.ID, kind).FirstOrCreate(&row).Error; e != nil {
			return e
		}
		if builtin && kind == "incremental" {
			continue
		}
		// Other scheduling is explicitly activated after a trial; existing settings are preserved.
		if row.LastStatus == "" && row.CreatedAt.Equal(row.UpdatedAt) && kind != "health" {
			if e := tx.Model(&row).Updates(map[string]any{"enabled": false, "last_status": "未启用"}).Error; e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *Service) Start(ctx context.Context) error {
	sources, e := s.ListSources(ctx)
	if e != nil {
		return e
	}
	for _, src := range sources {
		if e := seedSchedules(s.db, src); e != nil {
			return e
		}
	}
	child, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-child.Done():
				return
			case <-ticker.C:
				s.tick(child)
			}
		}
	}()
	return nil
}
func (s *Service) Close(ctx context.Context) error {
	if s.cancel == nil {
		return nil
	}
	s.once.Do(s.cancel)
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type CollectOptions struct {
	Mode     string `json:"mode"`
	MaxItems int    `json:"maxItems"`
	MaxPages int    `json:"maxPages"`
}

func (s *Service) Enqueue(ctx context.Context, id uint, opt CollectOptions) (CollectionRun, error) {
	var run CollectionRun
	if opt.Mode == "" {
		opt.Mode = "sample"
	}
	if opt.Mode != "sample" && opt.Mode != "incremental" && opt.Mode != "full" {
		return run, errors.New("不支持的任务类型")
	}
	if opt.MaxItems <= 0 || opt.MaxItems > 1000 {
		opt.MaxItems = 100
	}
	if opt.MaxPages <= 0 || opt.MaxPages > 100 {
		opt.MaxPages = 10
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var src Source
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&src, id).Error; e != nil {
			return e
		}
		if !src.Enabled {
			return errors.New("来源已停用")
		}
		if src.Suspended {
			return errors.New("来源已自动暂停，健康检查连续成功后恢复")
		}
		if opt.Mode == "full" && !src.FullEnabled {
			return errors.New("请先完成小批量验证，再开启来源的全量采集")
		}
		var count int64
		if e := tx.Model(&CollectionRun{}).Where("source_id = ? AND status IN ?", id, []string{"queued", "running"}).Count(&count).Error; e != nil {
			return e
		}
		if count > 0 {
			return errBusy
		}
		run = CollectionRun{SourceID: id, Mode: opt.Mode, StartedAt: time.Now(), Status: "queued", NextPage: 1, MaxItems: opt.MaxItems, MaxPages: opt.MaxPages}
		if opt.Mode == "full" {
			run.MaxItems = 0
			run.MaxPages = 0
			if src.NextPage > 0 {
				run.NextPage = src.NextPage
			}
		}
		if opt.Mode == "incremental" {
			run.MaxItems = 0
			run.MaxPages = 0
			run.Hours = 24
			if src.LastIncrementalAt != nil {
				run.Hours = int(time.Since(*src.LastIncrementalAt).Hours()) + 2
				if run.Hours < 3 {
					run.Hours = 3
				}
			}
		}
		return tx.Create(&run).Error
	})
	return run, err
}
func (s *Service) CancelRun(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run CollectionRun
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&run, id).Error; e != nil {
			return e
		}
		if run.Status == "queued" {
			return tx.Model(&run).Updates(map[string]any{"status": "cancelled", "cancel_requested": true, "finished_at": time.Now()}).Error
		}
		if run.Status != "running" {
			return errors.New("只能暂停排队中或执行中的任务")
		}
		return tx.Model(&run).Update("cancel_requested", true).Error
	})
}
func (s *Service) ListSchedules(ctx context.Context) ([]CollectionSchedule, error) {
	var rows []CollectionSchedule
	err := s.db.WithContext(ctx).Order("source_id,kind").Find(&rows).Error
	return rows, err
}
func (s *Service) UpdateSchedule(ctx context.Context, id uint, spec string, enabled bool) error {
	parsed, e := cron.ParseStandard("CRON_TZ=Asia/Shanghai " + spec)
	if e != nil {
		return errors.New("请输入有效的五段 cron 表达式")
	}
	now := time.Now()
	next := parsed.Next(now)
	if next.IsZero() {
		return errors.New("cron 没有下一次执行时间")
	}
	if len(spec) > 80 {
		return errors.New("cron 表达式过长")
	}
	var row CollectionSchedule
	if e = s.db.WithContext(ctx).First(&row, id).Error; e != nil {
		return e
	}
	if enabled && row.Kind == "full" {
		src, e := s.GetSource(ctx, row.SourceID)
		if e != nil {
			return e
		}
		if !src.FullEnabled {
			return errors.New("此来源尚未开启全量采集")
		}
	}
	return s.db.WithContext(ctx).Model(&row).Updates(map[string]any{"cron": spec, "enabled": enabled, "next_run_at": next}).Error
}
func (s *Service) tick(ctx context.Context) {
	now := time.Now()
	// Recover abandoned jobs after the old source lease expires, retaining their checkpoint.
	s.db.WithContext(ctx).Model(&CollectionRun{}).Where("status = ? AND source_id IN (SELECT id FROM media_sources WHERE lease_until IS NULL OR lease_until < ?) AND updated_at < ?", "running", now, now.Add(-12*time.Minute)).Updates(map[string]any{"status": "queued", "error_message": "CMS 重启或任务超时，已恢复到分页断点"})
	var schedules []CollectionSchedule
	if s.db.WithContext(ctx).Where("enabled = ? AND next_run_at <= ?", true, now).Order("next_run_at").Limit(20).Find(&schedules).Error == nil {
		for _, sch := range schedules {
			parsed, e := cron.ParseStandard("CRON_TZ=Asia/Shanghai " + sch.Cron)
			if e != nil {
				continue
			}
			next := parsed.Next(now)
			r := s.db.WithContext(ctx).Model(&CollectionSchedule{}).Where("id = ? AND next_run_at = ?", sch.ID, sch.NextRunAt).Updates(map[string]any{"next_run_at": next, "last_run_at": now})
			if r.Error != nil || r.RowsAffected != 1 {
				continue
			}
			var status, msg string
			if sch.Kind == "health" {
				status, msg = s.scheduledHealthCheck(ctx, sch.SourceID)
			} else if _, e = s.Enqueue(ctx, sch.SourceID, CollectOptions{Mode: sch.Kind, MaxItems: 1000, MaxPages: 100}); e != nil {
				status, msg = "skipped", e.Error()
			} else {
				status = "queued"
			}
			s.db.WithContext(ctx).Model(&sch).Updates(map[string]any{"last_status": status, "last_error": msg})
		}
	}
	var run CollectionRun
	if s.db.WithContext(ctx).Where("status = ?", "queued").Order("updated_at, id").First(&run).Error == nil {
		claim := s.db.WithContext(ctx).Model(&run).Where("status = ?", "queued").Updates(map[string]any{"status": "running"})
		if claim.Error == nil && claim.RowsAffected == 1 {
			s.execute(ctx, &run)
		}
	}
	// Indexed retention, small deletes keep the web workload responsive.
	if now.Minute() == 0 {
		var ids []uint
		s.db.Model(&HealthCheck{}).Where("checked_at < ?", now.AddDate(0, 0, -60)).Limit(500).Pluck("id", &ids)
		if len(ids) > 0 {
			s.db.Where("id IN ?", ids).Delete(&HealthCheck{})
		}
		ids = nil
		s.db.Model(&CollectionRun{}).Where("finished_at < ? AND status NOT IN ?", now.AddDate(0, 0, -30), []string{"running", "queued"}).Limit(500).Pluck("id", &ids)
		if len(ids) > 0 {
			s.db.Where("collection_run_id IN ?", ids).Delete(&CollectionRunItem{})
			s.db.Where("collection_run_id IN ?", ids).Delete(&CollectionPageFailure{})
			s.db.Where("id IN ?", ids).Delete(&CollectionRun{})
		}
	}
}

// scheduledHealthCheck records the probe in the health history only; it is
// not a collection run and does not appear in the collection log.
func (s *Service) scheduledHealthCheck(ctx context.Context, sourceID uint) (string, string) {
	src, e := s.GetSource(ctx, sourceID)
	if e != nil {
		return "skipped", e.Error()
	}
	if !src.Enabled {
		return "skipped", "来源已停用"
	}
	check, e := s.HealthCheck(ctx, sourceID)
	switch {
	case e != nil:
		return "skipped", e.Error()
	case check.Result != "ok":
		return "failed", check.ErrorMessage
	}
	return "success", ""
}
func (s *Service) execute(ctx context.Context, run *CollectionRun) {
	status := "success"
	var err error
	// Health checks are recorded in their own history; this only drains runs queued before that change.
	if run.Mode == "health" {
		var check HealthCheck
		check, err = s.HealthCheck(ctx, run.SourceID)
		if err == nil && check.Result != "ok" {
			err = errors.New(check.ErrorMessage)
		}
	} else {
		err = s.withSource(ctx, run.SourceID, func(ctx context.Context, src Source) error {
			if !src.Enabled || src.Suspended {
				return errors.New("来源已停用或暂停")
			}
			return s.collectPages(ctx, src, run, &status)
		})
	}
	if errors.Is(err, errBusy) || errors.Is(err, context.Canceled) {
		status = "queued"
		err = nil
	}
	if err != nil {
		status = "failed"
		run.ErrorMessage = err.Error()
		run.ErrorCount++
	}
	updates := map[string]any{"status": status, "pages": run.Pages, "fetched": run.Fetched, "inserted_count": run.InsertedCount, "updated_count": run.UpdatedCount, "skipped_count": run.SkippedCount, "error_count": run.ErrorCount, "error_message": run.ErrorMessage, "next_page": run.NextPage}
	if status != "queued" {
		updates["finished_at"] = time.Now()
	}
	s.db.Model(run).Updates(updates)
	s.db.Model(&CollectionSchedule{}).Where("source_id = ? AND kind = ?", run.SourceID, run.Mode).Updates(map[string]any{"last_status": status, "last_error": run.ErrorMessage})
}
func (s *Service) collectPages(ctx context.Context, src Source, run *CollectionRun, status *string) error {
	if run.Mode == "full" && src.FullStartedAt == nil {
		now := time.Now()
		src.FullStartedAt = &now
		src.FullHadErrors = false
		if e := s.db.WithContext(ctx).Model(&src).Updates(map[string]any{"full_started_at": now, "full_had_errors": false, "next_page": 1}).Error; e != nil {
			return e
		}
	}
	for chunk := 0; chunk < 5; chunk++ {
		var current CollectionRun
		if e := s.db.WithContext(ctx).First(&current, run.ID).Error; e != nil {
			return e
		}
		if current.CancelRequested {
			*status = "cancelled"
			return nil
		}
		fresh, e := s.GetSource(ctx, src.ID)
		if e != nil {
			return e
		}
		if !fresh.Enabled {
			return errors.New("来源已停用")
		}
		page, e := s.retryPage(ctx, src, run.NextPage, run.Hours)
		if e != nil {
			// A page-level upstream timeout must not abort the entire collection.
			// Preserve the failure, advance the checkpoint, and continue with the next page.
			if errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded) {
				return e
			}
			failedPage := run.NextPage
			run.Pages++
			run.ErrorCount++
			run.ErrorMessage = e.Error()
			if saveErr := s.db.WithContext(ctx).Create(&CollectionPageFailure{
				CollectionRunID: run.ID,
				Page:            failedPage,
				Attempts:        pageRetryAttempts,
				ErrorMessage:    e.Error(),
			}).Error; saveErr != nil {
				return saveErr
			}
			if run.Mode == "full" {
				src.FullHadErrors = true
				if saveErr := s.db.WithContext(ctx).Model(&src).Updates(map[string]any{
					"next_page": run.NextPage + 1, "full_had_errors": true,
				}).Error; saveErr != nil {
					return saveErr
				}
			}
			run.NextPage++
			if saveErr := s.db.WithContext(ctx).Model(run).Updates(map[string]any{
				"next_page": run.NextPage, "pages": run.Pages, "error_count": run.ErrorCount,
				"error_message": run.ErrorMessage,
			}).Error; saveErr != nil {
				return saveErr
			}
			if run.MaxPages > 0 && run.Pages >= run.MaxPages {
				*status = "partial"
				return nil
			}
			if e = pause(ctx, time.Duration(src.RequestIntervalSeconds)*time.Second); e != nil {
				return e
			}
			continue
		}
		if page.Page > 0 && page.Page != run.NextPage {
			return errors.New("来源未按请求返回分页")
		}
		records := page.Records
		completePage := true
		if run.MaxItems > 0 && len(records) > run.MaxItems-run.Fetched {
			records = records[:run.MaxItems-run.Fetched]
			completePage = false
		}
		wanted := []string{}
		hashes := map[string]string{}
		titles := map[string]string{}
		for _, r := range records {
			id := firstString(r, "vod_id")
			hash := hashRecord(r)
			hashes[id] = hash
			titles[id] = firstString(r, "vod_name")
			run.Fetched++
			if !categoryAllowed(src, r) {
				run.SkippedCount++
				continue
			}
			var old SourceItem
			e := s.db.WithContext(ctx).Where("source_id = ? AND external_id = ?", src.ID, id).First(&old).Error
			if e == nil && old.ListHash == hash && run.Mode != "full" {
				if e = s.db.WithContext(ctx).Model(&old).Updates(map[string]any{"last_seen_at": time.Now(), "state": "active"}).Error; e != nil {
					return e
				}
				run.SkippedCount++
				continue
			}
			if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
				return e
			}
			wanted = append(wanted, id)
		}
		if len(wanted) > 0 {
			if e = pause(ctx, time.Duration(src.RequestIntervalSeconds)*time.Second); e != nil {
				return e
			}
			details, e := s.retryDetails(ctx, src, wanted)
			if e != nil {
				return e
			}
			returned := map[string]bool{}
			logged := []CollectionRunItem{}
			entry := func(id, action string, itemID *uint, message string) CollectionRunItem {
				return CollectionRunItem{CollectionRunID: run.ID, Action: action, ItemID: itemID, ExternalID: clipped(id, 100), Title: clipped(titles[id], 500), ErrorMessage: message}
			}
			for _, r := range details {
				id := firstString(r, "vod_id")
				hash, ok := hashes[id]
				if !ok || returned[id] {
					continue
				}
				returned[id] = true
				outcome, itemID, e := s.saveRecord(ctx, src, r, hash)
				if e != nil {
					run.ErrorCount++
					run.ErrorMessage = fmt.Sprintf("条目 %s 保存失败: %v", id, e)
					logged = append(logged, entry(id, "failed", nil, "保存失败: "+e.Error()))
					continue
				}
				switch outcome {
				case "inserted":
					run.InsertedCount++
					logged = append(logged, entry(id, "inserted", &itemID, ""))
				case "updated":
					run.UpdatedCount++
					logged = append(logged, entry(id, "updated", &itemID, ""))
				default:
					run.SkippedCount++
				}
			}
			for _, id := range wanted {
				if !returned[id] {
					run.ErrorCount++
					run.ErrorMessage = "来源详情响应缺少请求的 ID"
					logged = append(logged, entry(id, "failed", nil, "来源详情响应缺少该条目"))
				}
			}
			if len(logged) > 0 {
				if e := s.db.WithContext(ctx).CreateInBatches(logged, 100).Error; e != nil {
					return e
				}
			}
		}
		run.Pages++
		ended := len(page.Records) == 0 || (page.Pages > 0 && run.NextPage >= page.Pages)
		if run.Mode == "full" {
			if run.ErrorCount > 0 {
				src.FullHadErrors = true
			}
			if ended && completePage {
				if !src.FullHadErrors {
					if e := s.db.WithContext(ctx).Model(&SourceItem{}).Where("source_id = ? AND last_seen_at < ? AND state = ?", src.ID, src.FullStartedAt, "active").Update("state", "missing").Error; e != nil {
						return e
					}
				}
				finished := map[string]any{"next_page": 0, "full_started_at": nil, "full_had_errors": false}
				// The next incremental run continues from the start of this traversal.
				if src.FullStartedAt != nil && (src.LastIncrementalAt == nil || src.LastIncrementalAt.Before(*src.FullStartedAt)) {
					finished["last_incremental_at"] = *src.FullStartedAt
				}
				if e := s.db.WithContext(ctx).Model(&src).Updates(finished).Error; e != nil {
					return e
				}
			} else {
				if e := s.db.WithContext(ctx).Model(&src).Updates(map[string]any{"next_page": run.NextPage + 1, "full_had_errors": src.FullHadErrors}).Error; e != nil {
					return e
				}
			}
		}
		if ended && completePage {
			if run.ErrorCount > 0 {
				*status = "partial"
			} else if run.Mode == "incremental" {
				if e := s.db.WithContext(ctx).Model(&src).Update("last_incremental_at", run.StartedAt).Error; e != nil {
					return e
				}
			}
			run.NextPage = 0
			return nil
		}
		if !completePage || (run.MaxItems > 0 && run.Fetched >= run.MaxItems) || (run.MaxPages > 0 && run.Pages >= run.MaxPages) {
			if run.Mode == "sample" {
				*status = "success"
			} else {
				*status = "partial"
			}
			return nil
		}
		run.NextPage++
		if e := s.db.WithContext(ctx).Model(run).Updates(map[string]any{"next_page": run.NextPage, "pages": run.Pages, "fetched": run.Fetched, "inserted_count": run.InsertedCount, "updated_count": run.UpdatedCount, "skipped_count": run.SkippedCount, "error_count": run.ErrorCount}).Error; e != nil {
			return e
		}
		if e = pause(ctx, time.Duration(src.RequestIntervalSeconds)*time.Second); e != nil {
			return e
		}
	}
	*status = "queued"
	return nil
}
func (s *Service) retryPage(ctx context.Context, src Source, page, hours int) (upstreamPage, error) {
	var out upstreamPage
	var err error
	for attempt := 0; attempt < pageRetryAttempts; attempt++ {
		if attempt > 0 {
			if e := pause(ctx, time.Duration(src.RequestIntervalSeconds+attempt*2)*time.Second); e != nil {
				return out, e
			}
		}
		out, err = s.fetchPage(ctx, src, page, hours)
		if err == nil {
			return out, nil
		}
	}
	return out, fmt.Errorf("第 %s 页失败: %w", strconv.Itoa(page), err)
}
func (s *Service) retryDetails(ctx context.Context, src Source, ids []string) ([]rawRecord, error) {
	var out []rawRecord
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			if e := pause(ctx, time.Duration(src.RequestIntervalSeconds+attempt*2)*time.Second); e != nil {
				return out, e
			}
		}
		out, err = s.fetchDetails(ctx, src, ids)
		if err == nil {
			return out, nil
		}
	}
	return out, err
}
