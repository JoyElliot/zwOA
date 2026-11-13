package api

import (
	"strconv"

	"github.com/JoyElliot/zwOA/internal/modules/chat"
	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/gogf/gf/v2/net/ghttp"
)

func bindChat(server *ghttp.Server, store *chat.Store, auth authentication) {
	handle := func(fn func(*ghttp.Request, int64)) ghttp.HandlerFunc {
		return auth.authorized(false, func(r *ghttp.Request, account organization.Account) {
			allowed, err := store.CanAccess(r.Context(), account.ID)
			if err != nil {
				writeError(r, err)
				return
			}
			if !allowed {
				writeError(r, organization.ErrForbidden)
				return
			}
			fn(r, account.ID)
		})
	}
	server.Group("/api/v1/chat", func(group *ghttp.RouterGroup) {
		group.GET("/contacts", handle(func(r *ghttp.Request, actor int64) {
			contacts, err := store.Contacts(r.Context(), actor, r.URL.Query().Get("q"))
			writeResult(r, contacts, err)
		}))
		group.GET("/conversations", handle(func(r *ghttp.Request, actor int64) {
			conversations, err := store.Conversations(r.Context(), actor)
			writeResult(r, conversations, err)
		}))
		group.POST("/conversations", handle(func(r *ghttp.Request, actor int64) {
			var input chat.CreateInput
			if err := readJSON(r, &input); err != nil {
				writeError(r, err)
				return
			}
			id, err := store.Create(r.Context(), actor, input)
			writeResult(r, map[string]int64{"id": id}, err)
		}))
		group.GET("/conversations/:id/messages", handle(func(r *ghttp.Request, actor int64) {
			id, err := recordID(r)
			query := r.URL.Query()
			before, beforeErr := messageCursor(query.Get("before"))
			after, afterErr := messageCursor(query.Get("after"))
			if err != nil || beforeErr != nil || afterErr != nil || (before != 0 && after != 0) {
				writeError(r, chat.ErrInvalid)
				return
			}
			page, err := store.Messages(r.Context(), actor, id, before, after)
			writeResult(r, page, err)
		}))
		group.POST("/conversations/:id/messages", handle(func(r *ghttp.Request, actor int64) {
			id, err := recordID(r)
			var input chat.SendInput
			if err == nil {
				err = readJSON(r, &input)
			}
			if err != nil {
				writeError(r, err)
				return
			}
			message, err := store.Send(r.Context(), actor, id, input)
			writeResult(r, message, err)
		}))
		group.PUT("/conversations/:id/read", handle(func(r *ghttp.Request, actor int64) {
			id, err := recordID(r)
			var input struct {
				MessageID int64 `json:"messageId"`
			}
			if err == nil {
				err = readJSON(r, &input)
			}
			if err == nil {
				err = store.Read(r.Context(), actor, id, input.MessageID)
			}
			writeResult(r, nil, err)
		}))
	})
}

func messageCursor(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, chat.ErrInvalid
	}
	return id, nil
}
