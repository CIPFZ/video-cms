package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type rawRecord map[string]any
type upstreamPage struct {
	Records            []rawRecord
	Page, Pages, Total int
}

func validateURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || len(raw) > 2000 {
		return errors.New("需要无凭据的 HTTPS URL")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !publicIP(ip) {
		return errors.New("禁止访问内部网络")
	}
	return nil
}
func publicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified() &&
		!(ip.To4() != nil && ip.To4()[0] == 100 && ip.To4()[1] >= 64 && ip.To4()[1] <= 127)
}
func safeClient() *http.Client {
	tr := &http.Transport{TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 20 * time.Second, MaxIdleConns: 10, IdleConnTimeout: 30 * time.Second}
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, errors.New("来源域名解析失败")
		}
		if len(ips) == 0 {
			return nil, errors.New("来源域名无地址")
		}
		for _, ip := range ips {
			if !publicIP(ip.IP) {
				return nil, errors.New("禁止访问内部网络")
			}
		}
		d := net.Dialer{Timeout: 10 * time.Second}
		var last error
		for _, ip := range ips {
			conn, e := d.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if e == nil {
				return conn, nil
			}
			last = e
		}
		return nil, last
	}
	return &http.Client{Transport: tr, Timeout: 15 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("重定向过多")
		}
		return validateURL(req.URL.String())
	}}
}
func (s *Service) request(ctx context.Context, source Source, params url.Values) ([]byte, int, error) {
	if err := validateURL(source.APIURL); err != nil {
		return nil, 0, err
	}
	u, _ := url.Parse(source.APIURL)
	q := u.Query()
	for k, v := range params {
		q[k] = v
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "VideoCMS-MediaCollector/1.0")
	client := *s.client
	client.Timeout = time.Duration(source.TimeoutSeconds) * time.Second
	if client.Timeout <= 0 {
		client.Timeout = 15 * time.Second
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, errors.New("来源请求失败或超时")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, resp.StatusCode, fmt.Errorf("来源 HTTP %d", resp.StatusCode)
	}
	const max = 8 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, resp.StatusCode, errors.New("读取来源响应失败")
	}
	if len(body) > max {
		return nil, resp.StatusCode, errors.New("来源响应超过 8 MiB")
	}
	return body, resp.StatusCode, nil
}
func decodePage(body []byte) (upstreamPage, error) {
	var root map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return upstreamPage{}, errors.New("来源不是有效 JSON")
	}
	if code, ok := root["code"]; ok && asString(code) != "1" {
		return upstreamPage{}, errors.New("来源返回业务错误")
	}
	list, ok := root["list"].([]any)
	if !ok {
		return upstreamPage{}, errors.New("来源缺少 list 数组")
	}
	out := upstreamPage{Page: intValue(root["page"]), Pages: intValue(root["pagecount"]), Total: intValue(root["total"]), Records: []rawRecord{}}
	for _, value := range list {
		row, ok := value.(map[string]any)
		if !ok || firstString(row, "vod_id") == "" {
			return out, errors.New("来源条目缺少稳定 ID")
		}
		out.Records = append(out.Records, rawRecord(row))
	}
	if out.Pages < 0 || out.Pages > 1000000 {
		return out, errors.New("来源分页无效")
	}
	return out, nil
}
func decodeRecords(body []byte, apiType string) ([]rawRecord, error) {
	if apiType != "json" {
		return nil, errors.New("只支持 JSON")
	}
	p, e := decodePage(body)
	return p.Records, e
}
func (s *Service) fetchPage(ctx context.Context, src Source, page, hours int) (upstreamPage, error) {
	params := url.Values{"ac": {"list"}, "pg": {strconv.Itoa(page)}, "pagesize": {"20"}}
	if hours > 0 {
		params.Set("h", strconv.Itoa(hours))
	}
	body, _, err := s.request(ctx, src, params)
	if err != nil {
		return upstreamPage{}, err
	}
	return decodePage(body)
}
func (s *Service) fetchDetails(ctx context.Context, src Source, ids []string) ([]rawRecord, error) {
	body, _, err := s.request(ctx, src, url.Values{"ac": {"detail"}, "ids": {strings.Join(ids, ",")}})
	if err != nil {
		return nil, err
	}
	return decodeRecords(body, "json")
}
func hashRecord(r rawRecord) string {
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func firstString(r map[string]any, keys ...string) string {
	for _, k := range keys {
		v := strings.TrimSpace(asString(r[k]))
		if v != "" {
			return v
		}
	}
	return ""
}
func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	}
	return ""
}
func intValue(v any) int { n, _ := strconv.Atoi(asString(v)); return n }
func pause(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (s *Service) HealthCheck(ctx context.Context, id uint) (HealthCheck, error) {
	var check HealthCheck
	err := s.withSource(ctx, id, func(ctx context.Context, src Source) error {
		start := time.Now()
		check = HealthCheck{SourceID: id, CheckedAt: start, Result: "failed"}
		body, status, e := s.request(ctx, src, url.Values{"ac": {"list"}, "pg": {"1"}, "pagesize": {"1"}})
		check.HTTPStatus = status
		check.ResponseBytes = len(body)
		check.LatencyMS = int(time.Since(start).Milliseconds())
		if e == nil {
			_, e = decodePage(body)
		}
		updates := map[string]any{}
		if e == nil {
			check.Result = "ok"
			updates["successes"] = src.Successes + 1
			updates["failures"] = 0
			updates["last_ok_at"] = start
			updates["last_error"] = ""
			if src.Successes+1 >= 3 {
				updates["suspended"] = false
			}
		} else {
			check.ErrorMessage = e.Error()
			updates["successes"] = 0
			updates["failures"] = src.Failures + 1
			updates["last_error_at"] = start
			updates["last_error"] = e.Error()
			if src.Failures+1 >= 3 {
				updates["suspended"] = true
			}
		}
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if e := tx.Create(&check).Error; e != nil {
				return e
			}
			return tx.Model(&Source{}).Where("id = ?", id).Updates(updates).Error
		})
	})
	return check, err
}
