package handler

import (
	"git.sogou-inc.com/iweb/jstio/dashboard/service"

	"github.com/qiangxue/fasthttp-routing"
)

var (
	adminSrv = service.NewAdminService()
)

func AdminHandler(ctx *routing.Context) error {
	action := ctx.Param("action")
	res := ctx.Param("res")
	if handler, ok := (*adminSrv)[action]; ok {
		return handler(ctx, res)
	}

	return service.NotFound(ctx)
}
