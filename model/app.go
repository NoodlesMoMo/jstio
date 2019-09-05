package model

import (
	"errors"
	"fmt"
	"github.com/ghodss/yaml"
	"jstio/internel/logs"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
)

func MigrateAppsTables() {
	app, refer := Application{}, ApplicationReference{}
	GetDBInstance().AutoMigrate(app, refer)
}

type Application struct {
	gorm.Model

	AppName      string `gorm:"not null; unique_index:idx_cluster; comment:'应用名称'" json:"app_name"`
	OdinCluster  string `gorm:"not null; unique_index:idx_cluster; comment:'odin集群名称'" json:"odin_cluster"`
	Namespace    string `gorm:"not null; unique_index:idx_cluster;" json:"namespace"`
	Environment  string `gorm:"not null; unique_index:idx_cluster; comment:'运行环境'" json:"env"`
	CustomDomain string `gorm:"not null; default:''" json:"custom_domain"`
	AppPorts     string `gorm:"not null" json:"app_ports"`
	Version      string `gorm:"not null; default:'v1'; comment:'版本'" json:"version"`
	Status       string `gorm:"not null; default:'pending'" json:"status"`
	UserId       string `gorm:"not null; comment:'创建人'" json:"user_id"`
	UserName     string `gorm:"not null; comment:'创建人姓名'" json:"user_name"`
	Description  string `gorm:"comment:'描述'" json:"description"`

	Upstream   []*Application `gorm:"-"`
	Downstream []*Application `gorm:"-"`

	Resources []Resource             `gorm:"foreignkey:AppID;"`
	Refers    []ApplicationReference `gorm:"foreignkey:AppID;"`

	ActivePods map[string]PodInfo `gorm:"-"`

	isCompleted bool `gorm:"-"`

	Guard *sync.RWMutex `gorm:"-"`
}

func (a *Application) UnsafeCopy() *Application {
	a.Guard.RLock()
	defer a.Guard.RUnlock()

	other := *a
	other.Guard = &sync.RWMutex{}
	other.Upstream, other.Downstream = make([]*Application, len(a.Upstream)), make([]*Application, len(a.Downstream))
	other.Resources, other.Refers = make([]Resource, len(a.Resources)), make([]ApplicationReference, len(a.Refers))

	for idx, app := range a.Upstream {
		o := *app
		other.Upstream[idx] = &o
	}
	for idx, app := range a.Downstream {
		o := *app
		other.Downstream[idx] = &o
	}
	for idx, res := range a.Resources {
		o := res
		other.Resources[idx] = o
	}
	for idx, refer := range a.Refers {
		o := refer
		other.Refers[idx] = o
	}

	return &other
}

type ApplicationReference struct {
	gorm.Model

	AppID     uint   `gorm:"not null; index:idx_app_id; unique_index:idx_id"`
	ReferID   uint   `gorm:"not null; index:idx_refer; unique_index:idx_id"`
	ReferKind string `gorm:"not null; unique_index:idx_id"`
}

func (r *ApplicationReference) String() string {
	return fmt.Sprintf("%d->%d~%s", r.AppID, r.ReferID, r.ReferKind)
}

func GetAppRefersById(id uint) (upstream, downstream []uint, err error) {
	if id == 0 {
		return
	}

	db := GetDBInstance()
	result := make([]ApplicationReference, 0)
	err = db.Where("app_id = ? and refer_id != 0", id).Or("refer_id = ? and app_id != 0", id).Find(&result).Error
	if err != nil {
		return
	}

	for _, row := range result {

		if row.AppID == id {
			if row.ReferKind == ReferKindUpstream {
				upstream = append(upstream, row.ReferID)
			} else {
				downstream = append(downstream, row.ReferID)
			}
		} else {
			if row.ReferKind == ReferKindUpstream {
				downstream = append(downstream, row.AppID)
			} else {
				upstream = append(upstream, row.AppID)
			}
		}
	}

	return
}

func (a *Application) SelectorFormat() string {
	return defaultRegistryPrefix + path.Join(
		strings.Trim(a.OdinCluster, "/"),
		strings.Trim(a.Namespace, "/"),
		strings.Trim(a.AppName, "/"),
		//strings.Trim(a.Environment, "/"),
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
	a.Environment = seps[4]

	return nil
}

func (a *Application) Hash() string {
	return a.Domain()
}

func (a *Application) Domain() string {
	if a.CustomDomain != "" {
		return a.CustomDomain
	}

	return strings.Join([]string{
		a.AppName,
		a.OdinCluster,
		"odin",
		"sogou",
		// a.Namespace,
		// a.Environment,
	}, ".")
}

func (a *Application) SelfAddresses() []string {
	address := make([]string, 0)
	for _, port := range strings.Split(a.AppPorts, ";") {
		address = append(address, localhost+":"+port)
	}
	return address
}

func (a *Application) SelfPorts() []uint32 {
	ports := make([]uint32, 0)

	for _, sp := range strings.Split(a.AppPorts, ";") {
		port, err := strconv.Atoi(sp)
		if err == nil && port > 0 {
			ports = append(ports, uint32(port))
		}
	}

	return ports
}

func (a *Application) AfterUpdate() error {
	var err error

	defer func() {
		if err == nil {
			RecordSuccess(a.UserId, a.UserName, `application`, OperateUpdate, a.ID)
		} else {
			RecordFailure(a.UserId, a.UserName, `application`, OperateUpdate, a.ID)
		}
	}()

	//err = GetEventNotifier().Push(EventApplicationUpdate, a)

	return err
}

func (a *Application) AfterSave() error {

	RecordSuccess(a.UserId, a.UserName, `application`, OperateCreate, a.ID)

	return nil
}

func (a *Application) AfterDelete() error {
	var err error

	defer func() {
		if err == nil {
			RecordSuccess(a.UserId, a.UserName, `application`, OperateCreate, a.ID)
		} else {
			RecordFailure(a.UserId, a.UserName, `application`, OperateCreate, a.ID)
		}
	}()

	err = GetEventNotifier().Push(EventApplicationDelete, a)

	return err
}

func (a *Application) completeXStream() error {
	if a.isCompleted {
		return nil
	}

	db := GetDBInstance()

	upstreamIds, downstreamIds, err := GetAppRefersById(a.ID)
	if err != nil {
		return err
	}

	if e := db.Where(upstreamIds).Find(&a.Upstream).Error; e != nil {
		return e
	}

	if e := db.Where(downstreamIds).Find(&a.Downstream).Error; e != nil {
		return e
	}

	a.isCompleted = true

	return nil
}

func (a *Application) completeResources() error {
	return GetDBInstance().Find(&a.Resources, "app_id=?", a.ID).Error
}

func (a *Application) Add(refers []uint, kind ReferKind) error {
	var err error

	for _, referId := range refers {
		a.Refers = append(a.Refers, ApplicationReference{
			ReferID:   referId,
			ReferKind: kind,
		})
	}

	err = GetDBInstance().Create(a).Error
	if err == nil {
		_ = a.completeXStream()

		/* add default 4 kinds resources */
		err = a.AddDefaultResources()
		if err != nil {
			return err
		}

		err = GetEventNotifier().Push(EventApplicationCreate, a)
	}

	return err
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
			return fmt.Errorf("yaml to json, appID: %d, error: %s", a.ID, err.Error())
		}

		res := Resource{
			AppID:      a.ID,
			Name:       a.Hash() + "_" + resType,
			ResType:    resType,
			Config:     string(jsonCfg),
			YamlConfig: string(yamlCfg),
		}

		if err = res.Create(); err != nil {
			return err
		}
	}

	return nil
}

func (a *Application) Update(refers []uint, kind ReferKind) error {
	var err error

	tagLog := logs.FuncTaggedLoggerFactory()

	old, err := GetApplicationById(a.ID)
	if err != nil {
		tagLog("get application").WithError(err).Errorln("app_id:", a.ID)
		return err
	}

	needPush := (a.Hash() != old.Hash()) || (a.AppPorts != old.AppPorts)

	db := GetDBInstance()
	if err = db.Model(a).Update(a).Error; err != nil {
		tagLog("app update").WithError(err).Errorln("app_id:", a.ID)
		return err
	}

	now := time.Now()
	for _, referId := range refers {
		appRefer := ApplicationReference{
			Model: gorm.Model{
				CreatedAt: now,
				UpdatedAt: now,
			},
			AppID:     a.ID,
			ReferID:   referId,
			ReferKind: kind,
		}
		a.Refers = append(a.Refers, appRefer)
	}

	if !a.ReferEqual(&old) {
		if err = UpdateAppRefers(a.ID, a.Refers); err != nil {
			tagLog("app refer update").WithError(err).Errorf("app_id: %d, refers: %v\n", a.ID, a.Refers)
			return err
		}
		if err = a.completeXStream(); err != nil {
			tagLog("app xstream complete").WithError(err).Errorln("app_id:", a.ID)
			return err
		}
		if err = a.completeResources(); err != nil {
			tagLog("app resources complete").WithError(err).Errorln("app_id:", a.ID)
			return err
		}
		_ = GetResourceRender().TryMergeUpstreamResources(a)
		needPush = true
	}

	if needPush {
		_ = GetEventNotifier().Push(EventApplicationUpdate, a)
	}

	return err
}

func (a *Application) ReferEqual(other *Application) bool {
	var (
		r, l []string
	)

	for _, refer := range a.Refers {
		r = append(r, refer.String())
	}

	for _, refer := range other.Refers {
		l = append(l, refer.String())
	}

	return strings.Join(r, ",") == strings.Join(l, ",")
}

func GetApplicationById(id uint) (Application, error) {
	var err error

	app := Application{Model: gorm.Model{ID: id}}
	app.Guard = &sync.RWMutex{}

	err = GetDBInstance().Preload("Resources").First(&app).Error
	if err != nil {
		return app, err
	}

	err = app.completeXStream()

	return app, err
}

func AllApps(onlyValid bool) ([]Application, error) {
	var err error

	result := make([]Application, 0)

	db := GetDBInstance().Preload("Resources")

	if onlyValid {
		err = db.Find(&result, "deleted_at is null").Error
	} else {
		err = db.Find(&result).Error
	}

	return result, err
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
		if e := app.completeXStream(); e != nil {
			err = e
		}
		app.Guard = &sync.RWMutex{}
		activeApps[app.Hash()] = &app
	}

	return activeApps, err
}

func UpdateAppRefers(appId uint, refers []ApplicationReference) error {
	var err error

	db := GetDBInstance().Model(&ApplicationReference{})

	if err = db.Unscoped().Where("app_id = ?", appId).Update("deleted_at", time.Now()).Error; err != nil {
		return err
	}

	for _, refer := range refers {
		refer := refer

		row := ApplicationReference{}
		if e := db.Unscoped().Where("app_id=?", appId).Where("refer_id=?", refer.ReferID).First(&row).Error; e != nil && e == gorm.ErrRecordNotFound {
			if err = db.Create(&refer).Error; err != nil {
				return err
			}
			continue
		}

		if e := db.Unscoped().Where("app_id=?", appId).Where("refer_id=?", refer.ReferID).
			Update(map[string]interface{}{
				"refer_kind": refer.ReferKind,
				"deleted_at": nil,
			}).Error; e != nil {
			return e
		}
	}

	return nil
}
