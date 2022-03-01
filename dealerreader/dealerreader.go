package dealerreader

import (
	"strings"

	"github.com/jimmywmt/gotool"
	"github.com/jimmywmt/twstockchip/model"
	"github.com/shakinm/xlsReader/xls"

	log "github.com/sirupsen/logrus"
)

func ReadDealerXLS(filepath string) bool {
	if data, err := xls.OpenFile(filepath); err == nil {
		sheet, err := data.GetSheet(0)
		if err != nil {
			log.WithError(err).Errorln("xls 操作失敗")
		}

		for i := 1; i <= sheet.GetNumberRows(); i++ {
			if row, err := sheet.GetRow(i); err == nil {
				cols := row.GetCols()
				if len(cols) == 5 {
					codes := strings.Split(gotool.CompressStr(cols[0].GetString()), "/")
					name := gotool.CompressStr(cols[1].GetString())

					for _, code := range codes {
						model.SmartAddDealer(&code, &name)
					}

				}
			} else {
				log.WithError(err).Errorln("xls 操作失敗")
			}

		}
		log.Infoln("更新股票交易所資料成功")
	} else {
		log.WithError(err).Errorln("開啓檔案失敗")
		return false
	}
	return true
}
