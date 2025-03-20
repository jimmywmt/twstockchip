package main

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
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
	"github.com/jimmywmt/twstockchip/config"
	"github.com/jimmywmt/twstockchip/csvreader"
	"github.com/jimmywmt/twstockchip/dealerreader"
	"github.com/jimmywmt/twstockchip/model"
	"github.com/jimmywmt/twstockchip/tool"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/rand"
)

type record struct {
	id   string
	name string
}

var imgFile = "captchaimage.jpeg"

var (
	stockCode, s, evValue, vsValue, vsgValue, today string
	success, request, nodata                        bool
	stocks                                          []*record
	requestImageCount                               = 0
	matchCount                                      = 0
)

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
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 Edg/131.0.2903.86"),
	)

	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

	c.OnRequest(func(r *colly.Request) {
		log.WithFields(log.Fields{
			"URL":    r.URL,
			"Method": r.Method,
		}).Debugln("訪問")
		r.Headers.Set("Sec-CH-UA", `"Chromium";v="132", "Microsoft Edge";v="131", "Not=A?Brand";v="99"`)
		r.Headers.Set("Sec-CH-UA-Mobile", "?0")          // 是否為行動裝置
		r.Headers.Set("Sec-CH-UA-Platform", `"Windows"`) // 作業系統
		r.Headers.Set("Sec-CH-UA-Platform-Version", `"10.0.0"`)
	})

	c.OnHTML("img[border='0']", func(e *colly.HTMLElement) {
		img_addr := "https://bsr.twse.com.tw/bshtm/" + e.Attr("src")

		c.OnResponse(func(r *colly.Response) {
			reader := bytes.NewReader(r.Body)
			body, _ := io.ReadAll(reader)
			err := os.WriteFile(imgFile, body, 0o755)

			if err != nil {
				log.WithError(err).Warnln("寫入 CAPTCHA 圖片失敗")
				request = false
			} else {
				log.Infoln("下載 CAPTCHA 圖片成功")
			}
		})

		err := c.Visit(img_addr)
		if err != nil {
			request = false
		}

		if request {
			requestImageCount++

			err := tool.ProcessImage(imgFile, "img.jpeg")
			if err != nil {
				log.WithError(err).Warnln("處理圖片失敗")
				return
			}

			cmd := exec.Command("tesseract", "img.jpeg", "stdout", "--psm", "7", "--oem", "1", "-c", "tessedit_char_whitelist=ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

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

	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

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
				body, _ := io.ReadAll(reader)
				err := os.WriteFile("./csv/"+today+"/"+stockCode+".csv", body, 0o755)

				if err != nil {
					log.WithError(err).Warnln("寫入交易籌碼檔案失敗")
					request = false
				} else {
					log.WithFields(log.Fields{
						"stockCode": stockCode,
					}).Infoln("下載交易籌碼檔案成功")
				}
			})

			err := c.Visit("https://bsr.twse.com.tw/bshtm/bsContent.aspx")
			if err != nil {
				request = false
			}

			if request {
				success = true
			}
		} else if result_check == "查無資料" {
			matchCount++
			nodata = true

			log.WithFields(log.Fields{
				"stockCode": stockCode,
			}).Warnln("無交易資料")
			success = true
		} else {
			log.WithFields(log.Fields{
				"CAPTCHA string": s,
			}).Infoln("CAPTCHA 不符合")
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.WithError(err).Warnln("訪問 URL:", r.Request.URL, "失敗")
		request = false
	})

	return c
}

func downloadChip(target *string) {
	stockCode = *target
	log.WithFields(log.Fields{
		"stockCode": stockCode,
	}).Infoln("開始下載交易籌碼")

	success = false
	for !success {
		request = true
		c := generateImageCollector()
		err := c.Visit("https://bsr.twse.com.tw/bshtm/bsMenu.aspx")
		if err != nil {
			request = false
		}
		if !request {
			log.Infoln("暫停1分鐘")
			time.Sleep(time.Minute)
			continue
		}

		formData := map[string]string{
			"__EVENTTARGET":        "",
			"__EVENTARGUMENT":      "",
			"__LASTFOCUS:":         "",
			"__VIEWSTATE":          vsValue,
			"__VIEWSTATEGENERATOR": vsgValue,
			"__EVENTVALIDATION":    evValue,
			"RadioButton_Normal":   "RadioButton_Normal",
			"TextBox_Stkno":        stockCode,
			"CaptchaControl1":      s,
			"btnOK":                "查詢",
		}

		request = true

		c2 := generateDownloadCollector()
		if len(s) == 5 {
			err := c2.Post("https://bsr.twse.com.tw/bshtm/bsMenu.aspx", formData)
			if err != nil {
				request = false
			}
		} else {
			log.WithFields(log.Fields{
				"CAPTCHA string": s,
			}).Infoln("錯誤 CAPTCHA 長度")
		}

		if !request {
			log.Infoln("暫停1分鐘")
			time.Sleep(time.Minute)
			continue
		}
		if !success {
			log.Infoln("暫停4~6秒")
			time.Sleep(time.Duration(4+rand.Intn(3)) * time.Second)
		}

	}
}

func readStockList() {
	status := false
	stocks = make([]*record, 0, 2048)

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36 Edge/44.18363.8131"),
	)

	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

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
		} else if status {
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

	err := c.Visit("https://isin.twse.com.tw/isin/C_public.jsp?strMode=2")
	if err != nil {
		os.Exit(1)
	}
}

func checkToday() bool {
	check := false

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36 Edge/44.18363.8131"),
	)

	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

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

	err := c.Visit("https://bsr.twse.com.tw/bshtm/bsWelcome.aspx")
	if err != nil {
		os.Exit(1)
	}

	return check
}

func downloadDealerInfo() bool {
	result := true
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64; Xbox; Xbox One) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36 Edge/44.18363.8131"),
	)

	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

	c.OnRequest(func(r *colly.Request) {
		log.WithFields(log.Fields{
			"url":    r.URL,
			"method": r.Method,
		}).Debugln("訪問")
	})

	c.OnHTML("html", func(e *colly.HTMLElement) {
		c.OnResponse(func(r *colly.Response) {
			reader := bytes.NewReader(r.Body)
			body, _ := io.ReadAll(reader)
			err := os.WriteFile("./dealers.xls", body, 0o755)

			if err != nil {
				log.Warnln(err)
				request = false
			} else {
				log.Infoln("下載股票交易所資料成功")
			}
		})
		err := c.Visit("https://www.twse.com.tw/rwd/zh/brokerService/outPutExcel")
		if err != nil {
			request = false
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.WithError(err).Warnln("訪問 URL:", r.Request.URL, "失敗")
		result = false
	})

	err := c.Visit("https://www.twse.com.tw/brokerService/brokerServiceAudit")
	if err != nil {
		result = false
	}
	return result
}

func createDir() {
	if _, err := os.Stat("./csv"); os.IsNotExist(err) {
		if err := os.Mkdir("./csv", 0o755); err != nil {
			log.WithError(err).Fatalln("建立資料夾失敗")
		}
		log.WithFields(log.Fields{
			"dir": "./csv",
		}).Infoln("建立資料夾")
	}

	if _, err := os.Stat("./csv/" + today); os.IsNotExist(err) {
		if err := os.Mkdir("./csv/"+today, 0o755); err != nil {
			log.WithError(err).Fatalln("建立資料夾失敗")
		}
		log.WithFields(log.Fields{
			"dir": "./csv/" + today,
		}).Infoln("建立資料夾")
	}
}

func updateEssentialInformation(sqlitefile string, postgresDSN string) {
	model.InitDBModel(sqlitefile, postgresDSN)

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
			log.Infoln("寫入routine結束")
			return
		default:
			fileName := dir + task + ".csv"
			csvreader.ReadCSV(fileName, task, today)
		}
	}
}

func ScheduleTask(hour, minute, second int, task func()) {
	go func() {
		for {
			// Get the current time
			now := time.Now()

			// Calculate the next scheduled time
			nextRun := time.Date(
				now.Year(),
				now.Month(),
				now.Day(),
				hour,
				minute,
				second,
				0,
				now.Location(),
			)

			// If the next scheduled time is in the past, schedule for the next day
			if nextRun.Before(now) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			// Wait until the next scheduled time
			time.Sleep(time.Until(nextRun))

			// Execute the task
			task()
		}
	}()
}

// ScheduleTaskWithSkipWeekends schedules a task to run at a specific time every weekday
func ScheduleTaskWithSkipWeekends(hour, minute, second int, task func()) {
	go func() {
		for {
			// Get the current time
			now := time.Now()

			// Calculate the next scheduled time
			nextRun := time.Date(
				now.Year(),
				now.Month(),
				now.Day(),
				hour,
				minute,
				second,
				0,
				now.Location(),
			)

			// If the next scheduled time is in the past, schedule for the next day
			if nextRun.Before(now) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			// Skip weekends (Saturday and Sunday)
			for nextRun.Weekday() == time.Saturday || nextRun.Weekday() == time.Sunday {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			// Wait until the next scheduled time
			time.Sleep(time.Until(nextRun))

			// Execute the task
			task()
		}
	}()
}

func downloadRutine(sqlitefile string, postgresDSN string, tasks chan string, date string) {
	updateEssentialInformation(sqlitefile, postgresDSN)
	if sqlitefile != "" || postgresDSN != "" {
		wg.Add(1)
		tasks = make(chan string, 16)
		go writingRoutine(tasks)
	}

	today = date

	count := 0
	for !checkToday() {
		log.Infoln("暫停1分鐘")
		time.Sleep(time.Minute)
		count++

		if count == 10 {
			// slackWebhook.SentMessage("今日無交易")
			return
		}
	}

	// slackWebhook.SentMessage("開始下載今日交易籌碼")

	createDir()
	start := time.Now()
	for _, s := range stocks {
		nodata = false
		downloadChip(&s.id)
		if !nodata && (sqlitefile != "" || postgresDSN != "") {
			tasks <- s.id
		}
		log.Infoln("暫停4~6秒")
		time.Sleep(time.Duration(4+rand.Intn(3)) * time.Second)
	}
	elapsed := time.Since(start)
	log.WithFields(log.Fields{
		"matchCount/requestImageCount": float64(matchCount) / float64(requestImageCount),
	}).Info("CAPTCHA 正確率")
	log.WithFields(log.Fields{
		"elapsed": elapsed,
	}).Printf("下載用時")
	// slackWebhook.SentMessage("下載今日交易籌碼成功")
	if sqlitefile != "" || postgresDSN != "" {
		tasks <- "close"
		wg.Wait()
	}
	err := gotool.CompressFolder(today)
	if err != nil {
		log.WithError(err).Warnln("壓縮檔案失敗")
	}
	// slackWebhook.SentMessage("歸檔今日交易籌碼成功")
}

func main() {
	runtime.GOMAXPROCS(2)
	const version = "v3.0.2"
	var tasks chan string
	sqlitefile := ""
	postgresconfig := ""
	postgresDSN := ""

	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "顯示版本",
	}

	app := &cli.App{
		Name:    "twstockchip",
		Usage:   "臺灣股市交易籌碼資料下載",
		Version: version,
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
			&cli.BoolFlag{
				Name:    "nowritesqlite",
				Aliases: []string{"n"},
				Usage:   "不寫入sqlite資料庫",
				Value:   false,
			},
			&cli.StringFlag{
				Name:        "sqlitefile",
				Aliases:     []string{"f"},
				Usage:       "指定sqlite數據庫檔案",
				Value:       "./twstockchip.sqlite",
				DefaultText: "./twstockchip.sqlite",
			},
			&cli.StringFlag{
				Name:        "postgresconfig",
				Aliases:     []string{"p"},
				Usage:       "指定postgres數據庫配置",
				Value:       "",
				DefaultText: "",
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
			if !c.Bool("nowritesqlite") {
				sqlitefile = c.String("sqlitefile")
			}
			postgresconfig = c.String("postgresconfig")
			if postgresconfig != "" {
				pconfig, err := config.LoadPostgreConfig(postgresconfig)
				if err != nil {
					log.WithError(err).Fatalln("讀取PostgreSQL配置失敗")
				} else {
					postgresDSN = model.BuildPostgresDSN(pconfig)
				}
			}

			return nil
		},

		Commands: []*cli.Command{
			{
				Name:    "rebuild",
				Aliases: []string{"r"},
				Usage:   "指定日期重新建立資料庫",
				Action: func(c *cli.Context) error {
					updateEssentialInformation(sqlitefile, postgresDSN)
					fileList, err := gotool.DirRegListFiles("./csv", "^[0-9]...-[0-1][0-9]-[0-3][0-9].tar.zst")
					if err != nil {
						log.WithError(err).Warnln("讀取檔案列表失敗")
					}
					reg, _ := regexp.Compile("[0-9]...-[0-1][0-9]-[0-3][0-9]")
					firstDate, _ := time.Parse("2006-01-02", c.String("date"))
					for _, fileName := range fileList {
						dateString := reg.FindString(*fileName)
						fileDate, _ := time.Parse("2006-01-02", dateString)
						if firstDate.Before(fileDate) || firstDate.Equal(fileDate) {
							err := gotool.UncompressFolder(fileName)
							if err != nil {
								log.WithError(err).Warnln("解壓縮檔案失敗")
							}
							folder := "./" + dateString
							csvFileList, err := gotool.DirRegListFiles(folder, ".*csv$")
							if err != nil {
								log.WithError(err).Warnln("讀取檔案列表失敗")
							}
							for _, csvFile := range csvFileList {
								csvNameSlice := strings.Split(*csvFile, "/")
								nameWithExtension := csvNameSlice[2]
								code := nameWithExtension[0 : len(nameWithExtension)-4]
								csvreader.ReadCSV(*csvFile, code, dateString)
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
					downloadRutine(sqlitefile, postgresDSN, tasks, c.String("date"))
					return nil
				},
			},
			{
				Name:    "daemon",
				Aliases: []string{"D"},
				Usage:   "每日自動下載交易籌碼",
				Action: func(c *cli.Context) error {
					log.Infoln("啟動每日自動下載交易籌碼服務")
					ScheduleTaskWithSkipWeekends(16, 30, 0, func() {
						requestImageCount = 0
						matchCount = 0
						downloadRutine(sqlitefile, postgresDSN, tasks, time.Now().Format("2006-01-02"))
					})
					// Keep the main function running
					select {}
					// return nil
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
