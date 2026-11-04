package media

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// collectorFixture serves one catalogue page of three titles: the first saves,
// the second has no title and the third is missing from the detail response.
func collectorFixture(t *testing.T) (*Service, Source) {
	t.Helper()
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("ac") == "detail" {
			_, _ = w.Write([]byte(`{"code":1,"list":[
				{"vod_id":1,"vod_name":"测试电影","type_name":"电影","vod_year":"2025","vod_play_from":"m3u8","vod_play_url":"正片$https://cdn.example/a.m3u8"},
				{"vod_id":2,"vod_name":"","type_name":"电影","vod_year":"2025"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":1,"page":1,"pagecount":1,"list":[
			{"vod_id":1,"vod_name":"测试电影","type_name":"电影"},
			{"vod_id":2,"vod_name":"无名条目","type_name":"电影"},
			{"vod_id":3,"vod_name":"缺失详情","type_name":"电影"}]}`))
	}))
	t.Cleanup(upstream.Close)

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "media.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Source{}, &Item{}, &SourceItem{}, &PlayLine{}, &PlayEpisode{},
		&CollectionRun{}, &CollectionPageFailure{}, &CollectionRunItem{}, &HealthCheck{}, &CollectionSchedule{}))
	svc, err := NewService(Config{DB: db})
	require.NoError(t, err)
	// The source URL passes the public-HTTPS check; every connection reaches the fixture.
	client := upstream.Client()
	transport := client.Transport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // local test server
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, upstream.Listener.Addr().String())
	}
	client.Transport = transport
	svc.client = client

	src := Source{Name: "测试站", APIURL: "https://collector.example/api.php/provide/vod/", APIType: "json", Enabled: true, FullEnabled: true, TimeoutSeconds: 5}
	require.NoError(t, db.Create(&src).Error)
	require.NoError(t, db.Model(&src).Update("request_interval_seconds", 0).Error)
	return svc, src
}

func runOnce(t *testing.T, svc *Service, sourceID uint, mode string) CollectionRun {
	t.Helper()
	run, err := svc.Enqueue(context.Background(), sourceID, CollectOptions{Mode: mode})
	require.NoError(t, err)
	require.NoError(t, svc.db.Model(&run).Update("status", "running").Error)
	svc.execute(context.Background(), &run)
	require.NoError(t, svc.db.First(&run, run.ID).Error)
	return run
}

func TestCollectionRunRecordsSavedAndFailedTitles(t *testing.T) {
	svc, src := collectorFixture(t)
	run := runOnce(t, svc, src.ID, "sample")
	require.Equal(t, "partial", run.Status)
	require.Equal(t, 1, run.InsertedCount)
	require.Equal(t, 2, run.ErrorCount)

	all, err := svc.ListRunItems(context.Background(), run.ID, "", 1, 20)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"inserted": 1, "failed": 2}, all["summary"])
	failed, err := svc.ListRunItems(context.Background(), run.ID, "failed", 1, 20)
	require.NoError(t, err)
	rows := failed["list"].([]CollectionRunItem)
	require.Len(t, rows, 2)
	require.Equal(t, "无名条目", rows[0].Title)
	require.Contains(t, rows[0].ErrorMessage, "缺少标题")
	require.Equal(t, "缺失详情", rows[1].Title)
	require.Contains(t, rows[1].ErrorMessage, "缺少该条目")
	inserted, err := svc.ListRunItems(context.Background(), run.ID, "inserted", 1, 20)
	require.NoError(t, err)
	saved := inserted["list"].([]CollectionRunItem)[0]
	require.Equal(t, "测试电影", saved.Title)
	require.NotNil(t, saved.ItemID)
	require.NoError(t, svc.db.First(&Item{}, *saved.ItemID).Error)
}

func TestFullCollectionHandsOffToIncremental(t *testing.T) {
	svc, src := collectorFixture(t)
	run := runOnce(t, svc, src.ID, "full")
	require.Equal(t, 0, run.NextPage, "the traversal reached the last page")
	require.NoError(t, svc.db.First(&src, src.ID).Error)
	require.Nil(t, src.FullStartedAt)
	require.NotNil(t, src.LastIncrementalAt)
	require.False(t, src.LastIncrementalAt.Before(run.StartedAt))
}

func TestHealthChecksStayOutOfTheCollectionLog(t *testing.T) {
	svc, src := collectorFixture(t)
	status, msg := svc.scheduledHealthCheck(context.Background(), src.ID)
	require.Equal(t, "success", status, msg)
	var checks, runs int64
	require.NoError(t, svc.db.Model(&HealthCheck{}).Count(&checks).Error)
	require.NoError(t, svc.db.Model(&CollectionRun{}).Count(&runs).Error)
	require.EqualValues(t, 1, checks)
	require.Zero(t, runs)

	// Health runs queued by earlier versions are hidden from the log.
	require.NoError(t, svc.db.Create(&CollectionRun{SourceID: src.ID, Mode: "health", Status: "success"}).Error)
	sample := runOnce(t, svc, src.ID, "sample")
	for _, mode := range []string{"", "collect", "health"} {
		listed, err := svc.ListRuns(context.Background(), RunFilter{Mode: mode})
		require.NoError(t, err)
		rows := listed["list"].([]CollectionRun)
		if mode == "health" {
			require.Empty(t, rows)
			continue
		}
		require.Len(t, rows, 1)
		require.Equal(t, sample.ID, rows[0].ID)
	}

	_, err := svc.Enqueue(context.Background(), src.ID, CollectOptions{Mode: "health"})
	require.Error(t, err)
	require.NoError(t, svc.db.Model(&src).Update("enabled", false).Error)
	status, _ = svc.scheduledHealthCheck(context.Background(), src.ID)
	require.Equal(t, "skipped", status)
}
