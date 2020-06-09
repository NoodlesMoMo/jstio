package model

import (
	"bufio"
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	eev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"math/rand"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"git.sogou-inc.com/iweb/jstio/internel/logs"
	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/jsonpb"
	"github.com/mohae/deepcopy"

	"github.com/jinzhu/gorm"
)

var (
	_appCacheOnce = sync.Once{}
	_appCache     *ApplicationCache
)

func MigrateAppsTables() {
	GetDBInstance().AutoMigrate(Application{})
}

type ApplicationCache struct {
	lock  sync.RWMutex
	cache map[string]*Application
}

// Warning: don't support cycle pointer in application's Xstreams
func (ac *ApplicationCache) GetActiveApplications() map[string]*Application {
	ac.lock.RLock()
	defer ac.lock.RUnlock()

	return deepcopy.Copy(ac.cache).(map[string]*Application)
}

func (ac *ApplicationCache) cycleAutoReBuild() {
	tagLog := logs.TaggedLoggerFactory("cycle auto refresh")

	rand.Seed(time.Now().UnixNano())

	for {
		r := rand.Intn(10) + 300 // about 5 minute
		tagLog("").Printf("next schedule will be after %d seconds.\n", r)
		time.Sleep(time.Duration(r) * time.Second)

		ac.ReBuild()
	}
}

func (ac *ApplicationCache) ReBuild() {

	tagLog := logs.FuncTaggedLoggerFactory()

	_cache, err := CompleteActiveApps()
	if err != nil {
		tagLog("complement apps").WithError(err).Errorln("get active applications error")
		return
	}

	ac.lock.Lock()
	ac.cache = _cache
	ac.lock.Unlock()

	tagLog("complement apps").Println("update application cache success")
}

func GetApplicationCache() *ApplicationCache {
	if _appCache != nil {
		return _appCache
	}

	tagLog := logs.FuncTaggedLoggerFactory()

	_appCacheOnce.Do(func() {
		_appCache = &ApplicationCache{
			lock: sync.RWMutex{},
		}
		var err error
		_appCache.cache, err = CompleteActiveApps()
		if err != nil {
			tagLog("create application cache").Errorln("error:", err)
			panic(err)
		}
		go _appCache.cycleAutoReBuild()
	})

	return _appCache
}

type ApplicationProtocols []ApplicationProtocol

type ApplicationProtocol struct {
	Protocol  string `json:"protocol"`
	Domain    string `json:"domain"`
	AppPort   uint32 `json:"app_port"`
	ProxyPort uint32 `json:"proxy_port"`

	OwnerRef *Application `json:"-"`
}

func (aps ApplicationProtocols) Value() (driver.Value, error) {
	return json.Marshal(aps)
}

func (aps *ApplicationProtocols) Scan(v interface{}) error {
	var err error
	switch t := v.(type) {
	case string:
		err = json.Unmarshal([]byte(t), aps)
	case []byte:
		err = json.Unmarshal(t, aps)
	default:
		err = fmt.Errorf("application-protocols: unsupport scan type %T", v)
	}

	return err
}

func (aps ApplicationProtocols) Len() int {
	return len(aps)
}

func (aps ApplicationProtocols) Less(i, j int) bool {
	return aps[i].ProxyPort < aps[j].ProxyPort
}

func (aps ApplicationProtocols) Swap(i, j int) {
	aps[i], aps[j] = aps[j], aps[i]
}

func (aps *ApplicationProtocols) Pop(domain string) *ApplicationProtocol {
	for idx, ap := range *aps {
		if ap.Domain == domain {
			*aps = append((*aps)[:idx], (*aps)[idx+1:]...)
			return &ap
		}
	}
	return nil
}

func (aps *ApplicationProtocols) ApplyOwnerRefer(app *Application) {
	for idx, proto := range *aps {
		(*aps)[idx].OwnerRef = app
		if proto.Domain == "" {
			if proto.Protocol == ProtocolHTTP {
				(*aps)[idx].Domain = app.Domain()
				continue
			}
			(*aps)[idx].Domain = strings.Join([]string{
				app.AppName,
				proto.Protocol,
				app.OdinCluster,
				SOUGOSLD,
			}, ".")
		}
	}
}

func (aps *ApplicationProtocols) ResetOwnerRefer(app *Application) {
	for idx := range *aps {
		(*aps)[idx].OwnerRef = app
	}
}

type ApplicationXstreams []uint

func (ax ApplicationXstreams) IDs() []uint {
	return ax
}

func (ax ApplicationXstreams) Value() (driver.Value, error) {
	return json.Marshal(ax)
}

func (ax *ApplicationXstreams) Scan(v interface{}) error {
	var err error
	switch t := v.(type) {
	case string:
		err = json.Unmarshal([]byte(t), ax)
	case []byte:
		err = json.Unmarshal(t, ax)
	default:
		err = fmt.Errorf("application-protocols: unsupport scan type %T", v)
	}
	return err
}

func (ax *ApplicationXstreams) Add(id uint) bool {
	if id == 0 {
		return false
	}

	pos := 0
	for idx, one := range *ax {
		if one == id {
			return false
		} else if one > id {
			pos = idx
			break
		} else {
			pos++
		}
	}

	*ax = append(*ax, 0)
	copy((*ax)[pos+1:], (*ax)[pos:])
	(*ax)[pos] = id

	return true
}

func (ax *ApplicationXstreams) Remove(id uint) bool {
	if id == 0 {
		return false
	}
	for idx, one := range *ax {
		if one == id {
			*ax = append((*ax)[:idx], (*ax)[idx+1:]...)
			return true
		}
	}
	return false
}

func (ax *ApplicationXstreams) Diff(ox *ApplicationXstreams) ([]uint, []uint) {
	as, os := make(map[uint]bool, 0), make(map[uint]bool, 0)
	for _, id := range *ax {
		as[id] = false
	}
	for _, id := range *ox {
		os[id] = false
	}

	for id, _ := range as {
		if _, ok := os[id]; ok {
			delete(as, id)
			delete(os, id)
		}
	}

	decrease, increase := ApplicationXstreams{}, ApplicationXstreams{}
	for id, _ := range as {
		increase.Add(id)
	}
	for id, _ := range os {
		decrease.Add(id)
	}

	return decrease.IDs(), increase.IDs()
}

//func (ax *ApplicationXstreams) Diff(ox *ApplicationXstreams) []uint {
//	axLen, oxLen := len(*ax), len(*ox)
//	if axLen == 0 {
//		return *ox
//	}
//	if oxLen == 0 {
//		return nil
//	}
//
//	set := make([]uint, 0)
//	oIdx := 0
//	for i := 0; i < axLen && oIdx < oxLen; {
//		oid := (*ox)[oIdx]
//		aid := (*ax)[i]
//		if aid < oid {
//			i++
//			continue
//		} else if aid == oid {
//			i++
//			oIdx++
//			continue
//		} else {
//			set = append(set, oid)
//			oIdx++
//		}
//	}
//
//	if oIdx < oxLen {
//		set = append(set, (*ox)[oIdx:]...)
//	}
//
//	return set
//}

func (ax *ApplicationXstreams) Union(ox *ApplicationXstreams) []uint {
	buf := make(map[uint]struct{})
	for _, id := range *ax {
		buf[id] = struct{}{}
	}
	for _, id := range *ox {
		buf[id] = struct{}{}
	}

	set := make([]uint, 0)
	for k := range buf {
		set = append(set, k)
	}

	tmp := (ApplicationXstreams)(set)
	sort.Sort(tmp)

	return tmp
}

func (ax *ApplicationXstreams) Merge(ox *ApplicationXstreams) []uint {
	*ax = ax.Union(ox)
	return *ax
}

func (ax ApplicationXstreams) Len() int {
	return len(ax)
}

func (ax ApplicationXstreams) Less(i, j int) bool {
	return ax[i] < ax[j]
}

func (ax ApplicationXstreams) Swap(i, j int) {
	ax[i], ax[j] = ax[j], ax[i]
}

type Application struct {
	gorm.Model

	AppName       string               `gorm:"not null; unique_index:idx_cluster;" json:"app_name"`
	OdinCluster   string               `gorm:"not null; unique_index:idx_cluster;" json:"odin_cluster"`
	Namespace     string               `gorm:"not null; unique_index:idx_cluster;" json:"namespace"`
	Protocols     ApplicationProtocols `gorm:"not null; type:varchar(255)" json:"protocols"`
	UpstreamIDs   ApplicationXstreams  `gorm:"not null; type:varchar(255)" json:"upstream_ids"`
	DownstreamIDs ApplicationXstreams  `gorm:"not null; type:varchar(255)" json:"downstream_ids"`
	Version       string               `gorm:"not null; default:'v1';" json:"version"`    // FIXME:
	Status        string               `gorm:"not null; default:'pending'" json:"status"` // FIXME:
	UserID        string               `gorm:"not null;" json:"user_id"`                  // FIXME
	UserName      string               `gorm:"not null;" json:"user_name"`                // FIXME
	Description   string               `gorm:"not null; default:''" json:"description"`   // FIXME

	Upstream   []*Application `gorm:"-"`
	Downstream []*Application `gorm:"-"`

	Resources []Resource `gorm:"ForeignKey:AppID"`

	HasCompleted bool `gorm:"-"`
}

func (a *Application) SelectorFormat() string {
	return defaultRegistryPrefix + path.Join(
		strings.Trim(a.OdinCluster, "/"),
		strings.Trim(a.Namespace, "/"),
		strings.Trim(a.AppName, "/"),
		defaultRegistrySuffix,
	)
}

func (a *Application) SelectorScan(key string) error {
	seps := strings.Split(strings.Trim(key, "/"), "/")
	if len(seps) < 6 {
		return errors.New("invalid Jstio path format")
	}

	a.OdinCluster = seps[1]
	a.Namespace = seps[2]
	a.AppName = seps[3]

	return nil
}

func (a *Application) Hash() string {
	return a.Domain()
}

func (a *Application) Domain() string {
	return strings.Join([]string{
		a.AppName,
		a.OdinCluster,
		SOUGOSLD,
	}, ".")
}

func (a *Application) BeforeUpdate(tx *gorm.DB) error {
	sort.Sort(a.UpstreamIDs)
	sort.Sort(a.DownstreamIDs)
	return nil
}

func (a *Application) BeforeCreate(tx *gorm.DB) error {
	sort.Sort(a.UpstreamIDs)
	sort.Sort(a.DownstreamIDs)
	return nil
}

func (a *Application) AfterUpdate(tx *gorm.DB) error {
	var err error

	defer func() {
		if err == nil {
			RecordSuccess(a.UserID, a.UserName, `application`, OperateUpdate, a.ID)
		} else {
			RecordFailure(a.UserID, a.UserName, `application`, OperateUpdate, a.ID)
		}
	}()

	return err
}

func (a *Application) AfterCreate(tx *gorm.DB) error {
	var err error

	err = a.UpdateReference(tx)

	RecordSuccess(a.UserID, a.UserName, `application`, OperateCreate, a.ID)

	return err
}

func (a *Application) AfterDelete() error {
	var err error

	defer func() {
		if err == nil {
			RecordSuccess(a.UserID, a.UserName, `application`, OperateCreate, a.ID)
		} else {
			RecordFailure(a.UserID, a.UserName, `application`, OperateCreate, a.ID)
		}
	}()

	GetApplicationCache().ReBuild()

	err = GetEventNotifier().Push(EventApplicationDelete, a)

	return err
}

func (a *Application) loadApplicationXstream() error {
	if a.HasCompleted {
		return nil
	}

	db := GetDBInstance()

	if a.UpstreamIDs.Len() > 0 {
		if e := db.Where(a.UpstreamIDs.IDs()).Find(&a.Upstream).Error; e != nil {
			return e
		}
	}

	if a.DownstreamIDs.Len() > 0 {
		if e := db.Where(a.DownstreamIDs.IDs()).Find(&a.Downstream).Error; e != nil {
			return e
		}
	}

	a.HasCompleted = true

	return nil
}

func (a *Application) Add() error {
	var err error
	tagLog := logs.FuncTaggedLoggerFactory()

	tx := GetDBInstance().Model(a).Begin()

	defer func() {
		if err == nil {
			tx.Commit()
			GetApplicationCache().ReBuild()
			if pa, ok := GetApplicationCache().GetActiveApplications()[a.Hash()]; !ok {
				err = errors.New("cache app error")
			} else {
				err = GetEventNotifier().Push(EventApplicationCreate, pa)
			}
		} else {
			tx.Rollback()
		}

		if err == nil {
			tagLog(a.Hash()).Println("create success")
		} else {
			tagLog(a.Hash()).WithError(err).Errorln("create failed")
		}
	}()

	if err = a.loadApplicationXstream(); err != nil {
		return err
	}

	if err = a.AddDefaultResources(); err != nil {
		return err
	}

	err = tx.Create(a).Error

	return err
}

func (a *Application) UpdateReference(tx *gorm.DB) error {
	var err error

	if a.UpstreamIDs.Len() == 0 {
		return nil
	}

	var upstreams []Application

	if err = tx.Where(a.UpstreamIDs.IDs()).Find(&upstreams).Error; err != nil {
		return err
	}

	for _, upstream := range upstreams {
		if upstream.DownstreamIDs.Add(a.ID) {
			if err = tx.Model(upstream).UpdateColumn(`downstream_ids`, upstream.DownstreamIDs).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *Application) AddDefaultResources() error {

	resRender := GetResourceRender()

	for _, resType := range [...]ResourceType{
		ResourceTypeRoute,
		ResourceTypeListener,
		ResourceTypeEndpoint,
		ResourceTypeCluster,
	} {
		yamlCfg, err := resRender.Render(resType, a)
		if err != nil {
			return fmt.Errorf("render %d error: %s", a.ID, err.Error())
		}

		jsonCfg, err := yaml.YAMLToJSON(yamlCfg)
		if err != nil {
			return fmt.Errorf("yaml to json, appID: %d, resType: %s, error: %s", a.ID, resType, err.Error())
		}
		res := Resource{
			AppID:      a.ID,
			Name:       a.Hash() + "_" + resType,
			ResType:    resType,
			Config:     string(jsonCfg),
			YamlConfig: string(yamlCfg),
		}
		a.Resources = append(a.Resources, res)
	}

	return nil
}

func (a *Application) Update() error {
	var err error

	tagLog := logs.FuncTaggedLoggerFactory()

	tx := GetDBInstance().Model(a).Preload("Resources").Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
			tagLog(a.Hash()).WithError(err).Errorln("update failed")
		} else {
			tagLog(a.Hash()).Println("update success")
		}
	}()

	if err = a.repairApplicationRelations(tx); err != nil {
		tagLog("repair app relations").WithError(err).Error("app_id:", a.ID)
		return err
	}

	if err = tx.Update(a).Error; err != nil {
		tagLog("app update").WithError(err).Errorln("app_id:", a.ID)
		return err
	}

	if err = tx.Commit().Error; err != nil {
		return err
	}

	GetApplicationCache().ReBuild()

	return GetEventNotifier().Push(EventApplicationUpdate, a)
}

func GetApplicationByID(id uint) (Application, error) {
	var err error

	app := Application{Model: gorm.Model{ID: id}}

	err = GetDBInstance().Preload("Resources").First(&app).Error
	if err != nil {
		return app, err
	}

	err = app.loadApplicationXstream()

	return app, err
}

func GetSingleAppByName(name string, cluster string) (Application, error) {
	app := Application{}
	err := GetDBInstance().Preload("Resources").Where("app_name=? and odin_cluster=?", name, cluster).Find(&app).Error

	return app, err
}
func (app *Application) GetApplicationByAppNameAndCluster() (err error) {

	err = GetDBInstance().Where("app_name = ? and odin_cluster = ?", app.AppName, app.OdinCluster).First(&app).Error

	return
}

func AllApps(onlyValid bool) ([]Application, error) {
	var err error

	result := make([]Application, 0)

	db := GetDBInstance().Preload("Resources")

	if onlyValid {
		err = db.Find(&result).Error
	} else {
		err = db.Unscoped().Find(&result).Error
	}

	return result, err
}

func QueryApplications(page, size int) ([]Application, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	apps := make([]Application, 0)

	count := 0
	db := GetDBInstance().Preload("Resources")
	err := db.Model(&Application{}).Count(&count).Error
	if err != nil {
		return apps, err
	}

	err = db.Offset((page - 1) * size).Limit(size).Find(&apps).Error

	return apps, err
}

func CompleteActiveApps() (map[string]*Application, error) {
	var (
		err        error
		activeApps = make(map[string]*Application)
	)

	appList, err := AllApps(true)
	if err != nil {
		return activeApps, err
	}

	for _, app := range appList {
		app := app
		if e := app.loadApplicationXstream(); e != nil {
			err = e
		}
		activeApps[app.Hash()] = &app
	}

	return activeApps, err
}

func (a *Application) repairApplicationRelations(tx *gorm.DB) error {
	var (
		err           error
		increaseVRess = make(map[string][]cache.Resource)
	)

	old, err := GetApplicationByID(a.ID)
	if err != nil {
		return err
	}

	if reflect.DeepEqual(old.UpstreamIDs, a.UpstreamIDs) && reflect.DeepEqual(old.Protocols, a.Protocols) {
		return nil
	}

	incrProtos, updateProtos := ApplicationProtocols{}, ApplicationProtocols{}
	for _, proto := range a.Protocols {
		if oldProto := old.Protocols.Pop(proto.Domain); oldProto == nil {
			incrProtos = append(incrProtos, proto)
		} else if oldProto.ProxyPort != proto.ProxyPort || oldProto.AppPort != proto.AppPort {
			updateProtos = append(updateProtos, proto)
		}
	}
	decrProtos := old.Protocols

	if incrProtos.Len() > 0 {
		a.Protocols.ResetOwnerRefer(nil) // 避免栈溢出
		dummyApp := deepcopy.Copy(a).(*Application)
		dummyApp.Protocols = incrProtos
		if err = dummyApp.loadApplicationXstream(); err != nil {
			return err
		}
		for idx, res := range old.Resources {
			var yb, jb []byte
			if yb, err = GetResourceRender().Render(res.ResType, dummyApp); err != nil {
				return err
			}
			old.Resources[idx].YamlConfig += *(*string)(unsafe.Pointer(&yb))
			if jb, err = yaml.YAMLToJSON([]byte(old.Resources[idx].YamlConfig)); err != nil {
				return err
			}
			old.Resources[idx].Config = *(*string)(unsafe.Pointer(&jb))
		}
	}
	// repair protos
	decreaseIDs, increaseIDs := a.UpstreamIDs.Diff(&old.UpstreamIDs)
	decreaseAppSet := make(map[string]struct{}, 0)

	// repair xstream
	for _, streamID := range decreaseIDs {
		if streamID == 0 {
			continue
		}
		app := Application{}
		if err = tx.First(&app, streamID).Error; err != nil {
			return err
		}
		if app.DownstreamIDs.Remove(a.ID) {
			if err = tx.Model(app).UpdateColumn("downstream_ids", app.DownstreamIDs).Error; err != nil {
				return err
			}
		}
		for _, proto := range app.Protocols {
			decreaseAppSet[proto.Domain] = Zero
		}
	}

	for _, streamID := range increaseIDs {
		if streamID == 0 {
			continue
		}
		app := Application{}
		if err = tx.First(&app, streamID).Error; err != nil {
			return err
		}
		if app.DownstreamIDs.Add(a.ID) {
			if err = tx.Model(app).UpdateColumn("downstream_ids", app.DownstreamIDs).Error; err != nil {
				return err
			}
		}
	}
	if len(increaseIDs) > 0 {
		a.Protocols.ResetOwnerRefer(nil) // 避免栈溢出
		dummyApp := deepcopy.Copy(a).(*Application)
		dummyApp.UpstreamIDs = increaseIDs
		if err = dummyApp.loadApplicationXstream(); err != nil {
			return err
		}
		for _, resType := range [...]ResourceType{
			ResourceTypeRoute,
			ResourceTypeEndpoint,
			ResourceTypeCluster,
		} {
			var yb, jb []byte
			if yb, err = GetResourceRender().RenderDelta(resType, dummyApp); err != nil {
				return err
			}
			if jb, err = yaml.YAMLToJSON(yb); err != nil {
				return err
			}
			vrss, err := ValidationResource(resType, jb)
			if err != nil {
				return err
			}
			if resType == ResourceTypeRoute {
				for _, vrs := range vrss {
					router := vrs.(*v2.RouteConfiguration)
					for range router.VirtualHosts {
						// self at first one in template, remove it!
						router.VirtualHosts = append(router.VirtualHosts[:0], router.VirtualHosts[1:]...)
						break
					}
				}
			}
			increaseVRess[resType] = vrss
		}
	}

outer:
	for resIdx, res := range old.Resources {
		crss, err := ValidationResource(res.ResType, *(*[]byte)(unsafe.Pointer(&res.Config)))
		if err != nil {
			return err
		}
		// merge
		if res.ResType == ResourceTypeEndpoint || res.ResType == ResourceTypeCluster {
			if vrs, ok := increaseVRess[res.ResType]; ok {
				crss = append(crss, vrs...)
			}
		}

		okCacheRess := crss[:0]
	inner:
		for _, crs := range crss {
			switch res.ResType {
			case ResourceTypeRoute:
				router := crs.(*v2.RouteConfiguration)
				// delete old protocols
				for _, delProto := range decrProtos {
					if router.Name == delProto.Domain {
						continue inner
					}
				}
				// merge new upstream
				if vRess, ok := increaseVRess[res.ResType]; ok {
					for _, vRes := range vRess {
						vHosts := vRes.(*v2.RouteConfiguration)
						if vHosts.Name == router.Name {
							router.VirtualHosts = append(router.VirtualHosts, vHosts.VirtualHosts...)
							break
						}
					}
				}
				// delete old
				rv := router.VirtualHosts[:0]
				for _, vHost := range router.VirtualHosts {
					if _, ok := decreaseAppSet[vHost.Name]; !ok {
						rv = append(rv, vHost)
					}
				}
				router.VirtualHosts = rv
			case ResourceTypeEndpoint:
				endpoint := crs.(*v2.ClusterLoadAssignment)
				if _, ok := decreaseAppSet[endpoint.ClusterName]; ok {
					continue inner
				}
				// 更新端口
				for _, updateProto := range updateProtos {
					if endpoint.ClusterName == updateProto.Domain {
						for i, ee := range endpoint.Endpoints {
							if len(ee.LbEndpoints) == 0 {
								break
							}
							eeHost, ok := ee.LbEndpoints[0].HostIdentifier.(*eev2.LbEndpoint_Endpoint)
							if !ok {
								break
							}
							eeAddr, ok := eeHost.Endpoint.Address.Address.(*envoy_api_v2_core.Address_SocketAddress)
							if !ok {
								break
							}
							eePort, ok := eeAddr.SocketAddress.PortSpecifier.(*envoy_api_v2_core.SocketAddress_PortValue)
							if !ok {
								break
							}
							eePort.PortValue = updateProto.AppPort
							endpoint.Endpoints[i] = ee
						}
					}
				}

			case ResourceTypeCluster:
				cluster := crs.(*v2.Cluster)
				for _, delProto := range decrProtos {
					if cluster.Name == delProto.Domain {
						continue inner
					}
				}
				if _, ok := decreaseAppSet[cluster.Name]; ok {
					continue inner
				}
			case ResourceTypeListener:
				listener := crs.(*v2.Listener)
				for _, delProto := range decrProtos {
					if listener.Name == delProto.Domain {
						continue inner
					}
				}
				// 更新端口
				for _, updateProto := range updateProtos {
					if listener.Name == updateProto.Domain {
						listener.Address.GetSocketAddress().PortSpecifier.(*envoy_api_v2_core.SocketAddress_PortValue).PortValue = updateProto.ProxyPort
					}
				}
			default:
				continue outer
			}
			okCacheRess = append(okCacheRess, crs)
		}

		var jbs []string
		marshaller := jsonpb.Marshaler{
			OrigName: true,
		}
		buf := bytes.Buffer{}
		for _, crs := range okCacheRess {
			// FIXME: 当增加协议&&增加新的上游时，新上游会有重复的bug
			// TODO: 增加去重逻辑
			bw := bufio.NewWriter(&buf)
			if err = marshaller.Marshal(bw, crs); err != nil {
				return err
			}
			_ = bw.Flush()
			jbs = append(jbs, buf.String())
			buf.Reset()
		}

		hackArray := "[" + strings.Join(jbs, ",") + "]"
		//yb, err := yaml.JSONToYAML(*(*[]byte)(unsafe.Pointer(&hackArray))) // FIXME: why?
		yb, err := yaml.JSONToYAML([]byte(hackArray))
		if err != nil {
			return err
		}
		// FIXME: necessary?
		jb, err := yaml.YAMLToJSON(yb)
		if err != nil {
			return err
		}
		old.Resources[resIdx].Config = *(*string)(unsafe.Pointer(&jb))
		old.Resources[resIdx].YamlConfig = *(*string)(unsafe.Pointer(&yb))
	}

	a.Resources = old.Resources

	return err
}

func ProtocolHash(resName string) string {
	if strings.Count(resName, ".") == 3 {
		return resName
	}

	seps := strings.Split(resName, ".")
	if len(seps) == 5 {
		seps = append(seps[:1], seps[2:]...)
	} else {
		logs.Logger.Errorln("invalid protocol hash:", resName)
		return resName
	}

	return strings.Join(seps, ".")
}

func ResProtocol(resName string) string {
	if strings.Count(resName, ".") == 3 {
		return ProtocolHTTP
	}

	seps := strings.Split(resName, ".")
	if len(seps) == 5 {
		return seps[1]
	}

	return "unknown protocol"
}

func ApplicationTopology() ([]Application, error) {
	var err error

	apps := make([]Application, 0)
	omits := []string{"version", "status", "user_id", "user_name", "description"}
	err = GetDBInstance().Model(Application{}).Omit(omits...).Find(&apps).Error

	return apps, err
}
