package model

import (
	"errors"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Dealer struct {
	gorm.Model
	Code string `gorm:"uniqueIndex"`
	Name string
}

func SmartAddDealer(code string, name string) {
	if err := DB.Where("Code = ?", code).First(&Dealer{}).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		DB.Create(&Dealer{Code: code, Name: name})
		log.WithFields(log.Fields{
			"code": code,
			"name": name,
		}).Infoln("add dealer success")
	}
}
