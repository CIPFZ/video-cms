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

部署后无需配置：新库内置 12 个资源站（暴风、无尽、豪华、非凡、极速、魔都、天涯、牛牛、红牛、速播、光速、鸭鸭），站点已启用，增量采集计划默认开启，迁移完成后立即开始第一轮采集，前台随即出现影片。

1. 想一次补齐片库：登录后在 **影视资源 → 采集计划** 对需要的站点点全量采集的“立即执行”，进度在 **采集日志** 查看，可随时取消并从断点继续。
2. 增量每 3 小时更新一次；全量计划默认每月 1 日校准，需要时在采集计划中启用。
3. 内置站点统一排除伦理、三级、写真、擦边等分类（见 `backend/internal/modules/media/catalog.go`）。不想要的站点在 **资源站点** 中停用；也可以添加自己的 HTTPS JSON 资源接口，自定义站点先做连通性检查和试采集，再在采集计划中启用。
4. 多个站点有同一部影片时合并为一条，播放页按站点优先级把各站线路列为“线路 1、2…”，数字越小越靠前。

内置站点名单取自资源组（ziyuanzu.com）收录的苹果 CMS 接口，2026-11-06 逐个核验：HTTPS 可用、当日有更新、浏览器能直接播放 m3u8、以常规影视分类为主；以成人内容为主的站点，以及在视频开头插入不可访问广告分片导致卡住的站点未收录。资源站地址会变动，失效的站点会因连续健康检查失败而自动暂停。已部署的实例升级时只补充缺少的站点，管理员停用或修改过的站点保持不变。

前台地址：首页 `/#/`、片库 `/#/browse`（筛选条件写在网址里）、详情 `/#/title/<id>`、播放 `/#/watch/<id>`。前台只读 `/api/v1/public/media/*`，无需登录。播放默认由浏览器直接请求资源站 CDN，首帧前失败才改走本站的 HLS 转发。新库不会复制现有片库、自定义站点或服务器凭据。

## 部署与维护

- [Docker 部署、备份与恢复](deploy/local/README.md)
- [本地开发说明](LOCAL_RUN.md)
- [CMS 基础架构](docs/DIRECTORY_STRUCTURE.md)

后端影视模块在 `backend/internal/modules/media`，前端页面在 `front-end/src/pages/media`。菜单和权限由独立迁移程序初始化，后端服务启动后执行采集调度。
