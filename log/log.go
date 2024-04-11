package log

import (
	"log"
	"os"
	"time"
)

var (
	// 日志文件名
	filename = time.Now().Format("2006-01-02") + ".txt"
	Info     *log.Logger
	Error    *log.Logger
)

func init() {
	// 提前关闭文件会导致写不进去日志文件中
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Faild to open error logger file:", err)
	}
	//自定义日志格式
	Info = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	return
}
