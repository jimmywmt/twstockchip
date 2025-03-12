package model

import (
	"gorm.io/gorm"
)

type Transaction struct {
	gorm.Model
	StockID  uint `gorm:"not null;index"`
	Stock    Stock
	DealerID uint `gorm:"not null;index"`
	Dealer   Dealer
	Date     *string `gorm:"type:date;not null;index"`
	TID      uint    `gorm:"not null"`
	Price    float64 `gorm:"not null"`
	Buy      uint    `gorm:"not null"`
	Sell     uint    `gorm:"not null"`
}

