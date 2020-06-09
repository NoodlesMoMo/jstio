package service

import (
	"git.sogou-inc.com/iweb/jstio/model"
	"github.com/k0kubun/pp"
	routing "github.com/qiangxue/fasthttp-routing"
)

func DiagnoseService(ctx *routing.Context) error {

	apps := model.GetApplicationCache().GetActiveApplications()

	_, _ = pp.Println(apps)

	return SuccessResponse(ctx, apps)
}
