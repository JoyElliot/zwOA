package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"github.com/JoyElliot/zwOA/internal/modules/chat"
	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/gogf/gf/v2/net/ghttp"
)

const sessionCookie = "zwoa_session"

type organizationAPI struct {
	store  *organization.Store
	config Config
}

func bindOrganization(server *ghttp.Server, store *organization.Store, config Config) {
	api := organizationAPI{store: store, config: config}
	server.Group("/api/v1/auth", func(group *ghttp.RouterGroup) {
		group.POST("/login", api.login(false))
		group.POST("/desktop/login", api.login(true))
		group.GET("/session", api.authorized(false, api.session))
		group.POST("/logout", api.logout)
		group.PUT("/password", api.authorized(false, api.password))
	})
	server.Group("/api/v1/admin", func(group *ghttp.RouterGroup) {
		group.GET("/overview", api.authorized(true, api.overview))
		group.POST("/accounts", api.authorized(true, saveRecord(store.SaveAccount)))
		group.PUT("/accounts/:id", api.authorized(true, saveRecord(store.SaveAccount)))
		group.POST("/groups", api.authorized(true, saveRecord(store.SaveGroup)))
		group.PUT("/groups/:id", api.authorized(true, saveRecord(store.SaveGroup)))
		group.DELETE("/groups/:id", api.authorized(true, deleteRecord(store.DeleteGroup)))
		group.POST("/teams", api.authorized(true, saveRecord(store.SaveTeam)))
		group.PUT("/teams/:id", api.authorized(true, saveRecord(store.SaveTeam)))
		group.DELETE("/teams/:id", api.authorized(true, deleteRecord(store.DeleteTeam)))
		group.PUT("/modules/:module/admins", api.authorized(true, api.moduleAdmins))
	})
}

func (api organizationAPI) login(desktop bool) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := readJSON(r, &input); err != nil {
			writeError(r, err)
			return
		}
		previous := api.cookieToken(r)
		if desktop {
			previous = bearerToken(r)
		}
		token, err := api.store.Login(r.Context(), input.Username, input.Password, api.clientIP(r), previous)
		if err != nil {
			writeError(r, err)
			return
		}
		if desktop {
			writeResult(r, map[string]string{"token": token}, nil)
			return
		}
		api.setCookie(r, token, int(organization.SessionLifetime.Seconds()))
		writeResult(r, nil, nil)
	}
}

func (api organizationAPI) clientIP(r *ghttp.Request) string {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	address, err := netip.ParseAddr(peer)
	if err == nil && slices.Contains(api.config.TrustedProxies, address.Unmap()) {
		if client, err := netip.ParseAddr(r.Header.Get("X-Real-IP")); err == nil {
			return client.Unmap().String()
		}
	}
	return peer
}

func (api organizationAPI) authorized(admin bool, handle func(*ghttp.Request, organization.Account)) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		if r.Method != http.MethodGet && r.Header.Get("X-ZwOA-Request") != "1" {
			writeError(r, organization.ErrForbidden)
			return
		}
		account, err := api.store.Session(r.Context(), api.sessionToken(r))
		if err != nil {
			writeError(r, err)
			return
		}
		if admin && (!account.SuperAdmin || account.MustChangePassword) {
			writeError(r, organization.ErrForbidden)
			return
		}
		handle(r, account)
	}
}

func (api organizationAPI) session(r *ghttp.Request, account organization.Account) {
	identity, err := api.store.Identity(r.Context(), account)
	writeResult(r, identity, err)
}

func (api organizationAPI) logout(r *ghttp.Request) {
	if r.Header.Get("X-ZwOA-Request") != "1" {
		writeError(r, organization.ErrForbidden)
		return
	}
	if err := api.store.Logout(r.Context(), api.sessionToken(r)); err != nil {
		writeError(r, err)
		return
	}
	api.setCookie(r, "", -1)
	writeResult(r, nil, nil)
}

func (api organizationAPI) password(r *ghttp.Request, account organization.Account) {
	var input struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	err := readJSON(r, &input)
	if err == nil {
		err = api.store.ChangePassword(r.Context(), account.ID, input.Current, input.Next)
	}
	if err == nil {
		api.setCookie(r, "", -1)
	}
	writeResult(r, nil, err)
}

func (api organizationAPI) overview(r *ghttp.Request, _ organization.Account) {
	data, err := api.store.Administration(r.Context())
	writeResult(r, data, err)
}

func (api organizationAPI) moduleAdmins(r *ghttp.Request, _ organization.Account) {
	var input struct {
		AdminIDs []int64 `json:"adminIds"`
	}
	revision, err := adminRevision(r)
	if err == nil {
		err = readJSON(r, &input)
	}
	if err == nil {
		err = api.store.SaveModuleAdmins(r.Context(), r.GetRouter("module").String(), revision, input.AdminIDs)
	}
	writeResult(r, nil, err)
}

func (api organizationAPI) setCookie(r *ghttp.Request, token string, maxAge int) {
	r.Response.Header().Add("Set-Cookie", (&http.Cookie{
		Name: api.cookieName(r), Value: token, Path: "/", HttpOnly: true,
		Secure: api.secureCookie(r), SameSite: http.SameSiteLaxMode, MaxAge: maxAge,
	}).String())
}

func (api organizationAPI) secureCookie(r *ghttp.Request) bool {
	return api.config.SecureCookies || r.TLS != nil
}

func (api organizationAPI) cookieName(r *ghttp.Request) string {
	if api.secureCookie(r) {
		return "__Host-" + sessionCookie
	}
	return sessionCookie
}

func (api organizationAPI) sessionToken(r *ghttp.Request) string {
	if token := bearerToken(r); token != "" {
		return token
	}
	return api.cookieToken(r)
}

func bearerToken(r *ghttp.Request) string {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return token
}

func (api organizationAPI) cookieToken(r *ghttp.Request) string {
	cookie, err := r.Request.Cookie(api.cookieName(r))
	if err != nil {
		return ""
	}
	return cookie.Value
}

func readJSON(r *ghttp.Request, value any) error {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || r.Header.Get("X-ZwOA-Request") != "1" {
		return organization.ErrInvalid
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	object := strings.TrimSpace(string(body))
	if !strings.HasPrefix(object, "{") {
		return organization.ErrInvalid
	}
	decoder := json.NewDecoder(strings.NewReader(object))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil || decoder.Decode(new(any)) != io.EOF {
		return organization.ErrInvalid
	}
	return nil
}

func writeResult(r *ghttp.Request, data any, err error) {
	if err != nil {
		writeError(r, err)
		return
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteJson(ghttp.DefaultHandlerResponse{Data: data})
}

func writeError(r *ghttp.Request, err error) {
	status := http.StatusInternalServerError
	message := err.Error()
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		status = http.StatusRequestEntityTooLarge
		message = "请求内容过大"
	case errors.Is(err, organization.ErrInvalid), errors.Is(err, organization.ErrPassword), errors.Is(err, chat.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, organization.ErrUnauthorized), errors.Is(err, organization.ErrCredentials):
		status = http.StatusUnauthorized
	case errors.Is(err, organization.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, organization.ErrNotFound), errors.Is(err, chat.ErrMissing):
		status = http.StatusNotFound
	case errors.Is(err, organization.ErrConflict), errors.Is(err, chat.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, organization.ErrStale):
		status = http.StatusPreconditionFailed
	case errors.Is(err, organization.ErrTooMany):
		status = http.StatusTooManyRequests
		r.Response.Header().Set("Retry-After", "60")
	default:
		log.Printf("业务接口失败：%v", err)
		message = "服务暂时不可用"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJson(ghttp.DefaultHandlerResponse{Code: status, Message: message})
}

func recordID(r *ghttp.Request) (int64, error) {
	id, err := strconv.ParseInt(r.GetRouter("id").String(), 10, 64)
	if err != nil || id <= 0 {
		return 0, organization.ErrInvalid
	}
	return id, nil
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
