package httpserver

import (
	"net/http"
	"time"

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
	return server
}
