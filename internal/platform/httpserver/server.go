package httpserver

import (
	"net/http"
	"slices"
	"time"

	"github.com/JoyElliot/zwOA/internal/modules/workbench"
	"github.com/gogf/gf/v2/net/ghttp"
)

func New(config Config) *ghttp.Server {
	server := ghttp.GetServer("zwoa-api")
	server.SetAddr(config.Address)
	server.SetDumpRouterMap(false)
	server.SetReadTimeout(10 * time.Second)
	server.SetWriteTimeout(15 * time.Second)
	server.SetIdleTimeout(60 * time.Second)
	server.SetClientMaxBodySize(1 << 20)

	if config.EnableOpenAPI {
		server.SetOpenApiPath("/openapi.json")
	}

	server.BindHandler("GET:/healthz", func(r *ghttp.Request) {
		r.Response.WriteJson(map[string]string{"status": "ok"})
	})
	server.BindMiddleware("/api/v1/*", allowOrigins(config.AllowedOrigins))
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
		if origin != "" && slices.Contains(allowed, origin) {
			r.Response.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == http.MethodOptions {
				r.Response.Header().Set("Access-Control-Allow-Methods", http.MethodGet)
				r.Response.WriteHeader(http.StatusNoContent)
				r.ExitAll()
			}
		}
		r.Middleware.Next()
	}
}
