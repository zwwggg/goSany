package main

import (
	"fmt"
	"gopkg.in/ini.v1"
	"io/ioutil"
	"os"
)

var frpcFilePath = "/etc/frp/frpc.ini"

func getUid() string {
	cfg0, err := ioutil.ReadFile("/sys/fsl_otp/HW_OCOTP_CFG0")
	if err != nil {
		fmt.Println(err)
		return ""
	}

	cfg1, err := ioutil.ReadFile("/sys/fsl_otp/HW_OCOTP_CFG1")
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return string(cfg0)[2:10] + string(cfg1)[2:10]
}

func fixFrpc(cfg *ini.File) {
	cfg.Section("common").Key("server_addr").SetValue("152.136.175.226")
	cfg.Section("common").Key("server_port").SetValue("8000")
	if uid := getUid(); uid != "" {
		cfg.Section("common").Key("user").SetValue(uid)
	}
	cfg.Section("common").Key("tls_enable").SetValue("true")
	cfg.Section("common").Key("token").SetValue("sysd")
	cfg.Section("common").Key("log_file").SetValue("/etc/frp/frpc.log")
	cfg.Section("common").Key("log_level").SetValue("info")
	cfg.Section("common").Key("log_max_days").SetValue("60")

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
