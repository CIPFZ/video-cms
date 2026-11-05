package media

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// The public site reads through these shapes only: no source API addresses,
// raw upstream metadata, leases or other back-office fields.

// PublicCard is one poster on the public site.
type PublicCard struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	MediaType string    `json:"mediaType"`
	Year      string    `json:"year,omitempty"`
	Area      string    `json:"area,omitempty"`
	Genres    string    `json:"genres,omitempty"`
	CoverURL  string    `json:"coverUrl,omitempty"`
	Episodes  int       `json:"episodes"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PublicEpisode struct {
	ID        uint   `json:"id"`
	EpisodeNo int    `json:"episodeNo"`
	Name      string `json:"name,omitempty"`
	URL       string `json:"url"`
	Type      string `json:"type"`
}

// PublicLine is numbered on the site; the upstream line name is kept as a hint.
type PublicLine struct {
	ID       uint            `json:"id"`
	Name     string          `json:"name"`
	Episodes []PublicEpisode `json:"episodes"`
}

type PublicDetail struct {
	PublicCard
	OriginalTitle string       `json:"originalTitle,omitempty"`
	Language      string       `json:"language,omitempty"`
	Director      string       `json:"director,omitempty"`
	Actors        string       `json:"actors,omitempty"`
	Description   string       `json:"description,omitempty"`
	Lines         []PublicLine `json:"lines"`
}

type PublicFacet struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type PublicFacets struct {
	Years  []PublicFacet `json:"years"`
	Areas  []PublicFacet `json:"areas"`
	Genres []PublicFacet `json:"genres"`
}

type PublicRow struct {
	Key   string       `json:"key"`
	Items []PublicCard `json:"items"`
}

// PublicFilter is the public catalogue query; every field comes from the URL.
type PublicFilter struct {
	Type     string
	Years    []string // four-digit years, or "none" for unknown
	Area     string
	Genre    string
	Query    string
	Sort     string // "updated" (default) or "year"
	Page     int
	PageSize int
}

const (
	publicMaxPageSize = 48
	facetCacheTTL     = 10 * time.Minute
	noYear            = "none"
)

var publicYear = regexp.MustCompile(`^(19|20)\d{2}$`)

// playable keeps titles that an enabled, running source still offers.
func (s *Service) playable(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx).Model(&Item{}).Where(`EXISTS (SELECT 1 FROM media_source_items si JOIN media_sources src ON src.id = si.source_id
		WHERE si.cms_media_id = media_items.id AND si.state = ? AND src.enabled = ? AND src.suspended = ?)`, "active", true, false)
}

// listMatch matches one value of a comma-separated column, as media_type does in ListItems.
func listMatch(db *gorm.DB, column, value string) *gorm.DB {
	return db.Where(column+" = ? OR "+column+" LIKE ? OR "+column+" LIKE ? OR "+column+" LIKE ?", value, value+",%", "%,"+value+",%", "%,"+value)
}

func (s *Service) ListPublic(ctx context.Context, f PublicFilter) (map[string]any, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > publicMaxPageSize {
		f.PageSize = 24
	}
	db := s.playable(ctx)
	if f.Type != "" {
		db = listMatch(db, "media_type", f.Type)
	}
	years, unknown := []string{}, false
	for _, y := range f.Years {
		if y == noYear {
			unknown = true
		} else if publicYear.MatchString(y) {
			years = append(years, y)
		}
	}
	switch {
	case len(years) > 0 && unknown:
		db = db.Where("year IN ? OR year IN ? OR year IS NULL", years, []string{"", "0"})
	case len(years) > 0:
		db = db.Where("year IN ?", years)
	case unknown:
		db = db.Where("year IN ? OR year IS NULL", []string{"", "0"})
	}
	if f.Area != "" {
		db = db.Where("area = ?", f.Area)
	}
	if f.Genre != "" {
		db = listMatch(db, "genres", f.Genre)
	}
	if q := publicQuery(f.Query); q != "" {
		pattern := "%" + q + "%"
		db = db.Where("title LIKE ? OR original_title LIKE ? OR director LIKE ? OR actors LIKE ?", pattern, pattern, pattern, pattern)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	order := "media_items.updated_at desc, media_items.id desc"
	if f.Sort == "year" {
		order = "media_items.year desc, " + order
	}
	var rows []Item
	if err := db.Order(order).Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	cards, err := s.publicCards(ctx, rows)
	return map[string]any{"list": cards, "total": total, "page": f.Page, "pageSize": f.PageSize}, err
}

// publicQuery drops LIKE wildcards so a search term is matched literally.
func publicQuery(q string) string {
	q = strings.NewReplacer("%", "", "_", "").Replace(strings.TrimSpace(q))
	if r := []rune(q); len(r) > 50 {
		q = string(r[:50])
	}
	return q
}

func (s *Service) PublicHome(ctx context.Context) ([]PublicRow, error) {
	rows := []PublicRow{}
	for _, key := range []string{"latest", "movie", "series", "anime", "variety"} {
		f := PublicFilter{PageSize: 12}
		if key != "latest" {
			f.Type = key
		}
		page, err := s.ListPublic(ctx, f)
		if err != nil {
			return nil, err
		}
		if cards := page["list"].([]PublicCard); len(cards) > 0 {
			rows = append(rows, PublicRow{Key: key, Items: cards})
		}
	}
	return rows, nil
}

// publicCards adds each title's longest playable line length, in one query per page.
func (s *Service) publicCards(ctx context.Context, rows []Item) ([]PublicCard, error) {
	cards := make([]PublicCard, len(rows))
	ids := make([]uint, len(rows))
	for i, item := range rows {
		ids[i] = item.ID
		cards[i] = publicCard(item)
	}
	if len(ids) == 0 {
		return cards, nil
	}
	var counts []struct {
		ItemID uint
		N      int
	}
	err := s.db.WithContext(ctx).Table("media_source_items si").
		Select("si.cms_media_id AS item_id, COUNT(e.id) AS n").
		Joins("JOIN media_sources src ON src.id = si.source_id").
		Joins("JOIN media_play_lines l ON l.source_item_id = si.id").
		Joins("JOIN media_play_episodes e ON e.play_line_id = l.id").
		Where("si.cms_media_id IN ? AND si.state = ? AND src.enabled = ? AND src.suspended = ? AND l.enabled = ?", ids, "active", true, false, true).
		Group("si.cms_media_id, l.id").Scan(&counts).Error
	if err != nil {
		return nil, err
	}
	longest := map[uint]int{}
	for _, c := range counts {
		longest[c.ItemID] = max(longest[c.ItemID], c.N)
	}
	for i := range cards {
		cards[i].Episodes = longest[cards[i].ID]
	}
	return cards, nil
}

func publicCard(item Item) PublicCard {
	return PublicCard{ID: item.ID, Title: item.Title, MediaType: item.MediaType, Year: item.Year, Area: item.Area,
		Genres: item.Genres, CoverURL: item.CoverURL, UpdatedAt: item.UpdatedAt}
}

func (s *Service) PublicDetail(ctx context.Context, id uint) (PublicDetail, error) {
	detail, err := s.GetItemDetail(ctx, id)
	if err != nil {
		return PublicDetail{}, err
	}
	item := detail.Item
	out := PublicDetail{PublicCard: publicCard(item), OriginalTitle: item.OriginalTitle, Language: item.Language,
		Director: item.Director, Actors: item.Actors, Description: item.Description, Lines: []PublicLine{}}
	for _, playback := range detail.Playbacks {
		for _, line := range playback.Lines {
			if len(line.Episodes) == 0 {
				continue
			}
			episodes := make([]PublicEpisode, len(line.Episodes))
			for i, ep := range line.Episodes {
				episodes[i] = PublicEpisode{ID: ep.ID, EpisodeNo: ep.EpisodeNo, Name: ep.EpisodeName, URL: ep.StreamURL, Type: ep.StreamType}
			}
			out.Lines = append(out.Lines, PublicLine{ID: line.ID, Name: line.LineName, Episodes: episodes})
			out.Episodes = max(out.Episodes, len(episodes))
		}
	}
	return out, nil
}

// PublicFacets counts the whole playable catalogue and is cached briefly; the
// counts guide the filter bar and need not follow every new title at once.
func (s *Service) PublicFacets(ctx context.Context) (PublicFacets, error) {
	s.facetMu.Lock()
	defer s.facetMu.Unlock()
	if s.facets != nil && time.Since(s.facetsAt) < facetCacheTTL {
		return *s.facets, nil
	}
	var years []struct {
		Value string
		Count int64
	}
	if err := s.playable(ctx).Select("year AS value, COUNT(*) AS count").Group("year").Scan(&years).Error; err != nil {
		return PublicFacets{}, err
	}
	out := PublicFacets{Years: []PublicFacet{}, Areas: []PublicFacet{}, Genres: []PublicFacet{}}
	var unknown int64
	for _, y := range years {
		if publicYear.MatchString(y.Value) {
			out.Years = append(out.Years, PublicFacet{y.Value, y.Count})
		} else {
			unknown += y.Count
		}
	}
	sort.Slice(out.Years, func(i, j int) bool { return out.Years[i].Value > out.Years[j].Value })
	if unknown > 0 {
		out.Years = append(out.Years, PublicFacet{noYear, unknown})
	}
	if err := s.playable(ctx).Select("area AS value, COUNT(*) AS count").Where("area <> ?", "").
		Group("area").Order("count desc, area").Limit(20).Scan(&out.Areas).Error; err != nil {
		return PublicFacets{}, err
	}
	var genreLists []string
	if err := s.playable(ctx).Where("genres <> ?", "").Pluck("genres", &genreLists).Error; err != nil {
		return PublicFacets{}, err
	}
	genres := map[string]int64{}
	for _, list := range genreLists {
		for _, g := range strings.Split(list, ",") {
			if g = strings.TrimSpace(g); g != "" {
				genres[g]++
			}
		}
	}
	for g, n := range genres {
		out.Genres = append(out.Genres, PublicFacet{g, n})
	}
	sort.Slice(out.Genres, func(i, j int) bool {
		if out.Genres[i].Count != out.Genres[j].Count {
			return out.Genres[i].Count > out.Genres[j].Count
		}
		return out.Genres[i].Value < out.Genres[j].Value
	})
	if len(out.Genres) > 30 {
		out.Genres = out.Genres[:30]
	}
	s.facets, s.facetsAt = &out, time.Now()
	return out, nil
}

var errEpisodeUnavailable = errors.New("episode unavailable")

// publicEpisode admits the proxy only for episodes the public detail would list.
func (s *Service) publicEpisode(ctx context.Context, id uint) error {
	var n int64
	err := s.db.WithContext(ctx).Table("media_play_episodes e").
		Joins("JOIN media_play_lines l ON l.id = e.play_line_id").
		Joins("JOIN media_source_items si ON si.id = l.source_item_id").
		Joins("JOIN media_sources src ON src.id = si.source_id").
		Where("e.id = ? AND l.enabled = ? AND si.state = ? AND src.enabled = ? AND src.suspended = ?", id, true, "active", true, false).
		Count(&n).Error
	if err == nil && n == 0 {
		err = errEpisodeUnavailable
	}
	return err
}
