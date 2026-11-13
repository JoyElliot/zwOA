package httpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JoyElliot/zwOA/internal/modules/chat"
	"github.com/JoyElliot/zwOA/internal/modules/organization"
)

func TestOrganizationAccess(t *testing.T) {
	const desktopOrigin = "http://tauri.localhost"
	store, err := organization.Open(t.Context(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Bootstrap(t.Context(), "root-admin", "管理员", "bootstrap-password"); err != nil {
		t.Fatal(err)
	}
	server := New(Config{Address: "127.0.0.1:0", AllowedOrigins: []string{desktopOrigin}, SecureCookies: true}, store)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Shutdown() })
	client := &http.Client{Timeout: 5 * time.Second}
	var revision int64

	call := func(method, path, body string, cookie *http.Cookie, headers map[string]string, want int) (*http.Response, []byte) {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), method, "http://"+server.GetListenedAddress()+"/api/v1"+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "oa.example.test"
		if cookie != nil {
			req.AddCookie(cookie)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-ZwOA-Request", "1")
		req.Header.Set("X-ZwOA-Revision", strconv.FormatInt(revision, 10))
		for name, value := range headers {
			req.Header.Set(name, value)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s = %d, want %d: %s", method, path, res.StatusCode, want, data)
		}
		if want == http.StatusOK && strings.HasPrefix(path, "/admin/") && method != http.MethodGet {
			revision++
		}
		return res, data
	}
	call("GET", "/admin/overview", "", nil, nil, 401)
	call("POST", "/auth/login", `{"username":"root-admin","password":"bootstrap-password"}`, nil,
		map[string]string{"Origin": "https://other.example.test"}, 403)
	res, _ := call("POST", "/auth/login", `{"username":"root-admin","password":"bootstrap-password"}`, nil,
		map[string]string{"Origin": "https://oa.example.test"}, 200)
	cookies := res.Cookies()
	if len(cookies) != 1 || cookies[0].Name != "__Host-zwoa_session" || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].Path != "/" {
		t.Fatal("登录响应缺少预期的安全会话 Cookie")
	}
	admin := cookies[0]
	call("OPTIONS", "/auth/logout", "", admin, map[string]string{"Access-Control-Request-Method": "POST"}, 204)
	call("GET", "/admin/overview", "", admin, nil, 200)
	call("POST", "/admin/groups", `{"name":"审批成员","description":"","permissions":["approvals.access"]}`, admin, nil, 200)
	call("POST", "/admin/accounts", `{"username":"member","name":"成员","password":"temporary-password","active":true,"groupIds":[1]}`, admin, nil, 200)
	call("POST", "/admin/teams", `{"name":"北区项目","kind":"project","memberIds":[2],"adminIds":[1]}`, admin, nil, 400)
	call("POST", "/admin/teams", `{"name":"北区项目","kind":"project","memberIds":[2],"adminIds":[2]}`, admin, nil, 200)
	call("PUT", "/admin/modules/approvals/admins", `{"adminIds":[1]}`, admin, nil, 200)
	call("PUT", "/admin/accounts/1?id=2", `{"username":"member","name":"成员","active":false,"groupIds":[]}`, admin, nil, 403)

	res, _ = call("OPTIONS", "/auth/desktop/login", "", nil, map[string]string{"Origin": desktopOrigin, "Access-Control-Request-Method": "POST"}, 204)
	if res.Header.Get("Access-Control-Allow-Origin") != desktopOrigin {
		t.Fatal("桌面来源的预检缺少允许来源")
	}
	memberLogin := func(password string) map[string]string {
		t.Helper()
		_, data := call("POST", "/auth/desktop/login", `{"username":"member","password":"`+password+`"}`, nil,
			map[string]string{"Origin": desktopOrigin}, 200)
		var result struct{ Data struct{ Token string } }
		if err := json.Unmarshal(data, &result); err != nil || result.Data.Token == "" {
			t.Fatal("桌面登录未返回会话")
		}
		return map[string]string{"Authorization": "Bearer " + result.Data.Token, "Origin": desktopOrigin}
	}
	member := memberLogin("temporary-password")
	call("GET", "/admin/overview", "", nil, member, 403)
	call("PUT", "/auth/password", `{"current":"temporary-password","next":"replacement-password"}`, nil, member, 200)
	call("GET", "/auth/session", "", nil, member, 401)
	member = memberLogin("replacement-password")
	_, data := call("GET", "/auth/session", "", nil, member, 200)
	var session struct{ Data organization.Identity }
	if err := json.Unmarshal(data, &session); err != nil {
		t.Fatal(err)
	}
	if len(session.Data.Permissions) != 1 || session.Data.Permissions[0] != "approvals.access" ||
		len(session.Data.Teams) != 1 || !session.Data.Teams[0].Admin || len(session.Data.Modules) != 0 {
		t.Fatalf("团队和模块权限范围错误：%+v", session.Data)
	}
	call("PUT", "/admin/modules/approvals/admins", `{"adminIds":[2]}`, nil, member, 403)
	call("GET", "/chat/conversations", "", nil, member, 403)
	call("PUT", "/admin/groups/1", `{"name":"审批成员","permissions":["approvals.access","chat.access"]}`, admin, nil, 200)
	call("GET", "/chat/contacts?q=成员", "", admin, nil, 200)
	_, data = call("POST", "/chat/conversations", `{"memberIds":[2],"requestKey":"00000000000000000000000000000001"}`, admin, nil, 200)
	var conversation struct{ Data struct{ ID int64 } }
	if err := json.Unmarshal(data, &conversation); err != nil || conversation.Data.ID <= 0 {
		t.Fatalf("创建单聊失败：%s", data)
	}
	_, data = call("POST", "/chat/conversations", `{"memberIds":[1],"requestKey":"00000000000000000000000000000002"}`, nil, member, 200)
	if !strings.Contains(string(data), fmt.Sprintf(`"id":%d`, conversation.Data.ID)) {
		t.Fatalf("双方发起单聊返回不同会话：%s", data)
	}
	thread := fmt.Sprintf("/chat/conversations/%d", conversation.Data.ID)
	const send = `{"text":"你好","requestKey":"00000000000000000000000000000003"}`
	_, first := call("POST", thread+"/messages", send, admin, nil, 200)
	_, retry := call("POST", thread+"/messages", send, admin, nil, 200)
	if string(first) != string(retry) {
		t.Fatalf("重试发送返回了不同的消息：%s, %s", first, retry)
	}
	_, data = call("GET", thread+"/messages", "", nil, member, 200)
	var history struct{ Data chat.MessagePage }
	if err := json.Unmarshal(data, &history); err != nil || len(history.Data.Messages) != 1 || history.Data.Messages[0].Text != "你好" {
		t.Fatalf("会话历史不正确：%s", data)
	}
	checkUnread := func(want int) {
		t.Helper()
		_, data := call("GET", "/chat/conversations", "", nil, member, 200)
		var result struct{ Data []chat.Conversation }
		if err := json.Unmarshal(data, &result); err != nil || len(result.Data) != 1 || result.Data[0].Unread != want {
			t.Fatalf("未读数应为 %d：%s", want, data)
		}
	}
	checkUnread(1)
	call("PUT", thread+"/read", `{"messageId":999}`, nil, member, 404)
	call("PUT", thread+"/read", fmt.Sprintf(`{"messageId":%d}`, history.Data.Messages[0].ID), nil, member, 200)
	checkUnread(0)
	call("POST", "/admin/accounts", `{"username":"outsider","name":"其他人","password":"temporary-password","active":true,"groupIds":[1]}`, admin, nil, 200)
	res, _ = call("POST", "/auth/login", `{"username":"outsider","password":"temporary-password"}`, nil, nil, 200)
	outsider := res.Cookies()[0]
	call("GET", thread+"/messages", "", outsider, nil, 403)
	call("PUT", "/auth/password", `{"current":"temporary-password","next":"replacement-password"}`, outsider, nil, 200)
	res, _ = call("POST", "/auth/login", `{"username":"outsider","password":"replacement-password"}`, nil, nil, 200)
	outsider = res.Cookies()[0]
	call("GET", thread+"/messages", "", outsider, nil, 404)
	call("POST", thread+"/messages", send, outsider, nil, 404)
	call("PUT", thread+"/read", `{"messageId":1}`, outsider, nil, 404)
	call("PUT", "/admin/accounts/2", `{"name":"成员","active":false,"groupIds":[1]}`, admin, nil, 200)
	call("PUT", "/admin/accounts/2", `{"name":"成员","active":true,"groupIds":[1]}`, admin,
		map[string]string{"X-ZwOA-Revision": strconv.FormatInt(revision-1, 10)}, 412)
	call("GET", "/auth/session", "", nil, member, 401)
	call("GET", "/chat/conversations", "", nil, member, 401)
	call("POST", "/auth/logout", "", admin, nil, 200)
	call("GET", "/admin/overview", "", admin, nil, 401)
}
