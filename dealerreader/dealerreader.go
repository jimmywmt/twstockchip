package dealerreader

import (
	"github.com/jimmywmt/gotool"
	"github.com/jimmywmt/twstockchip/model"
	"github.com/shakinm/xlsReader/xls"

	log "github.com/sirupsen/logrus"
)

func ReadDealerXLS(filepath string) {
	log.Infoln("start read dealer info")
	if data, err := xls.OpenFile(filepath); err == nil {
		sheet, err := data.GetSheet(0)
		if err != nil {
			log.WithError(err).Errorln("xls operation error")
		}

		for i := 1; i <= sheet.GetNumberRows(); i++ {
			if row, err := sheet.GetRow(i); err == nil {
				cols := row.GetCols()
				if len(cols) == 5 {
					code := gotool.CompressStr(cols[0].GetString())
					name := gotool.CompressStr(cols[1].GetString())
					model.SmartAddDealer(code, name)

				}
			} else {
				log.WithError(err).Errorln("xls operation error")
			}

		}
	} else {
		log.WithError(err).Errorln("open file error")
	}
	log.Infoln("update dealer info success")
}
