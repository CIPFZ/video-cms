package media

import (
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/gin-gonic/gin"
	"strconv"
)

type API struct{ service *Service }

func NewAPI(s *Service) *API { return &API{service: s} }

type sourceRequest struct {
	Name                   string `json:"name" binding:"required"`
	APIURL                 string `json:"apiUrl" binding:"required"`
	APIType                string `json:"apiType"`
	Enabled                bool   `json:"enabled"`
	Priority               int    `json:"priority"`
	AllowCategories        string `json:"allowCategories"`
	DenyCategories         string `json:"denyCategories"`
	RequestIntervalSeconds int    `json:"requestIntervalSeconds"`
	TimeoutSeconds         int    `json:"timeoutSeconds"`
	FullEnabled            bool   `json:"fullEnabled"`
}

func sourceFromRequest(r sourceRequest) Source {
	return Source{Name: r.Name, APIURL: r.APIURL, APIType: r.APIType, Enabled: r.Enabled, Priority: r.Priority, AllowCategories: r.AllowCategories, DenyCategories: r.DenyCategories, RequestIntervalSeconds: r.RequestIntervalSeconds, TimeoutSeconds: r.TimeoutSeconds, FullEnabled: r.FullEnabled}
}
func (a *API) ListSources(c *gin.Context) {
	v, e := a.service.ListSources(c.Request.Context())
	if e != nil {
		response.FailWithMessage("获取资源站失败", c)
		return
	}
	response.OkWithData(v, c)
}
func (a *API) CreateSource(c *gin.Context) {
	var r sourceRequest
	if e := c.ShouldBindJSON(&r); e != nil {
		response.FailWithValidation(e, c)
		return
	}
	v, e := a.service.CreateSource(c.Request.Context(), sourceFromRequest(r))
	if e != nil {
		response.FailWithMessage("创建资源站失败: "+e.Error(), c)
		return
	}
	response.OkWithData(v, c)
}
func (a *API) UpdateSource(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		response.FailWithMessage("来源 ID 无效", c)
		return
	}
	var r sourceRequest
	if e := c.ShouldBindJSON(&r); e != nil {
		response.FailWithValidation(e, c)
		return
	}
	v := sourceFromRequest(r)
	v.ID = id
	v, e := a.service.UpdateSource(c.Request.Context(), v)
	if e != nil {
		response.FailWithMessage("更新资源站失败: "+e.Error(), c)
		return
	}
	response.OkWithData(v, c)
}
func (a *API) Health(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		response.FailWithMessage("来源 ID 无效", c)
		return
	}
	v, e := a.service.HealthCheck(c.Request.Context(), id)
	if e != nil {
		response.FailWithMessage("健康检查失败: "+e.Error(), c)
		return
	}
	response.OkWithData(v, c)
}
func (a *API) Enqueue(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		response.FailWithMessage("来源 ID 无效", c)
		return
	}
	var req CollectOptions
	_ = c.ShouldBindJSON(&req)
	if req.Mode == "" {
		req.Mode = c.DefaultQuery("mode", "sample")
	}
	if v, e := a.service.Enqueue(c.Request.Context(), id, req); e != nil {
		response.FailWithMessage("任务排队失败: "+e.Error(), c)
	} else {
		response.OkWithData(v, c)
	}
}
func (a *API) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	sourceID, _ := strconv.ParseUint(c.Query("sourceId"), 10, 64)
	v, e := a.service.ListRuns(c.Request.Context(), RunFilter{SourceID: uint(sourceID), Mode: c.Query("mode"), Status: c.Query("status"), Page: page, PageSize: size})
	if e != nil {
		response.FailWithMessage("获取任务失败", c)
		return
	}
	response.OkWithData(v, c)
}
func (a *API) ListRunItems(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		response.FailWithMessage("任务 ID 无效", c)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	v, e := a.service.ListRunItems(c.Request.Context(), id, c.Query("action"), page, size)
	if e != nil {
		response.FailWithMessage("获取采集明细失败", c)
		return
	}
	response.OkWithData(v, c)
}
func (a *API) CancelRun(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		response.FailWithMessage("任务 ID 无效", c)
		return
	}
	if e := a.service.CancelRun(c.Request.Context(), id); e != nil {
		response.FailWithMessage("取消任务失败: "+e.Error(), c)
		return
	}
	response.Ok(c)
}
func (a *API) ListSchedules(c *gin.Context) {
	v, e := a.service.ListSchedules(c.Request.Context())
	if e != nil {
		response.FailWithMessage("获取调度失败", c)
		return
	}
	response.OkWithData(v, c)
}

type scheduleRequest struct {
	Cron    string `json:"cron"`
	Enabled bool   `json:"enabled"`
}

func (a *API) UpdateSchedule(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		response.FailWithMessage("调度 ID 无效", c)
		return
	}
	var r scheduleRequest
	if e := c.ShouldBindJSON(&r); e != nil {
		response.FailWithValidation(e, c)
		return
	}
	if e := a.service.UpdateSchedule(c.Request.Context(), id, r.Cron, r.Enabled); e != nil {
		response.FailWithMessage("更新调度失败: "+e.Error(), c)
		return
	}
	response.Ok(c)
}
func (a *API) ListHealth(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		response.FailWithMessage("来源 ID 无效", c)
		return
	}
	v, e := a.service.ListHealth(c.Request.Context(), id)
	if e != nil {
		response.FailWithMessage("获取健康记录失败", c)
		return
	}
	response.OkWithData(v, c)
}
func (a *API) ListItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	v, e := a.service.ListItems(c.Request.Context(), page, size, c.Query("query"), c.Query("mediaType"))
	if e != nil {
		response.FailWithMessage("获取影视条目失败", c)
		return
	}
	response.OkWithData(v, c)
}
func (a *API) GetItem(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		response.FailWithMessage("影视 ID 无效", c)
		return
	}
	v, e := a.service.GetItemDetail(c.Request.Context(), id)
	if e != nil {
		response.FailWithMessage("影视条目不存在", c)
		return
	}
	response.OkWithData(v, c)
}
func parseUintID(v string) (uint, bool) {
	n, e := strconv.ParseUint(v, 10, 32)
	return uint(n), e == nil && n > 0
}

func (a *API) RefreshItem(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		response.FailWithMessage("影视 ID 无效", c)
		return
	}
	if e := a.service.RefreshItem(c.Request.Context(), id); e != nil {
		response.FailWithMessage(e.Error(), c)
		return
	}
	response.Ok(c)
}
func (a *API) CheckEpisode(c *gin.Context) {
	id, ok := parseUintID(c.Param("id"))
	if !ok {
		response.FailWithMessage("剧集 ID 无效", c)
		return
	}
	v, e := a.service.CheckEpisode(c.Request.Context(), id)
	if e != nil {
		response.FailWithMessage(e.Error(), c)
		return
	}
	response.OkWithData(v, c)
}
