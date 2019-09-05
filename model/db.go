package model

import (
	"jstio/internel"
	. "jstio/internel/logs"
	"sync"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

var (
	instance_ *gorm.DB
	once_     = sync.Once{}
)

func GetDBInstance() *gorm.DB {
	if instance_ != nil {
		return instance_
	}

	connInfo := internel.GetAfxMeta().MySQLConn

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
		tablePrefix  = `jstio_`
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
		instance_.SetLogger(&sqlLogger{Logger})
		instance_.LogMode(true)
	})

	return instance_
}

type sqlLogger struct {
	*logrus.Logger
}

func (sql *sqlLogger) Print(v ...interface{}) {
	sql.WithField(`db`, v).Println()
}
