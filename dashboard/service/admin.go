package service

import (
	"encoding/base64"
	"errors"
	"jstio/model"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"

	"github.com/qiangxue/fasthttp-routing"
)

type AdminService map[string]func(ctx *routing.Context, res string) error

func (s *AdminService) initialization() {
	t := reflect.TypeOf(s)
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)

		fn, ok := method.Func.Interface().(func(*AdminService, *routing.Context, string) error)
		if !ok {
			continue
		}

		(*s)[strings.ToLower(pathFormat(method.Name))] = func(ctx *routing.Context, res string) error {
			return fn(s, ctx, res)
		}
	}
}

func NewAdminService() *AdminService {
	inst := &AdminService{}
	inst.initialization()
	return inst
}

func (s *AdminService) Index(ctx *routing.Context, res string) error {
	return RenderTemplate(ctx, `index.html`, nil)
}

func (s *AdminService) HTMLAppAdd(ctx *routing.Context, res string) error {
	return RenderTemplate(ctx, `app_add.html`, nil)
}

func (s *AdminService) HTMLAppUpdate(ctx *routing.Context, res string) error {

	id, _ := strconv.Atoi(res)

	app, _ := model.GetApplicationById(uint(id))

	return RenderTemplate(ctx, `app_update.html`, app)
}

func (s *AdminService) HTMLAppList(ctx *routing.Context, res string) error {
	//apps, _ := model.CompleteActiveApps()
	apps := model.GetApplicationCache().GetActiveApplications()

	return RenderTemplate(ctx, `app_list.html`, apps)
}

func (s *AdminService) HTMLResAdd(ctx *routing.Context, res string) error {
	id, _ := strconv.ParseUint(res, 10, 64)
	app, err := model.GetApplicationById(uint(id))
	if err != nil {
		// FIXME:
	}

	return RenderTemplate(ctx, `res_add.html`, app)
}

func (s *AdminService) HTMLResEdit(ctx *routing.Context, res string) error {
	id, _ := strconv.ParseUint(res, 10, 64)
	one, err := model.GetResourceById(uint(id))
	if err != nil {
		// FIXME:
	}

	return RenderTemplate(ctx, `res_edit.html`, one)
}

func (s *AdminService) HTMLResList(ctx *routing.Context, res string) error {

	apps, _ := model.AllApps(true)

	return RenderTemplate(ctx, `res_list.html`, apps)
}

func (s *AdminService) HTMLHistory(ctx *routing.Context, res string) error {

	page, _ := strconv.Atoi(string(ctx.FormValue("page")))
	size, _ := strconv.Atoi(string(ctx.FormValue("size")))

	_, items := model.GetHistoryRecord(page, size)

	return RenderTemplate(ctx, `history.html`, items)
}

func (s *AdminService) HTMLHelp(ctx *routing.Context, res string) error {
	return RenderTemplate(ctx, `help.html`, nil)
}

func (s *AdminService) HTMLTopology(ctx *routing.Context, res string) error {
	return RenderTemplate(ctx, `topology.html`, nil)
}

func (s *AdminService) AppAdd(ctx *routing.Context, res string) error {
	var (
		err      error
		upstream = []uint{}
	)

	defer ErrorLog(`admin rest`, `AddApp`, err)

	app := model.Application{}

	ctx.Request.Header.SetContentType("application/x-www-form-urlencoded")

	if err = ParamUnpack(ctx, &app); err != nil {
		return ErrorResponse(ctx, -1, "param unpack error")
	}

	params, err := url.ParseQuery(string(ctx.PostBody()))
	if err != nil {
		return ErrorResponse(ctx, -2, err)
	}

	for _, v := range params["upstreams"] {
		id, err := strconv.Atoi(v)
		if err != nil {
			return ErrorResponse(ctx, -3, "invalid upstream")
		}
		upstream = append(upstream, uint(id))
	}

	user, ok := ctx.Get(`authority`).(*model.UserData)
	if !ok {
		return ErrorResponse(ctx, -4, "authority failed")
	}

	app.UserId, app.UserName = user.UserId, user.UserName
	if err = app.Add(upstream, model.ReferKindUpstream); err != nil {
		return ErrorResponse(ctx, -5, "create error")
	}

	return SuccessResponse(ctx, app)
}

func (s *AdminService) AppUpstreams(ctx *routing.Context, res string) error {
	id, _ := strconv.Atoi(res)
	app, _ := model.GetApplicationById(uint(id))

	return SuccessResponse(ctx, app.Upstream)
}

func (s *AdminService) AppUpdate(ctx *routing.Context, res string) error {
	var (
		err      error
		upstream = []uint{}
	)
	ctx.Request.Header.SetContentType("application/x-www-form-urlencoded")

	defer ErrorLog(`admin rest`, `AddUpdate`, err)

	app := model.Application{}
	if err = ParamUnpack(ctx, &app); err != nil {
		return ErrorResponse(ctx, -1, "param unpack error")
	}

	params, err := url.ParseQuery(string(ctx.PostBody()))
	if err != nil {
		return ErrorResponse(ctx, -2, err)
	}

	for _, v := range params["upstreams"] {
		id, err := strconv.Atoi(v)
		if err != nil {
			return ErrorResponse(ctx, -3, "invalid upstream")
		}
		upstream = append(upstream, uint(id))
	}

	id, _ := strconv.Atoi(string(ctx.FormValue("id")))
	app.ID = uint(id)

	user, ok := ctx.Get(`authority`).(*model.UserData)
	if !ok {
		return ErrorResponse(ctx, -4, "authority failed")
	}

	app.UserId, app.UserName = user.UserId, user.UserName
	if err = app.Update(upstream, model.ReferKindUpstream); err != nil {
		return ErrorResponse(ctx, -5, "update error")
	}

	return SuccessResponse(ctx, app)
}

func (s *AdminService) AppList(ctx *routing.Context, res string) error {
	var err error

	defer ErrorLog(`admin rest`, `AppList`, err)

	apps, err := model.AllApps(true)
	if err != nil {
		return ErrorResponse(ctx, -2, err)
	}

	return SuccessResponse(ctx, apps)
}

func (s *AdminService) ResAdd(ctx *routing.Context, res string) error {
	var err error

	defer ErrorLog(`admin rest`, `AddResource`, err)

	ctx.Request.Header.SetContentType("application/x-www-form-urlencoded")

	jsonConfig := string(ctx.FormValue("json_config"))
	yamlConfig := string(ctx.FormValue("yaml_config"))
	resType := string(ctx.FormValue("res_type"))
	resName := string(ctx.FormValue("name"))
	AppId, _ := strconv.Atoi(string(ctx.FormValue("app_id")))

	if AppId <= 0 {
		err = errors.New("invalid appid")
		return ErrorResponse(ctx, -1, err)
	}

	jsonData, err := base64.StdEncoding.DecodeString(jsonConfig)
	if err != nil {
		return ErrorResponse(ctx, -2, err)
	}

	yamlData, err := base64.StdEncoding.DecodeString(yamlConfig)
	if err != nil {
		return ErrorResponse(ctx, -3, err)
	}

	if _, err = model.ValidationResource(resType, jsonData); err != nil {
		return ErrorResponse(ctx, -4, err)
	}

	resource := model.Resource{
		AppID:      uint(AppId),
		Name:       resName,
		ResType:    resType,
		Config:     string(jsonData),
		YamlConfig: string(yamlData),
	}

	if err = resource.Create(); err != nil {
		return ErrorResponse(ctx, -5, err)
	}

	return SuccessResponse(ctx, resource)
}

func (s *AdminService) ResUpdate(ctx *routing.Context, res string) error {
	var err error
	defer ErrorLog(`admin rest`, `resource update`, err)

	ctx.Request.Header.SetContentType("application/x-www-form-urlencoded")

	id, _ := strconv.Atoi(string(ctx.FormValue("id")))
	appId, _ := strconv.Atoi(string(ctx.FormValue("app_id")))
	jsonConfig := string(ctx.FormValue("json_config"))
	yamlConfig := string(ctx.FormValue("yaml_config"))
	resName := string(ctx.FormValue("name"))
	resType := string(ctx.FormValue("res_type"))

	jsonData, err := base64.StdEncoding.DecodeString(jsonConfig)
	if err != nil {
		return ErrorResponse(ctx, -1, err)
	}

	yamlData, err := base64.StdEncoding.DecodeString(yamlConfig)
	if err != nil {
		return ErrorResponse(ctx, -2, err)
	}

	if _, err = model.ValidationResource(resType, jsonData); err != nil {
		return ErrorResponse(ctx, -3, err)
	}

	resource := model.Resource{
		Model:      gorm.Model{ID: uint(id)},
		AppID:      uint(appId),
		Name:       resName,
		ResType:    resType,
		Config:     string(jsonData),
		YamlConfig: string(yamlData),
	}

	if err = resource.Update(true); err != nil {
		return ErrorResponse(ctx, -4, err)
	}

	return SuccessResponse(ctx, resource)
}

func (s AdminService) ResValidation(ctx *routing.Context, res string) error {
	var err error

	defer ErrorLog(`admin rest`, `ValidationResource`, err)

	cfg, err := base64.StdEncoding.DecodeString(string(ctx.PostBody()))
	if err != nil {
		return ErrorResponse(ctx, -1, err)
	}

	if _, err = model.ValidationResource(res, cfg); err != nil {
		return ErrorResponse(ctx, -2, err)
	}

	return SuccessResponse(ctx, nil)
}
