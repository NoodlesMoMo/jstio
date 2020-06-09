package handler

import (
	"encoding/json"
	"git.sogou-inc.com/iweb/jstio/dashboard/service"
	routing "github.com/qiangxue/fasthttp-routing"
)

func ApplicationsTopologyHandler(ctx *routing.Context) error {
	ctx.Serialize = func(data interface{}) (bytes []byte, e error) {
		return json.Marshal(data)
	}

	from := string(ctx.QueryArgs().Peek("from"))
	to := string(ctx.QueryArgs().Peek("to"))

	network, err := service.GenApplicationVisNetwork(from, to)
	if err != nil {
		return err
	}
	_ = ctx.WriteData(network)
	return nil
}
