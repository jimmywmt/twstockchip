package csvreader

import (
	"encoding/csv"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/jimmywmt/twstockchip/model"
	log "github.com/sirupsen/logrus"
)

func ReadCSV(filepath string, sid string, date string) {
	data, err := os.OpenFile(filepath, os.O_RDONLY, 0777)
	defer data.Close()

	reg, _ := regexp.Compile("^[a-zA-Z0-9]*")

	if err != nil {
		log.WithError(err).Errorln("開啓檔案失敗")
	}
	reader := csv.NewReader(data)

	log.WithFields(log.Fields{
		"sid":  sid,
		"date": date,
	}).Infoln("輸入籌碼資訊")
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
			dealer := reg.FindString(record[1])
			model.SmartAddTransaction(sid, dealer, &date, uint(tid), price, uint(buy), uint(sell))
			if record[6] != "" {
				tid, _ = strconv.ParseUint(record[6], 10, 64)
				price, _ = strconv.ParseFloat(record[8], 64)
				buy, _ = strconv.ParseUint(record[9], 10, 64)
				sell, _ = strconv.ParseUint(record[10], 10, 64)
				dealer = reg.FindString(record[7])
				log.WithFields(log.Fields{
					"sid": sid,
				}).Debugln("輸入交易序號")
				model.SmartAddTransaction(sid, dealer, &date, uint(tid), price, uint(buy), uint(sell))
			}
		}
	}
}
