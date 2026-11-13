# 中网联合 OA / CRM

zwOA 是面向公司内部使用的办公与客户管理平台。本项目为公开可见的闭源项目。

当前可运行登录、工作台、消息和独立管理面板。超级管理员可以创建账户、配置权限组、设置团队与项目成员，以及分别分配团队和模块管理员。客户管理和办公审批的业务流程待接入。

## 项目结构

| 路径 | 职责 |
| --- | --- |
| `cmd/api` | GoFrame API 进程入口 |
| `internal/api` | HTTP 路由、认证、跨域策略、请求解析和响应 |
| `internal/platform/httpserver` | GoFrame 服务配置和健康检查 |
| `internal/platform/sqlite` | SQLite 连接配置和按版本执行的迁移 |
| `internal/modules/workbench` | 工作台业务接口 |
| `internal/modules/chat` | 单聊、群聊、消息历史和已读位置 |
| `internal/modules/organization` | 账户、登录会话、权限组、团队与项目、模块管理员 |
| `apps/web/src/app` | Web 入口、应用路由、工作区外壳和全局主题 |
| `apps/web/src/features` | 登录与改密、组织管理、消息和工作台，各自维护界面、样式及接口 |
| `apps/web/src/shared` | 浏览器和 Windows 共用的 HTTP 客户端、顶栏和窗口控件 |
| `apps/desktop` | Tauri 2 桌面窗口和 Windows 构建配置 |

后端进程打开一份 SQLite 数据库，并把连接交给各业务模块。模块内部维护业务 SQL 和数据访问规则，数据库结构变更放在 `internal/platform/sqlite/migrations`。业务路由统一挂在 `/api/v1`。网页和 Windows 共用 React 界面与 HTTP 契约。

## 本地开发

需要 Go 1.26、Node.js 22.12 或更高版本及 npm。Go 版本由 `go.mod` 声明。

1. 在仓库根目录创建首个超级管理员，按提示输入并确认至少 12 个字符的密码。此命令只用于初始化空库。

   ```powershell
   go run ./cmd/api init-admin admin "系统管理员"
   ```

2. 在同一目录启动 API。

   ```powershell
   go run ./cmd/api
   ```

3. 在另一个终端启动网页。

   ```powershell
   npm --prefix apps/web ci
   npm --prefix apps/web run dev
   ```

4. 打开 `http://127.0.0.1:5273`，用刚创建的账户登录，再从工作台打开「管理面板」。网页端打开单独标签页，Windows 端打开独立窗口。

默认数据库位于运行账户的配置目录下 `zwOA/zwoa.db`，Windows 上是 `%APPDATA%\zwOA\zwoa.db`。可以通过 `ZWOA_DB_PATH` 指定仅服务账户可访问的文件，初始化和服务启动时须使用同一文件。自动化初始化可以从环境变量 `ZWOA_INIT_ADMIN_PASSWORD` 读取密码，使用后清除变量。数据库和会话由当前 API 实例管理。

API 默认监听 `127.0.0.1:8720`。`GET /healthz` 用于确认进程能处理请求，`GET /api/v1/workbench/overview` 返回系统名称和版本。设置 `ZWOA_ENABLE_OPENAPI=1` 后，可访问 `GET /openapi.json` 查看接口定义。

使用 `ZWOA_API_ADDR` 可调整 API 监听地址。若开发端口改变，用 `ZWOA_DEV_API_TARGET` 指定 Vite 代理目标。Web 生产构建执行 `npm --prefix apps/web run build`，产物在 `apps/web/dist`。部署网页时通过同源 HTTPS 转发 `/api`，保留原始 `Host`，并在 API 进程设置 `ZWOA_SECURE_COOKIES=1`。

如果通过反向代理接收多位用户的请求，可在 `ZWOA_TRUSTED_PROXIES` 中填入代理连接 API 时使用的 IP，以逗号分隔。配置中的代理必须以实际客户端 IP 覆盖 `X-Real-IP`，用于按来源限制登录尝试。

## 账户与管理权限

管理员创建账户或重置密码时，可设置任意非空且不超过 256 字节的密码。账户可直接使用管理员设置的密码登录。停用账户或修改密码会撤销该账户现有会话。用户可通过工作台的账户菜单自行修改密码。

账户可加入多个权限组，功能模块使用权限取各组的并集。团队管理员必须属于所管理的团队或项目。模块管理员的授权按功能模块记录，可以跨团队使用该模块。系统管理面板仅向超级管理员开放。

后续审批模板可从申请所属项目取得项目管理员，再从目标功能模块取得模块管理员。当前版本支持保存这两类人员关系，流程模板、审批节点和流转记录会随业务功能接入。

## 消息

超级管理员可在「权限组」中勾选「消息」并将该组分配给账户。进入工作台的「消息」，可按姓名或登录名选同事发起单聊或群聊。输入时用 Enter 换行，Ctrl+Enter 或 Command+Enter 发送。历史和已读位置保存在服务器，打开消息页时会定时刷新。会话历史仅向会话成员开放。

## Windows 桌面端

需要 Rust 的 Windows MSVC 工具链、Visual C++ Build Tools、Windows SDK 和 WebView2 Runtime。桌面端运行同一套界面，并访问独立启动的 Go API。

```powershell
npm --prefix apps/web ci
npm --prefix apps/desktop ci
npm --prefix apps/desktop run dev
```

桌面开发命令会启动 `5273` 端口的 Vite 服务。可以用浏览器打开同一个地址。已有网页开发服务占用此端口时，先结束该服务再启动桌面开发命令。

```powershell
npm --prefix apps/desktop run build
```

本地构建生成可直接运行的 `apps/desktop/src-tauri/target/release/zwoa-desktop.exe`，默认连接 `http://127.0.0.1:8720`。部署到公司环境时，用 `VITE_API_BASE_URL` 设置实际的 HTTPS API 地址，并把该来源加入 `apps/desktop/src-tauri/tauri.conf.json` 中的 `connect-src`。

API 使用 `ZWOA_ALLOWED_ORIGINS` 控制跨域来源，以逗号分隔完整的协议、主机和端口。Windows 桌面端的 WebView 默认来源为 `http://tauri.localhost`，已包含在本地默认配置中。

## 提交约定

使用 Gitmoji 三段式标题 `表情 (模块): 简述`，标题后空一行，用简短的分点正文说明具体改动。按模块或独立修改点提交。详见 [开发协作](CONTRIBUTING.md)。
