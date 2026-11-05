# 影视 CMS

影视采集与观看系统，从 [go-web-frame](https://github.com/CIPFZ/go-web-frame) 的 `my-hk-server` 分支抽出，保留 CMS 的账号、角色和权限管理。影视模块包括资源站、连通性检查、采集计划、采集日志、影视库和独立播放页。部署时无需配置磁力预览或代理管理服务。

## 全新部署

需要 Git、Python 3、Docker Engine 和 Docker Compose v2。在新目录执行：

```bash
git clone https://github.com/CIPFZ/video-cms.git
cd video-cms
python3 deploy/local/init.py
docker compose --env-file deploy/local/.env -f deploy/local/compose.yaml up -d --build --wait --wait-timeout 180
python3 deploy/local/smoke.py
```

打开 `http://localhost:8080` 是免登录的观影前台；后台入口是前台右上角的“管理后台”（`/#/admin`），首次管理员账号和随机密码在本机 `deploy/local/runtime/admin.txt`。默认使用专版独立的 MySQL、Redis 和上传文件卷。请使用新库，不要把专版迁移指向正在使用的综合版数据库。同机部署时，如果 8080 或 18081 已被占用，先在生成的 `deploy/local/.env` 修改 `CMS_WEB_PORT` 或 `CMS_API_PORT`。

## 第一次采集与观看

1. 登录后打开 **影视资源 → 资源站点**，选择内置站点，或添加自己的 HTTPS JSON 资源接口并启用站点。
2. 先做连通性检查和小批量试采集，在 **采集日志** 看执行结果。
3. 打开 **影视库**，搜索条目、查看详情、选择剧集播放。
4. 确认试采集正常后，在 **采集计划** 启用所需的增量或全量任务。计划使用北京时间，保存后按页面显示的下次执行时间触发。

前台地址：首页 `/#/`、片库 `/#/browse`（筛选条件写在网址里）、详情 `/#/title/<id>`、播放 `/#/watch/<id>`。前台只读 `/api/v1/public/media/*`，无需登录。播放默认由浏览器直接请求资源站 CDN，首帧前失败才改走本站的 HLS 转发。新库会初始化三个内置站点；不会复制现有片库、自定义站点或服务器凭据。

## 部署与维护

- [Docker 部署、备份与恢复](deploy/local/README.md)
- [本地开发说明](LOCAL_RUN.md)
- [CMS 基础架构](docs/DIRECTORY_STRUCTURE.md)

后端影视模块在 `backend/internal/modules/media`，前端页面在 `front-end/src/pages/media`。菜单和权限由独立迁移程序初始化，后端服务启动后执行采集调度。
