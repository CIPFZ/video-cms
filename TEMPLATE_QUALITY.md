# 影视 CMS 质量检查

版本、架构结论、已知限制见 [架构评审](docs/BASE_FRAME_REVIEW.md)。CI 配置为 [.github/workflows/ci.yml](.github/workflows/ci.yml)。新增 workflow 已在仓库中提供，远程 Actions 的执行结果需以 GitHub 为准。

## 本地检查

后端使用 Go 1.27.1：

```bash
cd backend
go test -p 2 ./...
go vet ./...
# 需要 C 编译器，用于 Go race detector。
go test -race -p 2 ./internal/core/audit ./internal/middleware ./internal/migrations ./internal/core/token
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

前端使用 `.nvmrc` 指定的 Node 24.21.0：

```bash
cd front-end
npm ci --legacy-peer-deps --ignore-scripts
npm run postinstall
npm run check:i18n
npm run tsc
npm test -- --runInBand
npm run build
npm audit
```

`--legacy-peer-deps` 是当前 Umi/Pro 依赖组合的安装策略；版本由 lockfile 和定向 overrides 约束。npm audit 包含上游构建依赖的已知问题，具体范围见评审记录，不将“命令退出 0”伪装为全部依赖安全。

## Docker 与隔离端到端测试

```bash
python3 deploy/local/init.py
docker compose --env-file deploy/local/.env -f deploy/local/compose.yaml build
# 在随机 Compose 项目、独立数据库/Redis/上传卷及随机端口上执行。
python3 deploy/local/e2e.py
# 通过后再更新共享部署。
docker compose --env-file deploy/local/.env -f deploy/local/compose.yaml up -d --no-build --wait
python3 deploy/local/smoke.py
```

不要直接在共享 8080 实例上执行会创建用户/通知的 Playwright 用例。`e2e.py` 只清理它自身创建的项目与卷。初次运行需在 front-end 中执行 `npx playwright install chromium`。

`/health` 检查进程存活；`/ready` 检查配置启用的数据库依赖。数据库升级前备份，恢复测试使用复制卷。已有账号、密码、菜单配置和上传卷应在常规应用升级中保留。

## 本轮新增回归范围

- 大请求/响应不被审计中间件改变，审计内容有界且敏感 JSON 字段脱敏。
- 历史日志迁移可重复执行；记录及非敏感字段保留。
- 审计队列并发 Push/Close、重复关闭和关闭后写入。
- SQL/API 分页规范与 offset 溢出防护；API 更新同步旧路径的权限规则。
- JWT 算法、issuer 和 expiry 约束；前端响应令牌续期格式兼容。
- 就绪检查失败返回 503 且不暴露连接错误。
- OTLP HTTP 三种信号可导出，W3C trace 延续，Zap 浮点字段与 trace ID 正确。
- 动态菜单决定页面路径；展开、折叠悬浮、嵌套菜单图标及排序；悬浮图标与文字垂直中心差小于 2px。


第二轮修复新增：会话/刷新家族撤销、改密/禁用/删除/角色变更失效、管理员保护；双权限管理实例即时撤权、失败事务回滚；多数据库并发迁移及重复执行保留编辑；非法角色/API/菜单和菜单循环校验；共享 Redis token 租约；注册开关 UI/API 一致；状态页概览。执行证据与部署结果见 [闭环记录](docs/REMEDIATION.md)。

```bash
cd backend && go run ./cmd/openapi
# 回到仓库根目录后
python3 scripts/check-api-contract.py
python3 deploy/local/db-matrix.py
python3 deploy/observability/verify.py
```

中英文适配约定、动态菜单英文名称与验证范围见 [国际化说明](docs/I18N.md)。

API Token 与通知回归约定见 [Token 设计](docs/apitoken-design.md) 和 [通知投递](docs/notices-design.md)。
新增用例覆盖实际开放路由、凭据生命周期、并发配额、重置与撤权交错、普通角色越权、角色收件人快照、有效期、登录弹窗以及定时发布。
浏览器批量登录触发真实 429 时按 Retry-After 等待，测试环境不关闭登录限流。
