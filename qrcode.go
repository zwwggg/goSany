package main

import (
	"flag"
	"fmt"
	"github.com/skip2/go-qrcode"
)

var content = flag.String("c", "", "二维码包含的信息")
var size = flag.Int("s", 256, "二维码图片尺寸")
var name = flag.String("n", "", "生成文件的名字和路径")

func main()  {
	flag.Parse()
	err := qrcode.WriteFile(*content, qrcode.Medium, *size, *name)
	if err != nil {
		fmt.Println(err)
	}
}
