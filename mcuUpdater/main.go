package main

import (
	"flag"
	"fmt"
	"log"
)

var cmdUpload = flag.String("u", "", "指定上传文件")
var cmdDownload = flag.String("d", "", "指定下载文件")
var version = flag.CommandLine.Bool("v", false, "软件版本")

func init() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Lshortfile)
}

func main() {
	var (
		filename string
		mode     string
	)

	if *version {
		fmt.Println("1.1.0")
		return
	}

	if *cmdDownload != "" {
		mode = "download"
		filename = *cmdDownload
	} else if *cmdUpload != "" {
		mode = "upload"
		filename = *cmdUpload
	}

	if filename != "" {
		updater := NewMcuUpdater(filename, mode)
		updater.RunUdpServer()
	} else {
		log.Println("no input params, please use '-v'")
	}
}
