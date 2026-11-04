package media

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var hlsURIAttribute = regexp.MustCompile(`URI="([^"]+)"`)

func (a *API) StreamEpisode(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		c.Status(http.StatusBadRequest)
		return
	}
	var episode PlayEpisode
	if err := a.service.db.WithContext(c.Request.Context()).First(&episode, id).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	rootURL, err := url.Parse(episode.StreamURL)
	if err != nil || !validPlaybackURL(rootURL) {
		c.Status(http.StatusBadGateway)
		return
	}
	targetURL := rootURL
	if raw := strings.TrimSpace(c.Query("target")); raw != "" {
		targetURL, err = url.Parse(raw)
		if err != nil || !validPlaybackURL(targetURL) || !allowedPlaybackTarget(rootURL, targetURL) {
			c.Status(http.StatusForbidden)
			return
		}
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, targetURL.String(), nil)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	req.Header.Set("User-Agent", "VideoCMS-MediaPlayer/1.0")
	req.Header.Set("Referer", rootURL.String())
	if rangeHeader := c.GetHeader("Range"); rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	client := *a.service.client
	// Video responses may remain open much longer than collection API calls.
	client.Timeout = 2 * time.Hour
	resp, err := client.Do(req)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		c.Status(resp.StatusCode)
		return
	}

	c.Header("X-Accel-Buffering", "no")
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	if value := resp.Header.Get("Accept-Ranges"); value != "" {
		c.Header("Accept-Ranges", value)
	}
	if value := resp.Header.Get("Content-Range"); value != "" {
		c.Header("Content-Range", value)
	}

	contentType := resp.Header.Get("Content-Type")
	manifest, readErr := readManifest(resp, resp.Request.URL, contentType, c.Request.URL.Path)
	if readErr != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	if manifest != nil {
		c.Data(http.StatusOK, "application/vnd.apple.mpegurl", manifest)
		return
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	if resp.ContentLength >= 0 {
		c.Header("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
	}
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}

func readManifest(resp *http.Response, target *url.URL, contentType, proxyPath string) ([]byte, error) {
	isManifest := strings.Contains(strings.ToLower(contentType), "mpegurl") ||
		strings.Contains(strings.ToLower(contentType), "m3u8") ||
		strings.HasSuffix(strings.ToLower(target.Path), ".m3u8")
	if !isManifest {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > 2<<20 {
		return nil, errors.New("HLS manifest exceeds size limit")
	}
	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "#EXTM3U") {
		return nil, errors.New("upstream response is not an HLS manifest")
	}
	return []byte(rewriteManifest(trimmed, target, proxyPath)), nil
}

func rewriteManifest(manifest string, base *url.URL, proxyPath string) string {
	lines := strings.Split(manifest, "\n")
	for i, line := range lines {
		lines[i] = hlsURIAttribute.ReplaceAllStringFunc(line, func(match string) string {
			parts := hlsURIAttribute.FindStringSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			if absolute, err := resolvePlaybackURL(base, parts[1]); err == nil {
				return `URI="` + proxyPath + "?target=" + url.QueryEscape(absolute.String()) + `"`
			}
			return match
		})
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		uri := strings.TrimSpace(line)
		if uri == "" {
			continue
		}
		if absolute, err := resolvePlaybackURL(base, uri); err == nil {
			lines[i] = proxyPath + "?target=" + url.QueryEscape(absolute.String())
		}
	}
	return strings.Join(lines, "\n")
}

func resolvePlaybackURL(base *url.URL, reference string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(reference))
	if err != nil {
		return nil, err
	}
	resolved := base.ResolveReference(parsed)
	if !validPlaybackURL(resolved) || !allowedPlaybackTarget(base, resolved) {
		return nil, errors.New("playback target is not allowed")
	}
	return resolved, nil
}

func validPlaybackURL(value *url.URL) bool {
	return value != nil && (value.Scheme == "http" || value.Scheme == "https") && value.Hostname() != "" && value.User == nil
}

func allowedPlaybackTarget(root, target *url.URL) bool {
	rootHost := strings.ToLower(strings.TrimSuffix(root.Hostname(), "."))
	targetHost := strings.ToLower(strings.TrimSuffix(target.Hostname(), "."))
	return targetHost == rootHost || strings.HasSuffix(targetHost, "."+rootHost)
}
