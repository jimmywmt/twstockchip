package csvreader

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/jimmywmt/twstockchip/model"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func stockIDByCode(stockCode string) uint {
	stock := model.Stock{}
	if err := model.DB.Where("Code = ?", stockCode).First(&stock).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithFields(log.Fields{
			"code": stockCode,
		}).Errorln("無股票記錄")
		return 0
	}
	return stock.ID
}

func dealerIDByCode(dealerCode string) uint {
	dealer := model.Dealer{}
	if err := model.DB.Where("Code = ?", dealerCode).First(&dealer).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithFields(log.Fields{
			"code": dealerCode,
		}).Errorln("無股票交易所記錄")
		return 0
	}
	return dealer.ID
}

func ReadCSV(filepath string, stockCode string, date string) {
	data, err := os.OpenFile(filepath, os.O_RDONLY, 0777)
	trans := make([]model.Transaction, 0, 512)
	defer data.Close()

	reg, _ := regexp.Compile("^[a-zA-Z0-9]*")

	if err != nil {
		log.WithError(err).Warningln("開啓檔案失敗")
		return
	}
	reader := csv.NewReader(data)

	log.WithFields(log.Fields{
		"stockCode": stockCode,
		"date":      date,
	}).Infoln("輸入籌碼資訊")
	stockID := stockIDByCode(stockCode)
	if stockID == 0 {
		return
	}
	for {
		record, err := reader.Read()

		if err == io.EOF {
			break
		}

		if len(record) == 11 && record[0] != "序號" {
			tid, _ := strconv.ParseUint(record[0], 10, 64)
			price, _ := strconv.ParseFloat(record[2], 64)
			buy, _ := strconv.ParseUint(record[3], 10, 64)
			sell, _ := strconv.ParseUint(record[4], 10, 64)
			dealerID := dealerIDByCode(reg.FindString(record[1]))
			//                         model.SmartAddTransaction(sid, dealer, &date, uint(tid), price, uint(buy), uint(sell))
			trans = append(trans, model.Transaction{
				StockID:  stockID,
				DealerID: dealerID,
				Date:     &date,
				TID:      uint(tid),
				Price:    price,
				Buy:      uint(buy),
				Sell:     uint(sell),
			})
			if record[6] != "" {
				tid, _ = strconv.ParseUint(record[6], 10, 64)
				price, _ = strconv.ParseFloat(record[8], 64)
				buy, _ = strconv.ParseUint(record[9], 10, 64)
				sell, _ = strconv.ParseUint(record[10], 10, 64)
				dealerID = dealerIDByCode(reg.FindString(record[7]))
				log.WithFields(log.Fields{
					"stockID": stockID,
				}).Debugln("輸入交易序號")
				//                                 model.SmartAddTransaction(sid, dealer, &date, uint(tid), price, uint(buy), uint(sell))
				trans = append(trans, model.Transaction{
					StockID:  stockID,
					DealerID: dealerID,
					Date:     &date,
					TID:      uint(tid),
					Price:    price,
					Buy:      uint(buy),
					Sell:     uint(sell),
				})
			}
		}
		if len(trans) == 512 {
			model.DB.Create(&trans)
			trans = make([]model.Transaction, 0, 512)
		}
	}
	if len(trans) != 0 {
		model.DB.Create(&trans)
	}
}
