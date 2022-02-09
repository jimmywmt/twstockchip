package main

import (
	"bytes"
	"image"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/jimmywmt/gotool"
	log "github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
)

type stock struct {
	id   string
	name string
}

var imgFile = "CaptchaImage.jpeg"

var stockID, s, evValue, vsValue, vsgValue string
var success, request bool
var stocks []*stock
var requestImageCount = 0
var matchCount = 0

func init() {

	log.SetFormatter(&log.TextFormatter{
		ForceQuote:      true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

}

func generateImageCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36 Edge/44.18363.8131"),
	)

	c.OnRequest(func(r *colly.Request) {
		log.WithFields(log.Fields{
			"url":    r.URL,
			"method": r.Method,
		}).Debugln("visit webpage")
	})

	c.OnHTML("img[border='0']", func(e *colly.HTMLElement) {
		img_addr := "https://bsr.twse.com.tw/bshtm/" + e.Attr("src")

		c.OnResponse(func(r *colly.Response) {
			reader := bytes.NewReader(r.Body)
			body, _ := ioutil.ReadAll(reader)
			err := ioutil.WriteFile(imgFile, body, 0755)

			if err != nil {
				log.Warnln(err)
				request = false
			} else {
				log.Infoln("download captcha image success")
			}
		})

		c.Visit(img_addr)

		if request {
			requestImageCount++
			img := gocv.IMRead(imgFile, gocv.IMReadColor)
			kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
			gocv.Erode(img, &img, kernel)
			gocv.Dilate(img, &img, kernel)
			nimg := gocv.NewMat()
			gocv.BilateralFilter(img, &nimg, 35, 35, 6)
			gocv.MedianBlur(nimg, &img, 5)
			gocv.MedianBlur(img, &nimg, 5)
			img = nimg.Region(image.Rect(0, int(float64(nimg.Rows())*0.1), nimg.Cols(), int(float64(nimg.Rows())*0.8)))
			gocv.Threshold(img, &nimg, 180, 255, gocv.ThresholdBinary)
			gocv.IMWrite("img.jpeg", nimg)

			cmd := exec.Command("tesseract", "img.jpeg", "stdout", "--psm", "13", "--oem", "0", "-c", "tessedit_char_whitelist=ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
			out, _ := cmd.Output()
			s = gotool.CompressStr(string(out))

		}
	})

	c.OnHTML("input[name='__EVENTVALIDATION']", func(e *colly.HTMLElement) {
		evValue = e.Attr("value")
	})

	c.OnHTML("input[name='__VIEWSTATE']", func(e *colly.HTMLElement) {
		vsValue = e.Attr("value")
	})

	c.OnHTML("input[name='__VIEWSTATEGENERATOR']", func(e *colly.HTMLElement) {
		vsgValue = e.Attr("value")
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Warnln("request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
		request = false
	})

	return c
}

func generateDownloadCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36 Edge/44.18363.8131"),
	)

	c.OnRequest(func(r *colly.Request) {
		log.WithFields(log.Fields{
			"url":    r.URL,
			"method": r.Method,
		}).Debugln("visit")
	})

	c.OnHTML("span[id='Label_ErrorMsg']", func(e *colly.HTMLElement) {
		result_check := e.Text
		if len(result_check) == 0 {
			matchCount++

			c.OnResponse(func(r *colly.Response) {
				reader := bytes.NewReader(r.Body)
				body, _ := ioutil.ReadAll(reader)
				err := ioutil.WriteFile("./csv/"+stockID+".csv", body, 0755)

				if err != nil {
					log.Warnln(err)
					request = false
				} else {
					log.WithFields(log.Fields{
						"stock": stockID,
					}).Infoln("download chip file success")
				}
			})

			c.Visit("https://bsr.twse.com.tw/bshtm/bsContent.aspx")

			if request {
				success = true

			}
		} else if result_check == "查無資料" {
			matchCount++

			log.WithFields(log.Fields{
				"stock": stockID,
			}).Warnln("no input stock")
			success = true
		} else {
			log.WithFields(log.Fields{
				"captcha string": s,
			}).Infoln("check captcha dismatch")
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Warnln("request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
		request = false
	})

	return c
}

func downloadChip(target string) {
	stockID = target
	log.WithFields(log.Fields{
		"stock": stockID,
	}).Infoln("start downloading process")

	success = false
	for !success {
		request = true
		c := generateImageCollector()
		c.Visit("https://bsr.twse.com.tw/bshtm/bsMenu.aspx")
		if !request {
			log.Infoln("wait 1 minute")
			time.Sleep(time.Minute)
			continue
		}

		var formData = map[string]string{
			"__EVENTTARGET":        "",
			"__EVENTARGUMENT":      "",
			"__LASTFOCUS:":         "",
			"__VIEWSTATE":          vsValue,
			"__VIEWSTATEGENERATOR": vsgValue,
			"__EVENTVALIDATION":    evValue,
			"RadioButton_Normal":   "RadioButton_Normal",
			"TextBox_Stkno":        stockID,
			"CaptchaControl1":      s,
			"btnOK":                "查詢",
		}

		request = true

		c2 := generateDownloadCollector()
		if len(s) == 5 {
			c2.Post("https://bsr.twse.com.tw/bshtm/bsMenu.aspx", formData)
		} else {
			log.WithFields(log.Fields{
				"captcha string": s,
			}).Infoln("wrong length of captcha")
		}

		if !request {
			log.Infoln("wait 1 minute")
			time.Sleep(time.Minute)
			continue
		}
		if !success {
			log.Infoln("wait 2 seconds")
			time.Sleep(2 * time.Second)
		}

	}
}

func readStockList() {
	status := false
	stocks = make([]*stock, 0, 2048)

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36 Edge/44.18363.8131"),
	)

	c.SetRequestTimeout(30 * time.Second)

	c.OnRequest(func(r *colly.Request) {
		r.ResponseCharacterEncoding = "big5"
	})

	c.OnRequest(func(r *colly.Request) {
		log.WithFields(log.Fields{
			"url":    r.URL,
			"method": r.Method,
		}).Debugln("visit webpage")
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Warnln("request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.OnHTML("table.h4 tr td[bgcolor='#FAFAD2']", func(e *colly.HTMLElement) {
		if string(e.Attr("colspan")) == "7" {
			stockClass := string(e.Text)
			if !strings.Contains(stockClass, "上市認購(售)權證") && !strings.Contains(stockClass, "受益證券") {
				status = true
			} else {
				status = false
			}
		} else if status == true {
			data := string(e.Text)
			idName := strings.Split(data, "\u3000")
			if len(idName) > 1 {
				stocks = append(stocks, &stock{id: gotool.CompressStr(idName[0]), name: gotool.CompressStr(idName[1])})
			}
		}
		if len(stocks) > 0 {
			request = true
		}
	})

	c.Visit("https://isin.twse.com.tw/isin/C_public.jsp?strMode=2")
}

func main() {
	start := time.Now()
	request = false
	for !request {
		readStockList()
	}
	log.Infoln("update stocks information success")

	for _, s := range stocks {
		downloadChip(s.id)
		log.Infoln("wait 2 seconds")
		time.Sleep(2 * time.Second)
	}
	elapsed := time.Since(start)
	log.WithFields(log.Fields{
		"matchCount/requestImageCount": float64(matchCount) / float64(requestImageCount),
	}).Info("captcha match rate")
	log.WithFields(log.Fields{
		"elapsed": elapsed,
	}).Printf("process took")
}
