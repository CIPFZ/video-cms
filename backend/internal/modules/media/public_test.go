package media

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type publicFixture struct {
	svc                    *Service
	db                     *gorm.DB
	live, off, paused      Source
	movie, series, hidden  Item
	unknownYear, offerless Item
}

// seed gives an item a source link and a line with n episodes.
func (f *publicFixture) seed(t *testing.T, item *Item, src Source, n int) {
	t.Helper()
	require.NoError(t, f.db.Create(item).Error)
	link := SourceItem{SourceID: src.ID, CMSMediaID: &item.ID, ExternalID: item.NormalizedKey, ExternalTitle: item.Title, State: "active", RawMetadata: `{"secret":"raw"}`, DetailURL: "https://upstream.example/detail"}
	require.NoError(t, f.db.Create(&link).Error)
	line := PlayLine{SourceItemID: link.ID, LineName: "m3u8", LineKey: "m3u8", Enabled: true}
	require.NoError(t, f.db.Create(&line).Error)
	for i := 1; i <= n; i++ {
		require.NoError(t, f.db.Create(&PlayEpisode{PlayLineID: line.ID, EpisodeNo: i, StreamURL: "https://cdn.example/" + item.NormalizedKey + "/index.m3u8", StreamType: "m3u8"}).Error)
	}
}

func newPublicFixture(t *testing.T) *publicFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "public.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Source{}, &Item{}, &SourceItem{}, &PlayLine{}, &PlayEpisode{}))
	svc, err := NewService(Config{DB: db})
	require.NoError(t, err)
	f := &publicFixture{svc: svc, db: db}
	source := func(name string, enabled, suspended bool) Source {
		s := Source{Name: name, APIURL: "https://" + name + ".example/api.php/provide/vod/", APIType: "json", Enabled: true}
		require.NoError(t, db.Create(&s).Error)
		require.NoError(t, db.Model(&s).Updates(map[string]any{"enabled": enabled, "suspended": suspended}).Error)
		return s
	}
	f.live, f.off, f.paused = source("live", true, false), source("off", false, false), source("paused", true, true)
	now := time.Now()
	f.movie = Item{Title: "流浪地球", MediaType: "movie", Year: "2023", Area: "中国", Genres: "科幻,冒险", Director: "郭帆", NormalizedKey: "movie", Status: "active", UpdatedAt: now.Add(-time.Hour)}
	f.series = Item{Title: "漫长的季节", MediaType: "series", Year: "2023", Area: "中国", Genres: "悬疑,剧情", Actors: "范伟,秦昊", NormalizedKey: "series", Status: "active", UpdatedAt: now}
	f.unknownYear = Item{Title: "无年份动画", MediaType: "anime", Year: "", Area: "日本", Genres: "冒险", NormalizedKey: "anime", Status: "pending_merge", UpdatedAt: now.Add(-2 * time.Hour)}
	f.hidden = Item{Title: "停用站点独有", MediaType: "movie", Year: "2020", NormalizedKey: "hidden", Status: "active"}
	f.offerless = Item{Title: "暂停站点独有", MediaType: "movie", Year: "2021", NormalizedKey: "paused", Status: "active"}
	f.seed(t, &f.movie, f.live, 1)
	f.seed(t, &f.series, f.live, 16)
	f.seed(t, &f.unknownYear, f.live, 3)
	f.seed(t, &f.hidden, f.off, 2)
	f.seed(t, &f.offerless, f.paused, 2)
	return f
}

func titles(t *testing.T, page map[string]any) []string {
	t.Helper()
	out := []string{}
	for _, card := range page["list"].([]PublicCard) {
		out = append(out, card.Title)
	}
	return out
}

func TestPublicListShowsOnlyPlayableTitles(t *testing.T) {
	f := newPublicFixture(t)
	ctx := context.Background()
	page, err := f.svc.ListPublic(ctx, PublicFilter{})
	require.NoError(t, err)
	require.EqualValues(t, 3, page["total"])
	require.Equal(t, []string{"漫长的季节", "流浪地球", "无年份动画"}, titles(t, page), "recently updated first")
	episodes := map[string]int{}
	for _, card := range page["list"].([]PublicCard) {
		episodes[card.Title] = card.Episodes
	}
	require.Equal(t, map[string]int{"漫长的季节": 16, "流浪地球": 1, "无年份动画": 3}, episodes)
}

func TestPublicListFilters(t *testing.T) {
	f := newPublicFixture(t)
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		filter PublicFilter
		want   []string
	}{
		{"type", PublicFilter{Type: "series"}, []string{"漫长的季节"}},
		{"year", PublicFilter{Years: []string{"2023"}}, []string{"漫长的季节", "流浪地球"}},
		{"unknown year", PublicFilter{Years: []string{"none"}}, []string{"无年份动画"}},
		{"year or unknown", PublicFilter{Years: []string{"2023", "none", "bogus"}}, []string{"漫长的季节", "流浪地球", "无年份动画"}},
		{"area", PublicFilter{Area: "日本"}, []string{"无年份动画"}},
		{"genre in a list", PublicFilter{Genre: "冒险"}, []string{"流浪地球", "无年份动画"}},
		{"genre is not a substring match", PublicFilter{Genre: "险"}, []string{}},
		{"query matches people", PublicFilter{Query: "秦昊"}, []string{"漫长的季节"}},
		{"wildcards are literal", PublicFilter{Query: "%"}, []string{"漫长的季节", "流浪地球", "无年份动画"}},
		{"hidden sources stay hidden", PublicFilter{Query: "独有"}, []string{}},
		{"sort by year", PublicFilter{Sort: "year"}, []string{"漫长的季节", "流浪地球", "无年份动画"}},
		{"paged", PublicFilter{Page: 2, PageSize: 2}, []string{"无年份动画"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, err := f.svc.ListPublic(ctx, tc.filter)
			require.NoError(t, err)
			require.Equal(t, tc.want, titles(t, page))
		})
	}
}

func TestPublicDetailExposesNoBackOfficeFields(t *testing.T) {
	f := newPublicFixture(t)
	detail, err := f.svc.PublicDetail(context.Background(), f.series.ID)
	require.NoError(t, err)
	require.Len(t, detail.Lines, 1)
	require.Len(t, detail.Lines[0].Episodes, 16)
	require.Equal(t, 16, detail.Episodes)
	body, err := json.Marshal(detail)
	require.NoError(t, err)
	for _, secret := range []string{"api.php", "upstream.example", "secret", "live.example"} {
		require.NotContains(t, string(body), secret)
	}
	hidden, err := f.svc.PublicDetail(context.Background(), f.hidden.ID)
	require.NoError(t, err)
	require.Empty(t, hidden.Lines, "a disabled source offers no lines")
}

func TestPublicFacetsCountPlayableCatalogue(t *testing.T) {
	f := newPublicFixture(t)
	facets, err := f.svc.PublicFacets(context.Background())
	require.NoError(t, err)
	require.Equal(t, []PublicFacet{{"2023", 2}, {"none", 1}}, facets.Years)
	require.Equal(t, []PublicFacet{{"中国", 2}, {"日本", 1}}, facets.Areas)
	require.Equal(t, PublicFacet{"冒险", 2}, facets.Genres[0])
	require.Len(t, facets.Genres, 4)
}

func TestPublicHomeRows(t *testing.T) {
	f := newPublicFixture(t)
	rows, err := f.svc.PublicHome(context.Background())
	require.NoError(t, err)
	keys := []string{}
	for _, row := range rows {
		keys = append(keys, row.Key)
	}
	require.Equal(t, []string{"latest", "movie", "series", "anime"}, keys, "empty rows are left out")
}

func TestPublicStreamOnlyServesListedEpisodes(t *testing.T) {
	f := newPublicFixture(t)
	ctx := context.Background()
	episodeOf := func(item Item) uint {
		var ep PlayEpisode
		require.NoError(t, f.db.Joins("JOIN media_play_lines l ON l.id = media_play_episodes.play_line_id").
			Joins("JOIN media_source_items si ON si.id = l.source_item_id").Where("si.cms_media_id = ?", item.ID).First(&ep).Error)
		return ep.ID
	}
	require.NoError(t, f.svc.publicEpisode(ctx, episodeOf(f.movie)))
	require.ErrorIs(t, f.svc.publicEpisode(ctx, episodeOf(f.hidden)), errEpisodeUnavailable)
	require.ErrorIs(t, f.svc.publicEpisode(ctx, episodeOf(f.offerless)), errEpisodeUnavailable)
	require.ErrorIs(t, f.svc.publicEpisode(ctx, 99999), errEpisodeUnavailable)
}
