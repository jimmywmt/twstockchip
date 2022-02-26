package model

import (
	"errors"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Transaction struct {
	gorm.Model
	StockID  uint `gorm:"not null;index"`
	Stock    Stock
	DealerID uint `gorm:"not null"`
	Dealer   Dealer
	Date     *string `gorm:"type:date;not null;index`
	TID      uint    `gorm:"not null"`
	Price    float64 `gorm:"not null"`
	Buy      uint    `gorm:"not null"`
	Sell     uint    `gorm:"not null"`
}

func SmartAddTransaction(stockCode string, dealerCode string, date *string, tid uint, price float64, buy uint, sell uint) {
	stock := Stock{}
	dealer := Dealer{}
	if err := DB.Where("Code = ?", stockCode).First(&stock).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithFields(log.Fields{
			"code": stockCode,
		}).Errorln("no stock")
		return
	}

	if err := DB.Where("Code = ?", dealerCode).First(&dealer).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithFields(log.Fields{
			"code": stockCode,
		}).Errorln("no dealer")
		return
	}

	if err := DB.Where("stock_id = ?", stock.ID).Where("dealer_id = ?", dealer.ID).Where("date = ?", date).Where("t_id = ?", tid).First(&Transaction{}).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		transaction := Transaction{
			StockID:  stock.ID,
			DealerID: dealer.ID,
			Date:     date,
			TID:      tid,
			Price:    price,
			Buy:      buy,
			Sell:     sell,
		}
		DB.Save(&transaction)
	} else {
		log.WithFields(log.Fields{
			"stockCode": stockCode,
			"date":      *date,
			"tid":       tid,
		}).Warnln("record has already existed in the DB")
	}

}
