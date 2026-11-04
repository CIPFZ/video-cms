package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var brackets = regexp.MustCompile(`[[](){}【】（）「」《》]`)
var sourceMarkers = regexp.MustCompile(`(?i)[[【(（][^]】)）]*(?:资源|1080p|2160p|4k|高清)[^]】)）]*[]】)）]`)
var htmlTags = regexp.MustCompile("<[^>]*>")

func clipped(v string, n int) string {
	r := []rune(v)
	if len(r) > n {
		return string(r[:n])
	}
	return v
}
func normalizeKey(title, year, kind, area string) string {
	title = sourceMarkers.ReplaceAllString(strings.ToLower(title), "")
	title = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, title)
	title = brackets.ReplaceAllString(title, "")
	for _, m := range []string{"1080p", "2160p", "4k", "高清", "无删减"} {
		title = strings.ReplaceAll(title, m, "")
	}
	return strings.Join([]string{title, strings.TrimSpace(year), kind, normalizeArea(area)}, "|")
}
func normalizeArea(a string) string {
	a = strings.TrimSpace(a)
	switch a {
	case "大陆", "中国大陆", "内地":
		return "中国大陆"
	case "香港", "中国香港":
		return "中国香港"
	case "台湾", "中国台湾":
		return "中国台湾"
	}
	return a
}

// normalizeGenres keeps the upstream genre order while removing duplicate labels.
// Most MacCMS-compatible APIs expose this value as vod_class, but a few sources
// use vod_tag(s) or vod_genre(s), so the mapper accepts all common spellings.
func normalizeGenres(value string) string {
	seen := map[string]struct{}{}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == '/' || r == '|' || r == '、' || r == ';' || r == '；' || r == '\n'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(htmlTags.ReplaceAllString(part, ""))
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, clipped(part, 40))
	}
	return strings.Join(out, ",")
}

func recordGenres(r rawRecord) string {
	for _, key := range []string{"vod_class", "vod_genres", "vod_genre", "vod_tags", "vod_tag"} {
		if value := normalizeGenres(firstString(r, key)); value != "" {
			return value
		}
	}
	return ""
}
func mapMediaType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, word := range []string{"伦理", "成人", "色情", "直播", "短剧", "短视频", "解说", "体育", "足球", "篮球"} {
		if strings.Contains(value, word) {
			return ""
		}
	}
	switch {
	case strings.Contains(value, "动漫") || strings.Contains(value, "动画") || value == "anime":
		return "anime"
	case strings.Contains(value, "综艺") || value == "variety":
		return "variety"
	case (strings.HasSuffix(value, "剧") && !strings.HasSuffix(value, "喜剧")) || value == "series":
		return "series"
	}
	for _, v := range []string{"电影", "动作", "喜剧", "爱情", "科幻", "恐怖", "战争", "剧情", "惊悚", "悬疑", "犯罪", "纪录", "记录", "家庭", "历史", "冒险", "奇幻", "灾难", "西部", "古装", "传记", "movie"} {
		if strings.Contains(value, v) {
			return "movie"
		}
	}
	return ""
}
func categoryAllowed(src Source, r rawRecord) bool {
	cat := firstString(r, "type_name")
	kind := mapMediaType(cat)
	if kind == "" {
		return false
	}
	matches := func(list string) bool {
		for _, v := range strings.FieldsFunc(list, func(r rune) bool { return r == ',' || r == '，' || r == '\n' }) {
			v = strings.TrimSpace(v)
			if v != "" && (v == cat || v == firstString(r, "type_id") || v == kind || map[string]string{"movie": "电影", "series": "电视剧", "anime": "动漫", "variety": "综艺"}[kind] == v) {
				return true
			}
		}
		return false
	}
	return !matches(src.DenyCategories) && (strings.TrimSpace(src.AllowCategories) == "" || matches(src.AllowCategories))
}
func (s *Service) saveRecord(ctx context.Context, src Source, r rawRecord, listHash string) (string, uint, error) {
	if !categoryAllowed(src, r) {
		return "skipped", 0, nil
	}
	title := clipped(firstString(r, "vod_name"), 500)
	externalID := firstString(r, "vod_id")
	if title == "" || externalID == "" || len(externalID) > 100 {
		return "", 0, errors.New("条目缺少标题或稳定 ID")
	}
	kind := mapMediaType(firstString(r, "type_name"))
	year := clipped(firstString(r, "vod_year"), 20)
	area := clipped(normalizeArea(firstString(r, "vod_area")), 100)
	key := normalizeKey(title, year, kind, area)
	status := "active"
	if year == "" || year == "0" {
		key += fmt.Sprintf("|source:%d:%s", src.ID, externalID)
		status = "pending_merge"
	}
	if len([]rune(key)) > 690 {
		h := sha256.Sum256([]byte(key))
		key = hex.EncodeToString(h[:])
	}
	incoming := Item{Title: title, OriginalTitle: clipped(firstString(r, "vod_sub", "vod_name"), 500), MediaType: kind, Genres: recordGenres(r), Year: year, Area: area,
		Language: clipped(firstString(r, "vod_lang"), 100), Director: clipped(firstString(r, "vod_director"), 2000), Actors: clipped(firstString(r, "vod_actor"), 4000),
		Description: clipped(htmlTags.ReplaceAllString(firstString(r, "vod_content"), ""), 12000), CoverURL: clipped(firstString(r, "vod_pic"), 1000),
		NormalizedKey: key, Status: status, MetadataSourceID: src.ID}
	if validateURL(incoming.CoverURL) != nil {
		incoming.CoverURL = ""
	}
	result := "updated"
	var itemID uint
	now := time.Now()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var link SourceItem
		e := tx.Where("source_id = ? AND external_id = ?", src.ID, externalID).First(&link).Error
		fresh := errors.Is(e, gorm.ErrRecordNotFound)
		if e != nil && !fresh {
			return e
		}
		var item Item
		if !fresh && link.CMSMediaID != nil {
			if e := tx.First(&item, *link.CMSMediaID).Error; e != nil {
				return e
			}
		} else {
			created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&incoming)
			if created.Error != nil {
				return created.Error
			}
			if created.RowsAffected > 0 {
				result = "inserted"
			}
			if e := tx.Where("normalized_key = ?", key).First(&item).Error; e != nil {
				return e
			}
		}
		itemID = item.ID
		canUpdate := item.MetadataSourceID == src.ID || item.MetadataSourceID == 0
		if !canUpdate {
			var owner Source
			if e := tx.First(&owner, item.MetadataSourceID).Error; errors.Is(e, gorm.ErrRecordNotFound) {
				canUpdate = true
			} else if e != nil {
				return e
			} else {
				canUpdate = src.Priority < owner.Priority
			}
		}
		if canUpdate {
			if e := tx.Model(&item).Updates(map[string]any{"title": incoming.Title, "original_title": incoming.OriginalTitle, "media_type": kind, "genres": incoming.Genres, "year": year, "area": area, "language": incoming.Language, "director": incoming.Director, "actors": incoming.Actors, "description": incoming.Description, "cover_url": incoming.CoverURL, "metadata_source_id": src.ID}).Error; e != nil {
				return e
			}
		}
		hash := hashRecord(r)
		if fresh {
			link = SourceItem{SourceID: src.ID, CMSMediaID: &item.ID, ExternalID: externalID, ExternalTitle: title, FirstSeenAt: now, LastSeenAt: now, RawHash: hash, ListHash: listHash, State: "active"}
			if e := tx.Create(&link).Error; e != nil {
				return e
			}
		} else {
			if link.RawHash == hash {
				result = "skipped"
			}
			if e := tx.Model(&link).Updates(map[string]any{"external_title": title, "last_seen_at": now, "raw_hash": hash, "list_hash": listHash, "state": "active"}).Error; e != nil {
				return e
			}
			if result == "skipped" {
				return nil
			}
		}
		// Store only necessary metadata, never the upstream response body or manifests.
		if t, e := time.ParseInLocation("2006-01-02 15:04:05", firstString(r, "vod_time"), time.FixedZone("CST", 8*3600)); e == nil {
			if e = tx.Model(&link).Update("source_updated_at", t).Error; e != nil {
				return e
			}
		}
		return savePlayback(tx, link.ID, r)
	})
	return result, itemID, err
}
func splitValues(v string) []string { return strings.Split(v, "$$$") }
func splitEpisode(v string) (string, string) {
	p := strings.SplitN(v, "$", 2)
	if len(p) == 1 {
		return strings.TrimSpace(p[0]), ""
	}
	return strings.TrimSpace(p[1]), strings.TrimSpace(p[0])
}
func streamType(v string) string {
	u, e := url.Parse(v)
	if e != nil {
		return "other"
	}
	p := strings.ToLower(u.Path)
	if strings.HasSuffix(p, ".m3u8") {
		return "m3u8"
	}
	if strings.HasSuffix(p, ".mp4") {
		return "mp4"
	}
	return "other"
}
func savePlayback(tx *gorm.DB, id uint, r rawRecord) error {
	froms := splitValues(firstString(r, "vod_play_from"))
	urls := splitValues(firstString(r, "vod_play_url"))
	if len(froms) != len(urls) {
		return errors.New("线路名称与地址组数量不一致")
	}
	// A complete detail response replaces this source's inventory only.
	if e := tx.Model(&PlayLine{}).Where("source_item_id = ?", id).Update("enabled", false).Error; e != nil {
		return e
	}
	for i, group := range urls {
		if group == "" || froms[i] == "" {
			continue
		}
		name := clipped(froms[i], 100)
		line := PlayLine{SourceItemID: id, LineName: name, LineKey: name, SortOrder: i + 1, Enabled: true}
		if e := tx.Where("source_item_id = ? AND line_key = ?", id, name).FirstOrCreate(&line).Error; e != nil {
			return e
		}
		if e := tx.Model(&line).Updates(map[string]any{"enabled": true, "sort_order": i + 1}).Error; e != nil {
			return e
		}
		seen := []int{}
		for n, v := range strings.Split(group, "#") {
			address, label := splitEpisode(v)
			if validateURL(address) != nil {
				continue
			}
			h := sha256.Sum256([]byte(address))
			episode := PlayEpisode{PlayLineID: line.ID, EpisodeNo: n + 1, EpisodeName: clipped(label, 255), StreamURL: address, StreamType: streamType(address), URLHash: hex.EncodeToString(h[:])}
			var old PlayEpisode
			e := tx.Where("play_line_id = ? AND episode_no = ?", line.ID, n+1).First(&old).Error
			if errors.Is(e, gorm.ErrRecordNotFound) {
				if e = tx.Create(&episode).Error; e != nil {
					return e
				}
			} else if e != nil {
				return e
			} else {
				updates := map[string]any{"episode_name": episode.EpisodeName, "stream_url": address, "stream_type": episode.StreamType, "url_hash": episode.URLHash}
				if old.URLHash != episode.URLHash {
					updates["last_checked_at"] = nil
					updates["last_ok_at"] = nil
					updates["last_status"] = ""
				}
				if e = tx.Model(&old).Updates(updates).Error; e != nil {
					return e
				}
			}
			seen = append(seen, n+1)
		}
		q := tx.Where("play_line_id = ?", line.ID)
		if len(seen) > 0 {
			q = q.Where("episode_no NOT IN ?", seen)
		}
		if e := q.Delete(&PlayEpisode{}).Error; e != nil {
			return e
		}
	}
	return nil
}
