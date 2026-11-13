# 接口调试

在本地启动 API 前设置 `ZWOA_ENABLE_OPENAPI=1`，然后打开 [OpenAPI JSON](http://127.0.0.1:8720/openapi.json)。文档包含登录、消息、后台管理和工作台接口的请求字段及响应。默认关闭文档端点。在本地调试时使用测试账户和测试数据。

## 登录和响应

- Web 通过 `POST /api/v1/auth/login` 登录，后续请求使用浏览器接收的 HttpOnly 会话 Cookie。
- Windows 客户端通过 `POST /api/v1/auth/desktop/login` 获得 `data.token`，后续请求使用 `Authorization: Bearer <token>`。
- 写请求带 `X-ZwOA-Request: 1`。有请求体时使用 `Content-Type: application/json`，按接口定义传入单个 JSON 对象。
- 业务接口以 `{code, message, data}` 返回。成功的 `code` 为 `0`。遇到 `401` 时检查登录状态，`403` 时检查权限和写请求头，`429` 时按 `Retry-After` 等待。

## 消息和管理

- 消息接口需要消息模块访问权限。历史消息可以按 `before` 向前分页，或按 `after` 增量获取，两者不能同时使用。创建会话或发送消息重试时，应复用原请求的 `requestKey`。
- 管理接口仅供超级管理员使用。写入前从 `GET /api/v1/admin/overview` 取得 `data.revision`，并作为 `X-ZwOA-Revision` 发送。若返回 `412`，先重新读取管理概览并核对当前数据，再提交。
