package main

import (
	"flag"
	"fmt"
	"log"
)

var updateFileName = flag.String("n", "", "指定升级文件")
var version = flag.CommandLine.Bool("v", false, "软件版本")

func init() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Lshortfile)
}

func main() {
	if *version {
		fmt.Println("1.0.0")
		return
	}

	updater := NewMcuUpdater()
	updater.RunUdpServer()
}
