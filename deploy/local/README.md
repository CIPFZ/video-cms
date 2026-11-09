# 本机 Docker 部署

此配置独立运行前端、后端、MySQL 和 Redis，Compose 项目名为 `media-edition`，默认与综合版的容器和持久卷分开。
文件存储使用本地持久卷，默认 Web 端口为 8080，后端健康接口仅映射到本机 18081。

从仓库根目录执行：

```bash
python3 deploy/local/init.py
DOCKER_BUILDKIT=0 docker compose --env-file deploy/local/.env -f deploy/local/compose.yaml up -d --build --wait --wait-timeout 180
python3 deploy/local/smoke.py
```

需要 Docker Engine、Docker Compose v2 和 Python 3；Go 与前端构建在容器中完成。
`DOCKER_BUILDKIT=0` 兼容本机没有 Buildx 插件的环境；已安装 Buildx 时可去掉这个变量。
无法访问 proxy.golang.org 或 npm 官方源时，在 `.env` 中追加构建镜像源，例如 `GOPROXY=https://goproxy.cn,direct` 和 `NPM_REGISTRY=https://registry.npmmirror.com/`、`ALPINE_MIRROR=https://mirrors.aliyun.com/alpine`；不设置时使用官方源。
后端构建包含 Go 测试，前端使用已锁定依赖执行 `npm ci` 和生产构建。

访问 `http://localhost:8080`；从其他机器访问时，将 localhost 替换为此开发机地址。
管理员账号为 `admin`，生成的密码保存在 `deploy/local/runtime/admin.txt`。
初始化脚本重复执行时保留 `.env` 中的密钥，不重置数据库中的已有账号密码。
`admin.txt` 保存初次初始化密码；如果通过后台修改密码，后续应使用修改后的密码。
端口和绑定地址在 `deploy/local/.env` 中配置。

```bash
# 状态与日志
docker compose --env-file deploy/local/.env -f deploy/local/compose.yaml ps
docker compose --env-file deploy/local/.env -f deploy/local/compose.yaml logs -f --tail=100 backend frontend

# 停止并保留数据
docker compose --env-file deploy/local/.env -f deploy/local/compose.yaml down

# 恢复运行
docker compose --env-file deploy/local/.env -f deploy/local/compose.yaml up -d --wait
```

持久卷：`media-edition_mysql-data`、`media-edition_redis-data`、`media-edition_uploads`。
Compose 的普通 down 保留数据；`down -v` 会删除这些数据，因此不用于日常停止。
后端等待数据库健康后执行迁移；迁移失败会退出，成功后再启动服务。
上传文件由前端 Nginx 通过共享卷提供访问。

`smoke.py` 检查管理员登录、影视菜单、片库与采集接口、已移除的业务路由、系统信息及图片上传下载。
完整浏览器验证使用 `python3 deploy/local/e2e.py`。它创建独立的临时 Compose 项目、数据库和持久卷，使用随机本机端口；退出时仅删除该测试项目。测试产物保留在私有的 `runtime/nexus-cms-e2e-*/`。
直接运行 Playwright 时默认跳过创建用户的通知测试；仅隔离测试入口设置 `E2E_ISOLATED=1`。

`.env`、运行配置、管理员密码和验证产物已加入 Git 忽略规则。
该配置用于开发机部署验证，后续 Bench 代码与日志产物应使用带任务权限的独立存储接口。

## 基础 CMS 数据迁移

迁移 `20260910_minimal_cms` 清除原插件、诗词模块的菜单、API、授权、专用角色、相关操作日志和 12 张业务表。
已有用户、密码和系统配置保留；使用被移除角色的用户转为普通 CMS 角色。
迁移成功后记录在 `sys_schema_migrations`，重复启动不会重新执行；初始化也不再生成这两个模块的数据。
MySQL 的表删除隐式提交，迁移按可重试步骤执行，完成标记最后写入。升级前保留私有数据库备份。

菜单和业务路由来自后端 `getMenu`，更新部署后刷新浏览器即可获取新菜单。

迁移 `20260910_cms_names_and_test_accounts` 把旧菜单名称中的翻译 Key 转为实际展示名称，保留 `locale` 和管理员自定义名称；清理旧测试脚本生成的 `e2e_user_<数字>`、`e2e_user_b_<数字>`、`smoke_user_<数字>` 账号及测试通知。
该迁移只执行一次，不删除真实团队账号，也不限制以后新建用户。全新数据库仅由管理员初始化函数创建一个管理员。

迁移 `20260910_cms_menu_order` 调整基础菜单顺序，和创建菜单时共用 `seed.MenuOrder`：

- 一级菜单：工作台（10）、系统管理（20）、系统状态（90）、关于（100）；个人设置保持隐藏（999）。
- 系统管理：用户管理（10）、角色管理（20）、菜单管理（30）、API 管理（40）、API Token（50）、通知公告（60）、操作日志（70）。

数字越小越靠前，相同排序值按菜单 ID 稳定排列。升级后在菜单管理中调整的排序与图标会保留，重启不会重置。
左侧导航通过布局组件的渲染接口展示所有层级的后端图标，路由、权限和菜单数据仍由后端下发。


## 独立迁移、备份与恢复

启动顺序是 MySQL → 一次性 migrator → backend → frontend。migrator 使用同连接数据库锁，完成版本迁移后退出；HTTP 只校验 schema，不执行 DDL 或补回授权。已有账号、密码、菜单和撤销的授权不会被 seed 覆盖。升级到服务端会话后旧 JWT 会失效，需要重新登录。

```bash
python3 deploy/local/backup.py create --retain 14
python3 deploy/local/backup.py verify deploy/local/runtime/backups/<时间戳>
python3 deploy/local/backup.py rehearse deploy/local/runtime/backups/<时间戳> --mysql-image mysql:8.4
python3 deploy/local/db-matrix.py
```

备份会短暂停止后端写入，保存 SQL、上传文件、配置和 SHA-256 manifest。成功后恢复之前运行的后端；失败不清理旧备份。备份包含密钥，只存于权限受限的 runtime 目录，不可提交 Git。保留最近 14 份成功备份。恢复演练始终使用随机项目和新卷，核对文件哈希、迁移、就绪状态、登录和动态菜单，结束只删除自身资源。

`systemd/nexus-cms-backup.{service,timer}` 提供每天北京时间 03:15（最多延后 5 分钟）的计划，部署路径改变时先修改 service。安装后用 `systemctl list-timers nexus-cms-backup.timer` 查看时间，`journalctl -u nexus-cms-backup.service` 查看结果。当前主机只保留本地副本；要覆盖整机磁盘损坏，需要再配置异机备份目标。

可观测部署、数据范围和 8080 入口见 [观测说明](../observability/README.md)。配置静态加载，变更 JWT、注册开关或代理信任范围后重启。Nginx 覆盖转发 IP 头，后端仅信任部署指定的代理网段；其他环境应缩小为实际代理地址。
