package model

import (
	gorm_logrus "github.com/onrik/gorm-logrus"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDBModel(file string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(file), &gorm.Config{
		Logger: gorm_logrus.New(),
	})
	if err != nil {
		log.WithError(err).Fatalln("failed to connect database")
	}

	DB.AutoMigrate(&Stock{})
	DB.AutoMigrate(&Dealer{})
}
