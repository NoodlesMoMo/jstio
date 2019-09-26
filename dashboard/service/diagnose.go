package service

import (
	"github.com/k0kubun/pp"
	routing "github.com/qiangxue/fasthttp-routing"
	"jstio/model"
)

func DiagnoseService(ctx *routing.Context) error {

	apps := model.GetApplicationCache().GetActiveApplications()

	_, _ = pp.Println(apps)

	return SuccessResponse(ctx, apps)
}
