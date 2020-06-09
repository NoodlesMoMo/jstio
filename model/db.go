package model

import (
	"git.sogou-inc.com/iweb/jstio/internel"
	"git.sogou-inc.com/iweb/jstio/internel/logs"
	"sync"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

var (
	instance_ *gorm.DB
	once_     = sync.Once{}
)

func GetDBInstance() *gorm.DB {
	if instance_ != nil {
		return instance_
	}

	connInfo := internel.GetAfxOption().MySQLConn

	cfg := mysql.Config{
		Addr:                 connInfo.Addr,
		User:                 connInfo.User,
		Passwd:               connInfo.Password,
		DBName:               connInfo.DB,
		Net:                  "tcp4",
		AllowNativePasswords: true,
		Params: map[string]string{
			"charset":   "utf8",
			"parseTime": "true",
		},
	}
	const (
		tableOptions = `ENGIN=InnoDB DEFAULT CHARSET=utf8`
		tablePrefix  = `jstio_v2_`
	)

	once_.Do(func() {
		var err error
		instance_, err = gorm.Open(`mysql`, cfg.FormatDSN())
		if err != nil {
			panic(err)
		}

		gorm.DefaultTableNameHandler = func(db *gorm.DB, tableName string) string {
			return tablePrefix + tableName
		}

		instance_.DB().SetMaxIdleConns(16)
		instance_.DB().SetMaxOpenConns(256)
		instance_.Set("gorm:table_options", tableOptions)
		instance_.SetLogger(logs.Logger)
		//instance_.LogMode(true)
	})

	return instance_
}
