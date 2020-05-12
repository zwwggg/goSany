package main

import (
	"fmt"
	"gopkg.in/ini.v1"
	"io/ioutil"
	"os"
	"strings"
)

var frpcFilePath = "/etc/frp/frpc.ini"
var cfg0FilePath = "/sys/fsl_otp/HW_OCOTP_CFG0"
var cfg1FilePath = "/sys/fsl_otp/HW_OCOTP_CFG1"

func getUid(filePath string) string {
	cfg, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	str := string(cfg)[2:]
	str = strings.Replace(str, "\n", "",-1)
	str = strings.Replace(str, "\r", "",-1)

	for true {
		if len(str) >= 8 {
			break
		}
		str = "0" + str
	}

	return str
}

func fixFrpc(cfg *ini.File) {
	cfg.Section("common").Key("server_addr").SetValue("152.136.175.226")
	cfg.Section("common").Key("server_port").SetValue("8000")
	uid0, uid1 := getUid(cfg0FilePath), getUid(cfg1FilePath);
	if uid0 != "" && uid1 != "" {
		cfg.Section("common").Key("user").SetValue(uid0 + uid1)
	}
	cfg.Section("common").Key("tls_enable").SetValue("true")
	cfg.Section("common").Key("token").SetValue("sysd")

	cfg.Section("ssh").Key("type").SetValue("tcp")
	cfg.Section("ssh").Key("local_ip").SetValue("127.0.0.1")
	cfg.Section("ssh").Key("local_port").SetValue("22")
	cfg.Section("ssh").Key("remote_port").SetValue("0")
	cfg.Section("ssh").Key("use_encryption").SetValue("true")
	cfg.Section("ssh").Key("use_compression").SetValue("true")

	cfg.Section("codesys").Key("type").SetValue("tcp")
	cfg.Section("codesys").Key("local_ip").SetValue("127.0.0.1")
	cfg.Section("codesys").Key("local_port").SetValue("1217")
	cfg.Section("codesys").Key("remote_port").SetValue("0")
	cfg.Section("codesys").Key("use_encryption").SetValue("true")
	cfg.Section("codesys").Key("use_compression").SetValue("true")
}

func main() {
	var cfg *ini.File
	if _, err := os.Stat(frpcFilePath); err == nil { //文件存在
		cfg, err = ini.Load(frpcFilePath)

		if err != nil {
			fmt.Println(err)
			return
		}
	} else if os.IsNotExist(err) { //文件不存在
		cfg = ini.Empty()
	} else { //未知错误
		fmt.Println(err)
		return
	}

	fixFrpc(cfg)
	cfg.SaveTo(frpcFilePath)
}
