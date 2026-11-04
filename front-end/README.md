# 影视 CMS 前端

Umi Max 4 + React 19 + Ant Design 5 / Pro Components。构建环境使用 Node 24.21.0 LTS，见 `.nvmrc`。

```bash
npm ci --legacy-peer-deps --ignore-scripts
npm run postinstall
npm run start:dev
```

开发 API 代理见 `config/config.ts`。Docker 部署入口在仓库根目录的 `deploy/local`，共享页面端口为 8080。

业务菜单、路由路径、图标、排序和权限由后端返回；`src/routing/componentMap.tsx` 声明允许加载的组件，`src/app.tsx` 将菜单转换为动态路由。不要把系统业务菜单重新硬编码进 `config/routes.ts`。

```bash
npm run tsc
npm test -- --runInBand
npm run build
```

需要修改数据的端到端测试，通过仓库根目录 `python3 deploy/local/e2e.py` 在独立容器和数据库中执行。

依赖版本选择、定向 overrides 及剩余上游审计问题见 [架构评审](../docs/BASE_FRAME_REVIEW.md)。本项目没有使用模板附带的地图/图表库、Petstore 客户端或外部样例 OpenAPI 配置。
