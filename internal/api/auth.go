package api

import (
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/JoyElliot/zwOA/internal/platform/httpserver"
	"github.com/gogf/gf/v2/net/ghttp"
)

const sessionCookie = "zwoa_session"

type authentication struct {
	store  *organization.Store
	config httpserver.Config
}

func bindAuth(server *ghttp.Server, api authentication) {
	server.Group("/api/v1/auth", func(group *ghttp.RouterGroup) {
		group.POST("/login", api.login(false))
		group.POST("/desktop/login", api.login(true))
		group.GET("/session", api.authorized(false, api.session))
		group.POST("/logout", api.logout)
		group.PUT("/password", api.authorized(false, api.password))
	})
}

func (api authentication) login(desktop bool) ghttp.HandlerFunc {
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

func (api authentication) clientIP(r *ghttp.Request) string {
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

func (api authentication) authorized(admin bool, handle func(*ghttp.Request, organization.Account)) ghttp.HandlerFunc {
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

func (api authentication) session(r *ghttp.Request, account organization.Account) {
	identity, err := api.store.Identity(r.Context(), account)
	writeResult(r, identity, err)
}

func (api authentication) logout(r *ghttp.Request) {
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

func (api authentication) password(r *ghttp.Request, account organization.Account) {
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

func (api authentication) setCookie(r *ghttp.Request, token string, maxAge int) {
	r.Response.Header().Add("Set-Cookie", (&http.Cookie{
		Name: api.cookieName(r), Value: token, Path: "/", HttpOnly: true,
		Secure: api.secureCookie(r), SameSite: http.SameSiteLaxMode, MaxAge: maxAge,
	}).String())
}

func (api authentication) secureCookie(r *ghttp.Request) bool {
	return api.config.SecureCookies || r.TLS != nil
}

func (api authentication) cookieName(r *ghttp.Request) string {
	if api.secureCookie(r) {
		return "__Host-" + sessionCookie
	}
	return sessionCookie
}

func (api authentication) sessionToken(r *ghttp.Request) string {
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

func (api authentication) cookieToken(r *ghttp.Request) string {
	cookie, err := r.Request.Cookie(api.cookieName(r))
	if err != nil {
		return ""
	}
	return cookie.Value
}
