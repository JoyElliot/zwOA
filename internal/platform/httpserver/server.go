package httpserver

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/JoyElliot/zwOA/internal/modules/workbench"
	"github.com/gogf/gf/v2/net/ghttp"
)

func New(config Config, store *organization.Store) *ghttp.Server {
	server := ghttp.GetServer("zwoa-api")
	server.SetAddr(config.Address)
	server.SetDumpRouterMap(false)
	server.SetReadTimeout(10 * time.Second)
	server.SetWriteTimeout(15 * time.Second)
	server.SetIdleTimeout(60 * time.Second)
	server.SetClientMaxBodySize(1 << 20)
	server.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-Url-Path")
		server.ServeHTTP(w, r)
	})

	if config.EnableOpenAPI {
		server.SetOpenApiPath("/openapi.json")
	}

	server.BindHandler("GET:/healthz", func(r *ghttp.Request) {
		r.Response.WriteJson(map[string]string{"status": "ok"})
	})
	server.BindMiddleware("/api/v1/*", allowOrigins(config.AllowedOrigins))
	bindOrganization(server, store, config)
	bindChat(server, store, config)
	server.Group("/api/v1", func(group *ghttp.RouterGroup) {
		group.Middleware(ghttp.MiddlewareHandlerResponse)
		group.Bind(workbench.Controller{})
	})
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
