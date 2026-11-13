package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/gogf/gf/v2/net/ghttp"
)

func bindOrganization(server *ghttp.Server, store *organization.Store, api authentication) {
	server.Group("/api/v1/admin", func(group *ghttp.RouterGroup) {
		group.GET("/overview", api.authorized(true, overview(store)))
		group.POST("/accounts", api.authorized(true, saveRecord(store.SaveAccount)))
		group.PUT("/accounts/:id", api.authorized(true, saveRecord(store.SaveAccount)))
		group.POST("/groups", api.authorized(true, saveRecord(store.SaveGroup)))
		group.PUT("/groups/:id", api.authorized(true, saveRecord(store.SaveGroup)))
		group.DELETE("/groups/:id", api.authorized(true, deleteRecord(store.DeleteGroup)))
		group.POST("/teams", api.authorized(true, saveRecord(store.SaveTeam)))
		group.PUT("/teams/:id", api.authorized(true, saveRecord(store.SaveTeam)))
		group.DELETE("/teams/:id", api.authorized(true, deleteRecord(store.DeleteTeam)))
		group.PUT("/modules/:module/admins", api.authorized(true, moduleAdmins(store)))
	})
}

func overview(store *organization.Store) func(*ghttp.Request, organization.Account) {
	return func(r *ghttp.Request, _ organization.Account) {
		data, err := store.Administration(r.Context())
		writeResult(r, data, err)
	}
}

func moduleAdmins(store *organization.Store) func(*ghttp.Request, organization.Account) {
	return func(r *ghttp.Request, _ organization.Account) {
		var input struct {
			AdminIDs []int64 `json:"adminIds"`
		}
		revision, err := adminRevision(r)
		if err == nil {
			err = readJSON(r, &input)
		}
		if err == nil {
			err = store.SaveModuleAdmins(r.Context(), r.GetRouter("module").String(), revision, input.AdminIDs)
		}
		writeResult(r, nil, err)
	}
}

func adminRevision(r *ghttp.Request) (int64, error) {
	revision, err := strconv.ParseInt(r.Header.Get("X-ZwOA-Revision"), 10, 64)
	if err != nil || revision < 0 {
		return 0, organization.ErrInvalid
	}
	return revision, nil
}

func saveRecord[T any](save func(ctx context.Context, id, revision int64, input T) error) func(*ghttp.Request, organization.Account) {
	return func(r *ghttp.Request, _ organization.Account) {
		var input T
		var id int64
		revision, err := adminRevision(r)
		if err == nil {
			err = readJSON(r, &input)
		}
		if err == nil && r.Method == http.MethodPut {
			id, err = recordID(r)
		}
		if err == nil {
			err = save(r.Context(), id, revision, input)
		}
		writeResult(r, nil, err)
	}
}

func deleteRecord(remove func(context.Context, int64, int64) error) func(*ghttp.Request, organization.Account) {
	return func(r *ghttp.Request, _ organization.Account) {
		revision, err := adminRevision(r)
		var id int64
		if err == nil {
			id, err = recordID(r)
		}
		if err == nil {
			err = remove(r.Context(), id, revision)
		}
		writeResult(r, nil, err)
	}
}
