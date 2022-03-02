### NAME:
        twstockship - 臺灣股市交易籌碼資料下載

### USAGE:
        twstockship [global options] command [command options] [arguments...]

### VERSION:
            v1.0.0

### COMMANDS:
        download, d  下載指定日期交易籌碼 (需交易所網頁釋出)
        rebuild, r   指定日期重新建立資料庫
        help, h      Shows a list of commands or help for one command

### GLOBAL OPTIONS:
        --date value, -d value      指定日期 (format 2016-01-02) (default: 2022-03-02)
        --dbfile value, -f value    指定sqlite數據庫檔案 (default: ./twstockship.sqlite)
        --loglevel value, -l value  設定log等級 (debug, info, warm, error, fatal, panic) (default: info)
        --nowritedb, -n             不寫入sqlite資料庫 (default: false)
        --help, -h                  show help (default: false)
        --version, -v               顯示版本 (default: false

