package api

import (
	"net/http"
	"slices"
	"strings"

	"github.com/JoyElliot/zwOA/internal/modules/chat"
	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/JoyElliot/zwOA/internal/modules/workbench"
	"github.com/JoyElliot/zwOA/internal/platform/httpserver"
	"github.com/gogf/gf/v2/net/ghttp"
)

func New(config httpserver.Config, accounts *organization.Store, messages *chat.Store) *ghttp.Server {
	server := httpserver.New(config)
	server.BindMiddleware("/api/v1/*", allowOrigins(config.AllowedOrigins))
	auth := authentication{store: accounts, config: config}
	bindAuth(server, auth)
	bindOrganization(server, accounts, auth)
	bindChat(server, messages, auth)
	server.Group("/api/v1", func(group *ghttp.RouterGroup) {
		group.Middleware(ghttp.MiddlewareHandlerResponse)
		group.Bind(workbench.Controller{})
	})
	if config.EnableOpenAPI {
		describeOpenAPI(server.GetOpenApi())
	}
	return server
}

func allowOrigins(allowed []string) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		r.Response.Header().Add("Vary", "Origin")
		origin := r.Header.Get("Origin")
		crossOriginAllowed := origin != "" && slices.Contains(allowed, origin)
		if crossOriginAllowed {
			r.Response.Header().Set("Access-Control-Allow-Origin", origin)
			r.Response.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			if crossOriginAllowed {
				r.Response.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
				r.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-ZwOA-Request, X-ZwOA-Revision")
			}
			r.Response.WriteHeader(http.StatusNoContent)
			r.ExitAll()
		}
		if origin != "" && r.Method != http.MethodGet && r.Method != http.MethodHead &&
			!crossOriginAllowed && !sameHostOrigin(origin, r.Host) {
			writeError(r, organization.ErrForbidden)
			r.ExitAll()
		}
		r.Middleware.Next()
	}
}

func sameHostOrigin(origin, host string) bool {
	return strings.EqualFold(origin, "http://"+host) || strings.EqualFold(origin, "https://"+host)
}
