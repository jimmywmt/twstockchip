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

// 通用函式：根據 Code 查詢 Stock ID
func stockIDByCode(db *gorm.DB, stockCode string) uint {
	if db == nil {
		return 0
	}
	stock := model.Stock{}
	if err := db.Where("Code = ?", stockCode).First(&stock).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithFields(log.Fields{"code": stockCode}).Errorln("無股票記錄")
		return 0
	}
	return stock.ID
}

// 通用函式：根據 Code 查詢 Dealer ID
func dealerIDByCode(db *gorm.DB, dealerCode string) uint {
	if db == nil {
		return 0
	}
	dealer := model.Dealer{}
	if err := db.Where("Code = ?", dealerCode).First(&dealer).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithFields(log.Fields{"code": dealerCode}).Errorln("無交易所記錄")
		return 0
	}
	return dealer.ID
}

// 將交易寫入資料庫
func addTransaction(db *gorm.DB, stockID, dealerID uint, date *string, tid uint, price float64, buy uint, sell uint) {
	if db == nil || stockID == 0 || dealerID == 0 {
		return
	}
	db.Create(&model.Transaction{
		StockID:  stockID,
		DealerID: dealerID,
		Date:     date,
		TID:      tid,
		Price:    price,
		Buy:      buy,
		Sell:     sell,
	})
}

// 批量寫入 SQLite 的交易數據
func batchInsertToSQLite(transactions *[]model.Transaction) {
	if model.SqliteDB != nil && len(*transactions) > 0 {
		model.SqliteDB.Create(transactions)
		*transactions = make([]model.Transaction, 0, 512)
	}
}

// 讀取 CSV 並寫入資料庫
func ReadCSV(filepath string, stockCode string, date string) {
	var sqliteStockID, postgresStockID uint

	// 打開 CSV 檔案
	data, err := os.OpenFile(filepath, os.O_RDONLY, 0o777)
	if err != nil {
		log.WithError(err).Warningln("開啓檔案失敗")
		return
	}

	defer data.Close()

	reader := csv.NewReader(data)
	trans := make([]model.Transaction, 0, 512)
	reg := regexp.MustCompile("^[a-zA-Z0-9]*")

	log.WithFields(log.Fields{"stockCode": stockCode, "date": date}).Infoln("輸入籌碼資訊")

	// 查詢 Stock ID
	if model.SqliteDB != nil {
		sqliteStockID = stockIDByCode(model.SqliteDB, stockCode)
		if sqliteStockID == 0 {
			return
		}
	}
	if model.PostgresDB != nil {
		postgresStockID = stockIDByCode(model.PostgresDB, stockCode)
		if postgresStockID == 0 {
			return
		}
	}

	// 讀取 CSV 記錄
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
			dealerCode := reg.FindString(record[1])

			if model.SqliteDB != nil {
				dealerID := dealerIDByCode(model.SqliteDB, dealerCode)
				if dealerID != 0 {
					trans = append(trans, model.Transaction{
						StockID:  sqliteStockID,
						DealerID: dealerID,
						Date:     &date,
						TID:      uint(tid),
						Price:    price,
						Buy:      uint(buy),
						Sell:     uint(sell),
					})
				}
			}

			if model.PostgresDB != nil {
				dealerID := dealerIDByCode(model.PostgresDB, dealerCode)
				addTransaction(model.PostgresDB, postgresStockID, dealerID, &date, uint(tid), price, uint(buy), uint(sell))
			}

			// 處理右側交易數據
			if record[6] != "" {
				tid, _ = strconv.ParseUint(record[6], 10, 64)
				price, _ = strconv.ParseFloat(record[8], 64)
				buy, _ = strconv.ParseUint(record[9], 10, 64)
				sell, _ = strconv.ParseUint(record[10], 10, 64)
				dealerCode = reg.FindString(record[7])

				if model.SqliteDB != nil {
					dealerID := dealerIDByCode(model.SqliteDB, dealerCode)
					if dealerID != 0 {
						trans = append(trans, model.Transaction{
							StockID:  sqliteStockID,
							DealerID: dealerID,
							Date:     &date,
							TID:      uint(tid),
							Price:    price,
							Buy:      uint(buy),
							Sell:     uint(sell),
						})
					}
				}

				if model.PostgresDB != nil {
					dealerID := dealerIDByCode(model.PostgresDB, dealerCode)
					addTransaction(model.PostgresDB, postgresStockID, dealerID, &date, uint(tid), price, uint(buy), uint(sell))
				}
			}
		}

		// 批量寫入 SQLite
		batchInsertToSQLite(&trans)
	}

	// 最後批量寫入 SQLite
	batchInsertToSQLite(&trans)

	log.WithFields(log.Fields{"stockCode": stockCode, "date": date}).Infoln("輸入籌碼資訊完成")
}
