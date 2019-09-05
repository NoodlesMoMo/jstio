package model

import "time"

const (
	OperateCreate = `create`
	OperateUpdate = `update`
	OperateDelete = `delete`

	OperateSuccess = `success`
	OperateFailure = `failure`
)

func MigrateHistoryTables() {
	GetDBInstance().AutoMigrate(History{})
}

type History struct {
	ID        uint      `gorm:"not null; primary_key" json:"id"`
	EntityId  uint      `gorm:"not null; column:entity_id" json:"entity_id"`
	Entity    string    `gorm:"not null" json:"entity"`
	Operate   string    `gorm:"not null;" json:"operate"`
	Status    string    `gorm:"not null;" json:"status"`
	UserId    string    `gorm:"not null;" json:"user_id"`
	UserName  string    `gorm:"not null;" json:"user_name"`
	CreatedAt time.Time `gorm:"not null; index:idx_create" json:"create_at"`
}

func RecordSuccess(userId, userName, entity, operate string, entityId uint) {
	now := time.Now()
	GetDBInstance().Create(&History{
		EntityId:  entityId,
		Entity:    entity,
		UserId:    userId,
		UserName:  userName,
		Operate:   operate,
		Status:    OperateSuccess,
		CreatedAt: now,
	})
}

func RecordFailure(userId, userName, entity, operate string, entityId uint) {
	now := time.Now()
	GetDBInstance().Create(&History{
		EntityId:  entityId,
		Entity:    entity,
		UserId:    userId,
		UserName:  userName,
		Operate:   operate,
		Status:    OperateFailure,
		CreatedAt: now,
	})
}

func GetHistoryRecord(page, size int) (int, []History) {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}

	count := 0
	offset := (page - 1) * size
	result := make([]History, 0)

	db := GetDBInstance()

	db.Model(&History{}).Count(&count)

	db.Offset(offset).Limit(size).Find(&result)

	return count, result
}
