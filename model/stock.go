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
	if err := DB.Where("Code = ?", code).First(&Stock{}).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		DB.Create(&Stock{Code: code, Name: name})
		log.WithFields(log.Fields{
			"code": *code,
			"name": *name,
		}).Infoln("新增股票")
	}
}
