package handler

import (
	"jstio/dashboard/service"

	"github.com/qiangxue/fasthttp-routing"
)

var (
	adminSrv = service.NewAdminService()
)

func AdminHandler(ctx *routing.Context) error {
	action := string(ctx.Param("action"))
	res := string(ctx.Param("res"))
	if handler, ok := (*adminSrv)[action]; ok {
		return handler(ctx, res)
	}

	return service.NotFound(ctx)
}
