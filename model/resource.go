package model

import (
	"github.com/jinzhu/gorm"
)

type ResourceType = string

const (
	ResourceTypeRoute    ResourceType = `route`
	ResourceTypeListener ResourceType = `listener`
	ResourceTypeEndpoint ResourceType = `endpoint`
	ResourceTypeCluster  ResourceType = `cluster`
)

func MigrateResourceTables() {
	GetDBInstance().AutoMigrate(Resource{})
}

type Resource struct {
	gorm.Model

	AppID      uint   `gorm:"not null; unique_index:idx_res;" json:"app_id"`
	Name       string `gorm:"not null; unique_index:idx_res;" json:"name"`
	ResType    string `gorm:"not null; unique_index:idx_res;" json:"res_type"`
	Config     string `gorm:"not null; type:LONGTEXT;" json:"config"`
	YamlConfig string `gorm:"not null; type:LONGTEXT;" json:"yaml_config"`
}

func (r *Resource) AfterUpdate() error {
	var err error
	defer r.historyRecord(err, OperateUpdate)
	return err
}

func (r *Resource) AfterDelete() error {
	var err error

	defer r.historyRecord(err, OperateDelete)

	e := EventNone
	switch r.ResType {
	case ResourceTypeRoute:
		e = EventResRouteDelete
	case ResourceTypeListener:
		e = EventResListenerDelete
	case ResourceTypeEndpoint:
		e = EventResEndpointDelete
	case ResourceTypeCluster:
		e = EventResClusterDelete
	}

	GetApplicationCache().ReBuild()

	err = GetEventNotifier().Push(Event(e), r)

	return err
}

func (r *Resource) GetByAppIDAndType() (err error) {
	err = GetDBInstance().Where("app_id = ? and res_type = ?", r.AppID, r.ResType).First(&r).Error

	return
}
func (r *Resource) Create() error {
	err := GetDBInstance().Create(r).Error

	GetApplicationCache().ReBuild()

	return err
}

func (r *Resource) Update(notify bool) error {
	err := GetDBInstance().Model(r).Omit("app_id").Update(r).Error
	if err != nil {
		return err
	}

	GetApplicationCache().ReBuild()

	e := EventNone
	switch r.ResType {
	case ResourceTypeRoute:
		e = EventResRouteUpdate
	case ResourceTypeListener:
		e = EventResListenerUpdate
	case ResourceTypeEndpoint:
		e = EventResEndpointUpdate
	case ResourceTypeCluster:
		e = EventResClusterUpdate
	}

	if notify {
		app, err := GetApplicationByID(uint(r.AppID))
		if err != nil {
			return err
		}
		err = GetEventNotifier().Push(Event(e), &app)
	}

	return err
}

func GetResourceByID(id uint) (Resource, error) {
	var err error

	res := Resource{Model: gorm.Model{ID: id}}
	err = GetDBInstance().First(&res).Error

	return res, err
}

func (r *Resource) historyRecord(err error, operate string) {
	app, _ := GetApplicationByID(uint(r.AppID))
	if err == nil {
		RecordSuccess(app.UserID, app.UserName, r.ResType, operate, r.ID)
	} else {
		RecordFailure(app.UserID, app.UserName, r.ResType, operate, r.ID)
	}
}
