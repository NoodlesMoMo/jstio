package handler

import (
	"git.sogou-inc.com/iweb/jstio/dashboard/service"
	"git.sogou-inc.com/iweb/jstio/model"
	"github.com/qiangxue/fasthttp-routing"
)

func GetClusterHandler(ctx *routing.Context) (err error) {

	appID, err := ctx.QueryArgs().GetUint(`jstio_app_id`)
	if err != nil {
		return service.ErrorResponse(ctx, -1, nil)
	}

	var res model.Resource
	res.AppID = uint(appID)
	res.ResType = `cluster`
	res.GetByAppIDAndType()
	return service.SuccessResponse(ctx, res)
}
func GetAppHandler(ctx *routing.Context) (err error) {
	appName := string(ctx.QueryArgs().Peek(`app_name`))
	cluster := string(ctx.QueryArgs().Peek(`cluster`))
	if appName == "" || cluster == "" {
		return service.ErrorResponse(ctx, -1, nil)
	}

	var app model.Application

	app.AppName = appName
	app.OdinCluster = cluster
	err = app.GetApplicationByAppNameAndCluster()

	return service.SuccessResponse(ctx, app)
}
