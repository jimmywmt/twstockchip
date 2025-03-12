package model

import (
	"errors"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Dealer struct {
	gorm.Model
	Code *string `gorm:"uniqueIndex;not null"`
	Name *string `gorm:"not null"`
}

func SmartAddDealer(code *string, name *string) {
	addDealerToDB := func(db *gorm.DB) {
		if db == nil {
			return
		}
		if err := db.Where("Code = ?", code).First(&Dealer{}).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			db.Create(&Dealer{Code: code, Name: name})
			log.WithFields(log.Fields{
				"code": *code,
				"name": *name,
			}).Infoln("新增 Dealer 到資料庫")
		}
	}

	for _, db := range []*gorm.DB{SqliteDB, PostgresDB} {
		addDealerToDB(db)
	}
}
