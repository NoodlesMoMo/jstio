package dashboard

import (
	"github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
	"jstio/dashboard/handler"
)

var (
	router_ = routing.New()
)

func HandleDashBoardRequest(ctx *fasthttp.RequestCtx) {
	router_.HandleRequest(ctx)
}

func init() {

	router_.Use(handler.ExceptionDump)

	adminGroup := router_.Group("/admin")
	adminGroup.Use(
		handler.Auth,
		handler.AccessLog,
	)
	adminGroup.Any("/view/<action>/<res>", handler.AdminHandler)
	adminGroup.Any("/data/<action>/<res>", handler.AdminHandler)
	adminGroup.Any("/diag", handler.DiagnoseHandler)

	pprofGroup := router_.Group("/debug/pprof")
	pprofGroup.To("GET,POST", "/cmdline", handler.CommandLine)
	pprofGroup.To("GET,POST", "/profile", handler.Profile)
	pprofGroup.To("GET,POST", "/symbol", handler.Symbol)
	pprofGroup.To("GET,POST", "/trace", handler.TraceX)
	pprofGroup.To("GET,POST", "/<name>", handler.ProfileIndex)

	router_.Any("/static/*", handler.StaticHandler)
}
