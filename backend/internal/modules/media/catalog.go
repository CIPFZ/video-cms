package media

import "strings"

// BuiltinDenyCategories are the adult and borderline categories the built-in
// sites publish; titles in them are never collected.
const BuiltinDenyCategories = "伦理片,伦理,港台三级,韩国伦理,西方伦理,日本伦理,三级片,写真热舞,擦边短剧,里番动漫,福利,成人,色情,短视频,直播"

// LegacyDenyCategories was the deny list of the first three built-in sites.
const LegacyDenyCategories = "伦理片,伦理,成人,色情,短视频,直播"

// BuiltinSources are MacCMS JSON sites checked on 2026-11-06 (HTTPS, catalogue of
// 70k–170k titles updated that day, m3u8 playback in the browser, mainstream
// categories). Not listed: sites whose catalogue is adult content, and sites whose
// manifests start with an ad segment on an unreachable host (playback stalls).
// Lower priority wins metadata and comes first among a title's lines.
var BuiltinSources = []Source{
	{Name: "暴风资源", APIURL: "https://bfzyapi.com/api.php/provide/vod/", Priority: 10},
	{Name: "无尽资源", APIURL: "https://api.wujinapi.com/api.php/provide/vod/", Priority: 20},
	{Name: "豪华资源", APIURL: "https://hhzyapi.com/api.php/provide/vod/at/json", Priority: 30},
	{Name: "非凡资源", APIURL: "https://ffzy5.tv/api.php/provide/vod/", Priority: 40},
	{Name: "极速资源", APIURL: "https://jszyapi.com/api.php/provide/vod/", Priority: 50},
	{Name: "魔都资源", APIURL: "https://www.mdzyapi.com/api.php/provide/vod/", Priority: 60},
	{Name: "天涯影视资源", APIURL: "https://tyyszyapi.com/api.php/provide/vod/", Priority: 70},
	{Name: "牛牛资源", APIURL: "https://api.niuniuzy.me/api.php/provide/vod/", Priority: 80},
	{Name: "红牛资源", APIURL: "https://hongniuzy2.com/api.php/provide/vod/", Priority: 90},
	{Name: "速播资源", APIURL: "https://subocj.com/api.php/provide/vod/", Priority: 100},
	{Name: "光速资源", APIURL: "https://api.guangsuapi.com/api.php/provide/vod/", Priority: 110},
	{Name: "鸭鸭资源", APIURL: "https://cj.yayazy.net/api.php/provide/vod/", Priority: 120},
}

// NewBuiltinSource fills the settings shared by every built-in site. Full
// collection is allowed so it can be started from the schedule page at once.
func NewBuiltinSource(s Source) Source {
	s.APIType, s.Enabled, s.FullEnabled = "json", true, true
	s.RequestIntervalSeconds, s.TimeoutSeconds = 3, 15
	s.DenyCategories = BuiltinDenyCategories
	return s
}

// IsBuiltinSource reports whether an API address is one of the built-in sites.
func IsBuiltinSource(apiURL string) bool {
	for _, s := range BuiltinSources {
		if strings.EqualFold(s.APIURL, strings.TrimSpace(apiURL)) {
			return true
		}
	}
	return false
}
