package dashboard

import (
	"git.sogou-inc.com/iweb/jstio/dashboard/handler"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
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
	adminGroup.Any("/diag/apps", handler.DiagnoseAppsHandler)
	adminGroup.Any("/diag/waits", handler.DiagnosePodWaitHandler)
	adminGroup.Any("/healthy", handler.HealthyHandler)

	adminGroup.Any("/route/get", handler.GetRoutes)
	adminGroup.Any("/route/update", handler.UpdateRoutes)
	adminGroup.Any("/cluster/list", handler.GetClusterList)
	adminGroup.Get("/topology", handler.ApplicationsTopologyHandler)

	router_.Any("/api/v1/cluster/get", handler.GetClusterHandler)
	router_.Any("/api/v1/app/get", handler.GetAppHandler)

	pprofGroup := router_.Group("/debug/pprof")
	pprofGroup.To("GET,POST", "/cmdline", handler.CommandLine)
	pprofGroup.To("GET,POST", "/profile", handler.Profile)
	pprofGroup.To("GET,POST", "/symbol", handler.Symbol)
	pprofGroup.To("GET,POST", "/trace", handler.TraceX)
	pprofGroup.To("GET,POST", "/<name>", handler.ProfileIndex)

	router_.Any("/static/*", handler.StaticHandler)
}
