package api

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/JoyElliot/zwOA/internal/modules/chat"
	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/net/goai"
)

type apiOperation struct {
	method, path, summary, description string
	input, output                      any
	params                             goai.Parameters
	statuses                           []int
	setCookie                          string
}

var sessionSecurity = goai.SecurityRequirements{
	{"DesktopBearer": {}},
	{"SessionCookie": {}},
	{"SecureSessionCookie": {}},
}

// describeOpenAPI 为手动绑定的业务路由登记文档。认证、验证和响应仍由原处理器执行。
func describeOpenAPI(doc *goai.OpenApiV3) {
	doc.Info.Title = "中网联合 OA / CRM API"
	doc.Info.Version = "1"
	doc.Info.Description = "写请求使用 X-ZwOA-Request: 1。JSON 请求体只接受接口声明的字段；管理写请求另需最新的 X-ZwOA-Revision。"
	doc.Config.CommonResponse = ghttp.DefaultHandlerResponse{}
	doc.Config.CommonResponseDataField = "Data"
	doc.Paths = make(goai.Paths)
	doc.Components.SecuritySchemes = goai.SecuritySchemes{
		"DesktopBearer":       {Value: &goai.SecurityScheme{Type: "http", Scheme: "bearer"}},
		"SessionCookie":       {Value: &goai.SecurityScheme{Type: "apiKey", In: "cookie", Name: sessionCookie}},
		"SecureSessionCookie": {Value: &goai.SecurityScheme{Type: "apiKey", In: "cookie", Name: "__Host-" + sessionCookie}},
	}

	describeAuth(doc)
	describeChat(doc)
	describeAdministration(doc)
}

func describeAuth(doc *goai.OpenApiV3) {
	loginErrors := []int{400, 401, 403, 413, 429}
	optionalSession := append(goai.SecurityRequirements{{}}, sessionSecurity...)
	optionalBearer := goai.SecurityRequirements{{}, {"DesktopBearer": {}}}
	cookie := "浏览器会话 Cookie，Max-Age=43200、HttpOnly、Path=/、SameSite=Lax；安全模式设置 Secure 并使用 __Host- 前缀。"
	addOperation(doc, "认证", "auth", nil, apiOperation{
		method: "POST", path: "/login", summary: "网页登录",
		input: loginInput{}, statuses: loginErrors, setCookie: cookie,
	})
	addOperation(doc, "认证", "auth", &optionalBearer, apiOperation{
		method: "POST", path: "/desktop/login", summary: "桌面端登录",
		description: "返回桌面端用于 Authorization: Bearer 的令牌。若请求携带旧 Bearer，成功后撤销旧令牌。",
		input:       loginInput{}, output: desktopLoginResult{}, statuses: loginErrors,
	})
	addOperation(doc, "认证", "auth", &sessionSecurity, apiOperation{
		method: "GET", path: "/session", summary: "获取当前会话身份及权限",
		output: organization.Identity{}, statuses: []int{401},
	})
	addOperation(doc, "账户", "auth", &sessionSecurity, apiOperation{
		method: "PUT", path: "/avatar", summary: "更新自己的头像",
		description: "接收 Base64 编码的 JPEG。图片不超过 128 KiB，任一边不超过 256 像素。",
		input:       avatarImage{}, statuses: []int{400, 401, 403, 413},
	})
	addOperation(doc, "账户", "auth", &sessionSecurity, apiOperation{
		method: "GET", path: "/avatars/{version}", summary: "读取指定版本的头像",
		params: goai.Parameters{parameter("version", "path", "头像版本", true, goai.Schema{Type: "string", Pattern: "^[a-f0-9]{32}$"})},
		output: avatarImage{}, statuses: []int{401, 403, 404},
	})
	addOperation(doc, "认证", "auth", &optionalSession, apiOperation{
		method: "POST", path: "/logout", summary: "退出登录", description: "撤销当前会话。无有效会话时也可调用。",
		statuses: []int{403}, setCookie: "清除浏览器会话 Cookie。",
	})
	addOperation(doc, "认证", "auth", &sessionSecurity, apiOperation{
		method: "PUT", path: "/password", summary: "修改当前账号密码",
		description: "验证当前密码。成功后撤销该账号的所有会话。",
		input:       passwordInput{}, statuses: []int{400, 401, 403, 413, 429}, setCookie: "清除浏览器会话 Cookie。",
	})
}

func describeChat(doc *goai.OpenApiV3) {
	const access = "需要消息模块访问权限。"
	const member = access + " 仅会话成员可以访问；不存在或无权访问的会话均返回 404。"
	id := parameter("id", "path", "会话 ID", true, intSchema(1))
	operations := []apiOperation{
		{method: "GET", path: "/contacts", summary: "查询可发起会话的联系人", description: access,
			params: goai.Parameters{parameter("q", "query", "按姓名或登录名搜索，可留空", false, goai.Schema{Type: "string"})}, output: []chat.Contact{}, statuses: []int{400, 401, 403}},
		{method: "GET", path: "/conversations", summary: "列出自己的消息会话", description: access,
			output: []chat.Conversation{}, statuses: []int{401, 403}},
		{method: "POST", path: "/conversations", summary: "创建单聊或群聊", description: access + " 一个成员且标题为空时创建单聊。最多可选择 99 位成员，标题最多 80 个字符。使用同一 requestKey 重试可取回已创建的会话 ID。",
			input: chat.CreateInput{}, output: conversationResult{}, statuses: []int{400, 401, 403, 413}},
		{method: "GET", path: "/conversations/{id}/messages", summary: "分页读取消息",
			description: member + " before 和 after 只能传一个；都不传时返回最新一页，每页最多 50 条。",
			params: goai.Parameters{id, parameter("before", "query", "查询该消息 ID 之前的历史", false, intSchema(1)),
				parameter("after", "query", "查询该消息 ID 之后的新消息", false, intSchema(1))}, output: chat.MessagePage{}, statuses: []int{400, 401, 403, 404}},
		{method: "POST", path: "/conversations/{id}/messages", summary: "发送消息",
			description: member + " 文本去掉首尾空白后应为 1–5000 个字符。使用相同 requestKey 重试相同文本可取回已保存的消息；同键不同文本返回 409。",
			input:       chat.SendInput{}, output: chat.Message{}, params: goai.Parameters{id}, statuses: []int{400, 401, 403, 404, 409, 413}},
		{method: "PUT", path: "/conversations/{id}/read", summary: "更新会话已读位置", description: member + " 已读位置只能向前推进。",
			input: readInput{}, params: goai.Parameters{id}, statuses: []int{400, 401, 403, 404, 413}},
	}
	for _, op := range operations {
		addOperation(doc, "消息", "chat", &sessionSecurity, op)
	}
}

func describeAdministration(doc *goai.OpenApiV3) {
	const access = "仅超级管理员可访问。"
	revision := parameter("X-ZwOA-Revision", "header", "管理概览最新 revision；版本过期返回 412，刷新概览后再提交", true, intSchema(0))
	id := parameter("id", "path", "记录 ID", true, intSchema(1))
	module := parameter("module", "path", "功能模块键", true, goai.Schema{Type: "string", Enum: []any{"chat", "crm", "approvals", "organization"}})
	addOperation(doc, "管理", "admin", &sessionSecurity, apiOperation{
		method: "GET", path: "/overview", summary: "获取管理概览及版本", description: access,
		output: organization.Administration{}, statuses: []int{401, 403},
	})
	operations := []apiOperation{
		{method: "POST", path: "/accounts", summary: "创建账户", description: "需填写登录名、姓名和密码。密码非空且最多 256 字节。",
			input: organization.AccountInput{}, statuses: []int{409, 429}},
		{method: "PUT", path: "/accounts/{id}", summary: "更新账户", description: "登录名不可改，密码留空保留原密码。停用或重置密码会撤销该账户的会话。",
			input: organization.AccountInput{}, params: goai.Parameters{id}, statuses: []int{404, 409, 429}},
		{method: "POST", path: "/groups", summary: "创建权限组", input: organization.PermissionGroup{}, statuses: []int{409}},
		{method: "PUT", path: "/groups/{id}", summary: "更新权限组", input: organization.PermissionGroup{}, params: goai.Parameters{id}, statuses: []int{404, 409}},
		{method: "DELETE", path: "/groups/{id}", summary: "删除权限组", params: goai.Parameters{id}, statuses: []int{404}},
		{method: "POST", path: "/teams", summary: "创建团队或项目", input: organization.Team{}, statuses: []int{409}},
		{method: "PUT", path: "/teams/{id}", summary: "更新团队或项目", input: organization.Team{}, params: goai.Parameters{id}, statuses: []int{404, 409}},
		{method: "DELETE", path: "/teams/{id}", summary: "删除团队或项目", params: goai.Parameters{id}, statuses: []int{404}},
		{method: "PUT", path: "/modules/{module}/admins", summary: "替换模块管理员", input: moduleAdminsInput{}, params: goai.Parameters{module}, statuses: []int{404, 409}},
	}
	for _, op := range operations {
		op.description = strings.TrimSpace(access + " " + op.description)
		op.params = append(op.params, revision)
		op.statuses = append(op.statuses, 400, 401, 403, 412)
		if op.input != nil {
			op.statuses = append(op.statuses, 413)
		}
		addOperation(doc, "管理", "admin", &sessionSecurity, op)
	}
}

func addOperation(doc *goai.OpenApiV3, tag, prefix string, security *goai.SecurityRequirements, op apiOperation) {
	operation := &goai.Operation{
		Tags: []string{tag}, Summary: op.summary, Description: op.description, Security: security,
		Parameters: op.params, Responses: goai.Responses{},
	}
	if op.method != http.MethodGet {
		requestHeader := goai.Schema{Type: "string", Enum: []any{"1"}}
		operation.Parameters = append(operation.Parameters, parameter("X-ZwOA-Request", "header", "写请求标识，固定为 1", true, requestHeader))
	}
	if op.input != nil {
		input := schemaFor(doc, op.input)
		operation.RequestBody = &goai.RequestBodyRef{Value: &goai.RequestBody{
			Required: true,
			Content:  goai.Content{"application/json": {Schema: input}},
		}}
	}
	data := inlineSchema(goai.Schema{Type: "object", Nullable: true, Enum: []any{nil}})
	if op.output != nil {
		data = schemaFor(doc, op.output)
	}
	success := jsonResponse(200, "请求成功", data)
	if op.setCookie != "" {
		success.Headers["Set-Cookie"] = responseHeader(op.setCookie, "string", nil)
	}
	operation.Responses["200"] = goai.ResponseRef{Value: success}
	for _, status := range append(op.statuses, 500) {
		response := jsonResponse(status, errorDescriptions[status], inlineSchema(goai.Schema{
			Type: "object", Nullable: true, Enum: []any{nil},
		}))
		if status == 429 {
			response.Headers["Retry-After"] = responseHeader("等待秒数", "integer", 60)
		}
		operation.Responses[strconv.Itoa(status)] = goai.ResponseRef{Value: response}
	}
	path := "/api/v1/" + prefix + op.path
	item := doc.Paths[path]
	switch op.method {
	case http.MethodGet:
		item.Get = operation
	case http.MethodPost:
		item.Post = operation
	case http.MethodPut:
		item.Put = operation
	case http.MethodDelete:
		item.Delete = operation
	}
	doc.Paths[path] = item
}

var errorDescriptions = map[int]string{
	400: "请求参数、请求头或 JSON 格式无效",
	401: "登录凭据错误或会话无效",
	403: "访问权限不足、跨域来源无效或缺少写请求标识",
	404: "记录、会话或消息不存在",
	409: "数据冲突或请求键已用于不同内容",
	412: "管理数据版本过期，请重新获取管理概览",
	413: "请求体超出大小限制",
	429: "请求过于频繁，请稍后重试",
	500: "服务暂时不可用",
}

func jsonResponse(status int, description string, data *goai.SchemaRef) *goai.Response {
	var properties goai.Schemas
	code := status
	if status == 200 {
		code = 0
	}
	properties.Set("code", *inlineSchema(goai.Schema{Type: "integer", Enum: []any{code}}))
	properties.Set("message", *inlineSchema(goai.Schema{Type: "string"}))
	properties.Set("data", *data)
	return &goai.Response{
		Description: description,
		Headers:     goai.Headers{"Cache-Control": responseHeader("会话和业务响应不缓存", "string", "no-store")},
		Content: goai.Content{"application/json": {Schema: inlineSchema(goai.Schema{
			Type: "object", Properties: &properties, Required: []string{"code", "message", "data"},
		})}},
	}
}

func schemaFor(doc *goai.OpenApiV3, value any) *goai.SchemaRef {
	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Slice {
		item := schemaFor(doc, reflect.Zero(typ.Elem()).Interface())
		return inlineSchema(goai.Schema{Type: "array", Items: item})
	}
	if err := doc.Add(goai.AddInput{Object: value}); err != nil {
		panic(fmt.Errorf("登记 OpenAPI 数据类型 %s: %w", typ, err))
	}
	name := strings.ReplaceAll(typ.PkgPath(), "/", ".") + "." + typ.Name()
	return &goai.SchemaRef{Ref: name}
}

func parameter(name, in, description string, required bool, schema goai.Schema) goai.ParameterRef {
	return goai.ParameterRef{Value: &goai.Parameter{
		Name: name, In: in, Description: description, Required: required, Schema: inlineSchema(schema),
	}}
}

func intSchema(minimum float64) goai.Schema {
	return goai.Schema{Type: "integer", Format: "int64", Min: &minimum}
}

func inlineSchema(schema goai.Schema) *goai.SchemaRef {
	return &goai.SchemaRef{Value: &schema}
}

func responseHeader(description, kind string, example any) goai.HeaderRef {
	return goai.HeaderRef{Value: &goai.Header{Parameter: goai.Parameter{
		Description: description, Schema: inlineSchema(goai.Schema{Type: kind}), Example: example,
	}}}
}
