package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func checkOpenAPI(t *testing.T, client *http.Client, url string) {
	t.Helper()
	res, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("OpenAPI 状态 = %d", res.StatusCode)
	}
	type operation struct {
		Security   []map[string][]string
		Parameters []struct {
			In, Name string
			Required bool
		}
		Responses map[string]json.RawMessage
	}
	var spec struct {
		Paths      map[string]map[string]operation
		Components struct {
			SecuritySchemes map[string]json.RawMessage
		}
	}
	if err := json.NewDecoder(res.Body).Decode(&spec); err != nil {
		t.Fatal(err)
	}
	for path, methods := range map[string][]string{
		"/auth/login":                       {"post"},
		"/auth/desktop/login":               {"post"},
		"/auth/session":                     {"get"},
		"/auth/logout":                      {"post"},
		"/auth/password":                    {"put"},
		"/auth/avatar":                      {"put"},
		"/auth/avatars/{version}":           {"get"},
		"/chat/contacts":                    {"get"},
		"/chat/conversations":               {"get", "post"},
		"/chat/conversations/{id}/messages": {"get", "post"},
		"/chat/conversations/{id}/read":     {"put"},
		"/admin/overview":                   {"get"},
		"/admin/accounts":                   {"post"},
		"/admin/accounts/{id}":              {"put"},
		"/admin/groups":                     {"post"},
		"/admin/groups/{id}":                {"put", "delete"},
		"/admin/teams":                      {"post"},
		"/admin/teams/{id}":                 {"put", "delete"},
		"/admin/modules/{module}/admins":    {"put"},
		"/workbench/overview":               {"get"},
	} {
		for _, method := range methods {
			if len(spec.Paths["/api/v1"+path][method].Responses["200"]) == 0 {
				t.Errorf("OpenAPI 未记录 %s %s 的成功响应", method, path)
			}
		}
	}

	admin := spec.Paths["/api/v1/admin/accounts/{id}"]["put"]
	required := map[string]bool{}
	for _, param := range admin.Parameters {
		if param.Required {
			required[param.In+":"+param.Name] = true
		}
	}
	for _, name := range []string{"path:id", "header:X-ZwOA-Revision", "header:X-ZwOA-Request"} {
		if !required[name] {
			t.Errorf("OpenAPI 未声明必填参数 %s", name)
		}
	}
	if len(admin.Responses["412"]) == 0 {
		t.Error("OpenAPI 未记录管理版本冲突")
	}
	if len(admin.Security) != 3 {
		t.Fatalf("OpenAPI 管理认证选项数 = %d, want 3", len(admin.Security))
	}
	for _, alternative := range admin.Security {
		if len(alternative) != 1 {
			t.Fatalf("每个认证选项应只有一种凭据，实际包含 %d 种", len(alternative))
		}
		for name := range alternative {
			if len(spec.Components.SecuritySchemes[name]) == 0 {
				t.Errorf("OpenAPI 未定义认证方式 %s", name)
			}
		}
	}
}
