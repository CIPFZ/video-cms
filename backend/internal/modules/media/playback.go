package media

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *Service) RefreshItem(ctx context.Context, id uint) error {
	var links []SourceItem
	if e := s.db.WithContext(ctx).Where("cms_media_id = ? AND state = ?", id, "active").Find(&links).Error; e != nil {
		return e
	}
	var success bool
	for _, link := range links {
		e := s.withSource(ctx, link.SourceID, func(ctx context.Context, src Source) error {
			if !src.Enabled || src.Suspended {
				return errors.New("来源不可用")
			}
			rows, e := s.fetchDetails(ctx, src, []string{link.ExternalID})
			if e != nil {
				return e
			}
			for _, row := range rows {
				if firstString(row, "vod_id") == link.ExternalID {
					_, _, e = s.saveRecord(ctx, src, row, link.ListHash)
					return e
				}
			}
			return errors.New("详情中没有该资源")
		})
		if e == nil {
			success = true
		}
	}
	if !success {
		return errors.New("刷新失败：来源不可用或正忙")
	}
	return nil
}
func (s *Service) CheckEpisode(ctx context.Context, id uint) (map[string]any, error) {
	var ep PlayEpisode
	if e := s.db.WithContext(ctx).First(&ep, id).Error; e != nil {
		return nil, e
	}
	var line PlayLine
	if e := s.db.WithContext(ctx).First(&line, ep.PlayLineID).Error; e != nil {
		return nil, e
	}
	var link SourceItem
	if e := s.db.WithContext(ctx).First(&link, line.SourceItemID).Error; e != nil {
		return nil, e
	}
	src, e := s.GetSource(ctx, link.SourceID)
	if e != nil {
		return nil, e
	}
	if !src.Enabled || src.Suspended || !line.Enabled || link.State != "active" {
		return nil, errors.New("线路已停用")
	}
	if validateURL(ep.StreamURL) != nil {
		return nil, errors.New("播放地址无效")
	}
	if ep.StreamType != "m3u8" && ep.StreamType != "mp4" {
		return nil, errors.New("仅检查 m3u8/MP4 直链")
	}
	// A short probe verifies headers and at most 64 KiB of a playlist, never segments.
	method := "HEAD"
	if ep.StreamType == "m3u8" {
		method = "GET"
	}
	req, e := http.NewRequestWithContext(ctx, method, ep.StreamURL, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", "VideoCMS-MediaCollector/1.0")
	start := time.Now()
	status := "failed"
	httpStatus := 0
	client := *s.client
	client.Timeout = 10 * time.Second
	resp, e := client.Do(req)
	if e == nil {
		httpStatus = resp.StatusCode
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if method == "HEAD" && strings.HasPrefix(resp.Header.Get("Content-Type"), "video/") {
				status = "ok"
			}
			if method == "GET" {
				body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
				if err == nil && strings.HasPrefix(strings.TrimSpace(string(body)), "#EXTM3U") {
					status = "ok"
				}
			}
		}
		resp.Body.Close()
	}
	latency := int(time.Since(start).Milliseconds())
	updates := map[string]any{"last_checked_at": start, "last_status": status}
	if status == "ok" {
		updates["last_ok_at"] = start
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&ep).Updates(updates).Error; e != nil {
			return e
		}
		updates["last_latency_ms"] = latency
		return tx.Model(&line).Updates(updates).Error
	})
	return map[string]any{"status": status, "httpStatus": httpStatus, "latencyMs": latency}, err
}
