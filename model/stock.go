package model

import (
	"errors"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Stock struct {
	gorm.Model
	Code *string `gorm:"uniqueIndex;not null"`
	Name *string `gorm:"not null"`
}

func SmartAddStock(code *string, name *string) {
	addStockToDB := func(db *gorm.DB) {
		if db == nil {
			return
		}
		if err := db.Where("Code = ?", code).First(&Stock{}).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			db.Create(&Stock{Code: code, Name: name})
			log.WithFields(log.Fields{
				"code": *code,
				"name": *name,
			}).Infoln("新增 Stock 到資料庫")
		}
	}

	for _, db := range []*gorm.DB{SqliteDB, PostgresDB} {
		addStockToDB(db)
	}
}
