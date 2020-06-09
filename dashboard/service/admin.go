package service

import (
	"encoding/base64"
	"errors"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"git.sogou-inc.com/iweb/jstio/model"

	"github.com/jinzhu/gorm"
	"github.com/qiangxue/fasthttp-routing"
)

const (
	URLencodeContentType = `application/x-www-form-urlencoded` // FIXME
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

	app, _ := model.GetApplicationByID(uint(id))

	return RenderTemplate(ctx, `app_update.html`, app)
}

func (s *AdminService) HTMLAppList(ctx *routing.Context, res string) error {

	apps := model.GetApplicationCache().GetActiveApplications()

	return RenderTemplate(ctx, `app_list.html`, apps)
}

func (s *AdminService) HTMLResAdd(ctx *routing.Context, res string) error {
	id, _ := strconv.ParseUint(res, 10, 64)
	app, err := model.GetApplicationByID(uint(id))
	if err != nil {
		// FIXME:
	}

	return RenderTemplate(ctx, `res_add.html`, app)
}

func (s *AdminService) HTMLResEdit(ctx *routing.Context, res string) error {
	id, _ := strconv.ParseUint(res, 10, 64)
	one, err := model.GetResourceByID(uint(id))
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
	var err error

	defer ErrorLog(`admin rest`, `AddApp`, err)

	app := model.Application{}

	ctx.Request.Header.SetContentType(URLencodeContentType)

	if err = ParamUnpack(ctx, &app); err != nil {
		return ErrorResponse(ctx, -1, "param unpack error")
	}

	protocols, err := s.unpackApplicationProtocol(ctx)
	if err != nil {
		return ErrorResponse(ctx, -4, err)
	}
	protocols.ApplyOwnerRefer(&app)

	params, err := url.ParseQuery(string(ctx.PostBody()))
	if err != nil {
		return ErrorResponse(ctx, -2, err)
	}

	for _, v := range params["upstreams"] {
		id, err := strconv.Atoi(v)
		if err != nil {
			return ErrorResponse(ctx, -3, "invalid upstream")
		}
		if id > 0 {
			app.UpstreamIDs = append(app.UpstreamIDs, uint(id))
		}
	}

	user, ok := ctx.Get(`authority`).(*model.UserData)
	if !ok {
		return ErrorResponse(ctx, -4, "authority failed")
	}

	app.Protocols = protocols
	app.UserID, app.UserName = user.UserID, user.UserName
	if err = app.Add(); err != nil {
		return ErrorResponse(ctx, -5, err)
	}

	return SuccessResponse(ctx, app)
}

func (s *AdminService) AppUpstreams(ctx *routing.Context, res string) error {
	id, _ := strconv.Atoi(res)
	app, _ := model.GetApplicationByID(uint(id))

	return SuccessResponse(ctx, app.Upstream)
}

func (s *AdminService) AppUpdate(ctx *routing.Context, res string) error {
	var err error

	ctx.Request.Header.SetContentType(URLencodeContentType)

	defer ErrorLog(`admin rest`, `AddUpdate`, err)

	appID, _ := strconv.Atoi(res)
	if appID <= 0 {
		return ErrorResponse(ctx, -1, "resource error")
	}

	app := model.Application{}
	if err = ParamUnpack(ctx, &app); err != nil {
		return ErrorResponse(ctx, -1, "param unpack error")
	}

	params, err := url.ParseQuery(string(ctx.PostBody()))
	if err != nil {
		return ErrorResponse(ctx, -2, err)
	}

	app.UpstreamIDs = model.ApplicationXstreams{}
	for _, v := range params["upstreams"] {
		id, err := strconv.Atoi(v)
		if err != nil || id <= 0 {
			return ErrorResponse(ctx, -3, "invalid upstream")
		}

		if id == appID {
			return ErrorResponse(ctx, -3, "invalid upstream: self upstream")
		}

		app.UpstreamIDs.Add(uint(id))
	}

	id, _ := strconv.Atoi(string(ctx.FormValue("id")))
	app.ID = uint(id)

	app.Protocols, err = s.unpackApplicationProtocol(ctx)
	if err != nil {
		return ErrorResponse(ctx, -4, err)
	}
	app.Protocols.ApplyOwnerRefer(&app)

	user, ok := ctx.Get(`authority`).(*model.UserData)
	if !ok {
		return ErrorResponse(ctx, -4, "authority failed")
	}

	app.UserID, app.UserName = user.UserID, user.UserName
	if err = app.Update(); err != nil {
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

//func (s *AdminService) ResAdd(ctx *routing.Context, res string) error {
//	var err error
//
//	defer ErrorLog(`admin rest`, `AddResource`, err)
//
//	ctx.Request.Header.SetContentType(URLencodeContentType)
//
//	jsonConfig := string(ctx.FormValue("json_config"))
//	yamlConfig := string(ctx.FormValue("yaml_config"))
//	resType := string(ctx.FormValue("res_type"))
//	resName := string(ctx.FormValue("name"))
//	AppId, _ := strconv.Atoi(string(ctx.FormValue("app_id")))
//
//	if AppId <= 0 {
//		err = errors.New("invalid appid")
//		return ErrorResponse(ctx, -1, err)
//	}
//
//	jsonData, err := base64.StdEncoding.DecodeString(jsonConfig)
//	if err != nil {
//		return ErrorResponse(ctx, -2, err)
//	}
//
//	yamlData, err := base64.StdEncoding.DecodeString(yamlConfig)
//	if err != nil {
//		return ErrorResponse(ctx, -3, err)
//	}
//
//	if _, err = model.ValidationResource(resType, jsonData); err != nil {
//		return ErrorResponse(ctx, -4, err)
//	}
//
//	resource := model.Resource{
//		AppID:      uint(AppId),
//		Name:       resName,
//		ResType:    resType,
//		Config:     string(jsonData),
//		YamlConfig: string(yamlData),
//	}
//
//	if err = resource.Create(); err != nil {
//		return ErrorResponse(ctx, -5, err)
//	}
//
//	return SuccessResponse(ctx, resource)
//}

func (s *AdminService) ResUpdate(ctx *routing.Context, res string) error {
	var err error
	defer ErrorLog(`admin rest`, `resource update`, err)

	ctx.Request.Header.SetContentType(URLencodeContentType)

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

func (s AdminService) unpackApplicationProtocol(ctx *routing.Context) (model.ApplicationProtocols, error) {
	var err error

	args := ctx.PostArgs()

	protos := args.PeekMulti("protocol")
	appPorts := args.PeekMulti("app_port")
	proxyPorts := args.PeekMulti("proxy_port")

	protocols := model.ApplicationProtocols{}

	validPort := func(port1, port2 int) error {
		if port1 == port2 {
			return errors.New("same port")
		}

		if port1 == 0 || port2 == 0 {
			return errors.New("invalid port")
		}

		if port1 == 9901 || port2 == 9901 {
			return errors.New("9901 port used")
		}

		return nil
	}

	for idx, proto := range protos {
		appPort, _ := strconv.Atoi(string(appPorts[idx]))
		proxyPort, _ := strconv.Atoi(string(proxyPorts[idx]))
		if err = validPort(appPort, proxyPort); err != nil {
			return protocols, err
		}

		protocols = append(protocols, model.ApplicationProtocol{
			Protocol:  string(proto),
			AppPort:   uint32(appPort),
			ProxyPort: uint32(proxyPort),
		})
	}

	sort.Sort(protocols)

	if protocols[0].ProxyPort < 80 {
		return protocols, errors.New("invalid proxy port")
	}

	return protocols, nil
}
