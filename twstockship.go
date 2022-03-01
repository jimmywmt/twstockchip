package main

import (
	"bytes"
	"image"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/jimmywmt/gotool"
	"github.com/jimmywmt/twstockchip/csvreader"
	"github.com/jimmywmt/twstockchip/dealerreader"
	"github.com/jimmywmt/twstockchip/model"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gocv.io/x/gocv"
)

type record struct {
	id   string
	name string
}

var imgFile = "CaptchaImage.jpeg"

var stockID, s, evValue, vsValue, vsgValue, today string
var success, request, nodata bool
var stocks []*record
var requestImageCount = 0
var matchCount = 0
var slackWebhookURL = "https://hooks.slack.com/services/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"
var wg sync.WaitGroup

func init() {

	log.SetFormatter(&log.TextFormatter{
		ForceQuote:      true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	log.SetOutput(os.Stdout)
}

func generateImageCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36 Edge/44.18363.8131"),
	)

	c.OnRequest(func(r *colly.Request) {
		log.WithFields(log.Fields{
			"url":    r.URL,
			"method": r.Method,
		}).Debugln("訪問")
	})

	c.OnHTML("img[border='0']", func(e *colly.HTMLElement) {
		img_addr := "https://bsr.twse.com.tw/bshtm/" + e.Attr("src")

		c.OnResponse(func(r *colly.Response) {
			reader := bytes.NewReader(r.Body)
			body, _ := ioutil.ReadAll(reader)
			err := ioutil.WriteFile(imgFile, body, 0755)

			if err != nil {
				log.WithError(err).Warnln("連接失敗")
				request = false
			} else {
				log.Infoln("下載 captcha 圖片成功")
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
		log.WithError(err).Warnln("訪問 URL:", r.Request.URL, "失敗")
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
		}).Debugln("訪問")
	})

	c.OnHTML("span[id='Label_ErrorMsg']", func(e *colly.HTMLElement) {
		result_check := e.Text
		if len(result_check) == 0 {
			matchCount++

			c.OnResponse(func(r *colly.Response) {
				reader := bytes.NewReader(r.Body)
				body, _ := ioutil.ReadAll(reader)
				err := ioutil.WriteFile("./csv/"+today+"/"+stockID+".csv", body, 0755)

				if err != nil {
					log.Warnln(err)
					request = false
				} else {
					log.WithFields(log.Fields{
						"stock": stockID,
					}).Infoln("下載交易籌碼檔案成功")
				}
			})

			c.Visit("https://bsr.twse.com.tw/bshtm/bsContent.aspx")

			if request {
				success = true

			}
		} else if result_check == "查無資料" {
			matchCount++
			nodata = true

			log.WithFields(log.Fields{
				"stock": stockID,
			}).Warnln("無交易資料")
			success = true
		} else {
			log.WithFields(log.Fields{
				"captcha string": s,
			}).Infoln("captcha 不符合")
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.WithError(err).Warnln("訪問 URL:", r.Request.URL, "失敗")
		request = false
	})

	return c
}

func downloadChip(target string) {
	stockID = target
	log.WithFields(log.Fields{
		"stock": stockID,
	}).Infoln("開始下載交易籌碼")

	success = false
	for !success {
		request = true
		c := generateImageCollector()
		c.Visit("https://bsr.twse.com.tw/bshtm/bsMenu.aspx")
		if !request {
			log.Infoln("暫停1分鐘")
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
			}).Infoln("錯誤 captcha 長度")
		}

		if !request {
			log.Infoln("暫停1分鐘")
			time.Sleep(time.Minute)
			continue
		}
		if !success {
			log.Infoln("暫停2秒")
			time.Sleep(2 * time.Second)
		}

	}
}

func readStockList() {
	status := false
	stocks = make([]*record, 0, 2048)

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36 Edge/44.18363.8131"),
	)

	c.SetRequestTimeout(30 * time.Second)

	c.OnRequest(func(r *colly.Request) {
		r.ResponseCharacterEncoding = "big5"
		log.WithFields(log.Fields{
			"url":    r.URL,
			"method": r.Method,
		}).Debugln("訪問")
	})

	c.OnError(func(r *colly.Response, err error) {
		log.WithError(err).Warnln("訪問 URL:", r.Request.URL, "失敗")
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
				id := gotool.CompressStr(idName[0])
				name := gotool.CompressStr(idName[1])
				stocks = append(stocks, &record{id: id, name: name})
				model.SmartAddStock(&id, &name)
			}
		}
		if len(stocks) > 0 {
			request = true
		}
	})

	c.Visit("https://isin.twse.com.tw/isin/C_public.jsp?strMode=2")
}

func checkToday() bool {
	check := false

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36 Edge/44.18363.8131"),
	)

	c.OnRequest(func(r *colly.Request) {
		log.WithFields(log.Fields{
			"url":    r.URL,
			"method": r.Method,
		}).Debugln("訪問")
	})

	c.OnHTML("span#Label_Date", func(e *colly.HTMLElement) {
		date := strings.ReplaceAll(e.Text, "/", "-")
		if date != today {
			log.WithFields(log.Fields{
				"date":  date,
				"today": today,
			}).Infoln("資料尚未釋出")
		} else {
			log.WithFields(log.Fields{
				"today": today,
			}).Infoln("資料已釋出")
			check = true
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.WithError(err).Warnln("訪問 URL:", r.Request.URL, "失敗")
	})

	c.Visit("https://bsr.twse.com.tw/bshtm/bsWelcome.aspx")

	return check
}

func downloadDealerInfo() bool {
	result := true
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36 Edge/44.18363.8131"),
	)

	c.OnRequest(func(r *colly.Request) {
		log.WithFields(log.Fields{
			"url":    r.URL,
			"method": r.Method,
		}).Debugln("訪問")
	})

	c.OnHTML("html", func(e *colly.HTMLElement) {
		c.OnResponse(func(r *colly.Response) {
			reader := bytes.NewReader(r.Body)
			body, _ := ioutil.ReadAll(reader)
			err := ioutil.WriteFile("./dealers.xls", body, 0755)

			if err != nil {
				log.Warnln(err)
				request = false
			} else {
				log.Infoln("下載股票交易所資料成功")
			}
		})
		c.Visit("https://www.twse.com.tw/brokerService/outPutExcel")
	})

	c.OnError(func(r *colly.Response, err error) {
		log.WithError(err).Warnln("訪問 URL:", r.Request.URL, "失敗")
		result = false
	})

	c.Visit("https://www.twse.com.tw/brokerService/brokerServiceAudit")
	return result
}

func createDir() {
	if _, err := os.Stat("./csv"); os.IsNotExist(err) {
		if err := os.Mkdir("./csv", 0755); err != nil {
			log.WithError(err).Fatalln("建立資料夾失敗")
		}
		log.WithFields(log.Fields{
			"dir": "./csv",
		}).Infoln("建立資料夾")
	}

	if _, err := os.Stat("./csv/" + today); os.IsNotExist(err) {
		if err := os.Mkdir("./csv/"+today, 0755); err != nil {
			log.WithError(err).Fatalln("建立資料夾失敗")
		}
		log.WithFields(log.Fields{
			"dir": "./csv/" + today,
		}).Infoln("建立資料夾")
	}
}

func compressFolder() {
	path := "./csv/" + today
	log.WithFields(log.Fields{
		"file": path + ".tar.zst",
	}).Infoln("壓縮資料")

	cmd := exec.Command("tar", "--exclude='.[^/]*'", "--zstd", "-cvf", path+".tar.zst", "-C", "./csv/", today)

	if _, err := cmd.Output(); err != nil {
		log.WithError(err).Warnln("壓縮資料失敗")
	} else {
		os.RemoveAll(path)
		log.WithFields(log.Fields{
			"file": path + ".tar.zst",
		}).Infoln("壓縮資料成功")
	}
}

func uncompressFolder(fileName *string) {
	cmd := exec.Command("tar", "xvf", *fileName)
	reg, _ := regexp.Compile("[0-9]...-[0-1][0-9]-[0-3][0-9]")
	date := reg.FindString(*fileName)
	log.WithFields(log.Fields{
		"dir": "./" + date,
	}).Infoln("解壓資料")
	if _, err := cmd.Output(); err != nil {
		log.WithError(err).Warnln("解壓資料失敗")
	} else {
		log.WithFields(log.Fields{
			"dir": "./" + date,
		}).Infoln("解壓資料成功")
	}
}

func updateEssentialInformation() {
	model.InitDBModel("./twstockship.sqlite")
	request = false
	for !request {
		if downloadDealerInfo() {
			if dealerreader.ReadDealerXLS("./dealers.xls") {
				request = true
			} else {
				log.Infoln("暫停1分鐘")
				time.Sleep(time.Minute)
			}

		} else {
			log.Infoln("暫停1分鐘")
			time.Sleep(time.Minute)
		}
	}

	request = false
	for !request {
		readStockList()
	}
	log.Infoln("更新股票資訊成功")

}

func writingRoutine(tasks chan string) {
	defer wg.Done()
	var task string
	dir := "./csv/" + today + "/"

	for {
		task = <-tasks
		switch task {
		case "close":
			break
		default:
			fileName := dir + task + ".csv"
			csvreader.ReadCSV(fileName, task, today)
		}
	}
}

func main() {
	runtime.GOMAXPROCS(1)
	app := &cli.App{
		Name:  "twstockship",
		Usage: "臺灣股市交易籌碼資料下載",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "date",
				Aliases:     []string{"d"},
				Usage:       "指定日期 (format 2016-01-02)",
				Value:       time.Now().Format("2006-01-02"),
				DefaultText: time.Now().Format("2006-01-02"),
			},
			&cli.StringFlag{
				Name:        "loglevel",
				Aliases:     []string{"l"},
				Usage:       "設定log等級 (debug, info, warm, error, fatal, panic)",
				Value:       "info",
				DefaultText: "info",
			},
		},

		Before: func(c *cli.Context) error {
			switch c.String("loglevel") {
			case "debug":
				log.SetLevel(log.DebugLevel)
			case "info":
				log.SetLevel(log.InfoLevel)
			case "warm":
				log.SetLevel(log.WarnLevel)
			case "error":
				log.SetLevel(log.ErrorLevel)
			case "fatal":
				log.SetLevel(log.FatalLevel)
			case "panic":
				log.SetLevel(log.PanicLevel)
			}
			return nil
		},

		Commands: []*cli.Command{
			{
				Name:    "rebuild",
				Aliases: []string{"r"},
				Usage:   "指定日期重新建立資料庫",
				Action: func(c *cli.Context) error {
					updateEssentialInformation()
					fileList := gotool.DirRegListFiles("./csv", "^[0-9]...-[0-1][0-9]-[0-3][0-9].tar.zst")
					reg, _ := regexp.Compile("[0-9]...-[0-1][0-9]-[0-3][0-9]")
					firstDate, _ := time.Parse("2006-01-02", c.String("date"))
					for _, fileName := range fileList {
						dateString := reg.FindString(*fileName)
						fileDate, _ := time.Parse("2006-01-02", dateString)
						if firstDate.Before(fileDate) || firstDate.Equal(fileDate) {
							uncompressFolder(fileName)
							folder := "./" + dateString
							csvFileList := gotool.DirRegListFiles(folder, ".*csv$")
							for _, csvFile := range csvFileList {
								csvNameSlice := strings.Split(*csvFile, "/")
								nameWithExtension := csvNameSlice[2]
								stockCode := nameWithExtension[0 : len(nameWithExtension)-4]
								csvreader.ReadCSV(*csvFile, stockCode, dateString)
							}
							os.RemoveAll(folder)
						}
					}
					return nil
				},
			},
			{
				Name:    "download",
				Aliases: []string{"d"},
				Usage:   "下載指定日期交易籌碼 (需交易所網頁釋出)",
				Action: func(c *cli.Context) error {
					slackWebhook := gotool.NewSlackWebhook(slackWebhookURL)
					slackWebhook.SentMessage("開始下載今日交易籌碼")
					updateEssentialInformation()
					wg.Add(1)
					tasks := make(chan string, 16)
					go writingRoutine(tasks)
					start := time.Now()
					today = c.String("date")
					for !checkToday() {
						log.Infoln("暫停1分鐘")
						time.Sleep(time.Minute)
					}

					createDir()
					for _, s := range stocks {
						nodata = false
						downloadChip(s.id)
						if !nodata {
							tasks <- s.id
						}
						log.Infoln("暫停2秒")
						time.Sleep(2 * time.Second)
					}
					tasks <- "close"
					elapsed := time.Since(start)
					log.WithFields(log.Fields{
						"matchCount/requestImageCount": float64(matchCount) / float64(requestImageCount),
					}).Info("captcha 正確率")
					log.WithFields(log.Fields{
						"elapsed": elapsed,
					}).Printf("程序用時")
					slackWebhook.SentMessage("下載今日交易籌碼成功")
					wg.Wait()
					compressFolder()

					return nil
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
