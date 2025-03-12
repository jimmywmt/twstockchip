# twstockship

<!--toc:start-->

- [twstockship](#twstockship)
- [NAME](#name)
- [USAGE](#usage)
- [VERSION](#version)
- [COMMANDS](#commands)
- [GLOBAL OPTIONS](#global-options)
<!--toc:end-->

## NAME

twstockship - 臺灣股市交易籌碼資料下載

## USAGE

twstockship [global options] command [command options] [arguments...]

## VERSION

v3.0.2

## COMMANDS

daemon, D 每日自動下載交易籌碼  
download, d 下載指定日期交易籌碼 (需交易所網頁釋出)  
rebuild, r 指定日期重新建立資料庫  
help, h Shows a list of commands or help for one command

## GLOBAL OPTIONS

--date value, -d value 指定日期 (format 2016-01-02) (default: 2025-03-12)  
--loglevel value, -l value 設定log等級 (debug, info, warm, error, fatal, panic)
(default: info)  
--nowritesqlite, -n 不寫入sqlite資料庫 (default: false)  
--postgresconfig value, -p value 指定postgres數據庫配置  
--sqlitefile value, -f value 指定sqlite數據庫檔案 (default: ./twstockchip.sqlite)  
--help, -h show help  
--version, -v 顯示版本 (default: false)
