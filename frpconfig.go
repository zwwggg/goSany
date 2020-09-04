package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"gopkg.in/ini.v1"
)

var frpcFilePath = "/etc/frp/frpc.ini"
var cfg0FilePath = "/sys/fsl_otp/HW_OCOTP_CFG0"
var cfg1FilePath = "/sys/fsl_otp/HW_OCOTP_CFG1"
var version = "1.0.0"

func getUid(filePath string) string {
	cfg, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	str := string(cfg)[2:]
	str = strings.Replace(str, "\n", "", -1)
	str = strings.Replace(str, "\r", "", -1)

	for true {
		if len(str) >= 8 {
			break
		}
		str = "0" + str
	}

	return str
}

func fixFrpc(cfg *ini.File) {
	uuid := ""
	cfg.Section("common").Key("server_addr").SetValue("152.136.175.226")
	cfg.Section("common").Key("server_port").SetValue("8000")
	uid0, uid1 := getUid(cfg0FilePath), getUid(cfg1FilePath)
	if uid0 != "" && uid1 != "" {
		uuid = uid0 + uid1
		cfg.Section("common").Key("user").SetValue(uuid)
	}
	cfg.Section("common").Key("tls_enable").SetValue("true")
	cfg.Section("common").Key("token").SetValue("sysd")

	cfg.Section("sssh").Key("type").SetValue("stcp")
	cfg.Section("sssh").Key("sk").SetValue(getHmac("uuid", "sysd_sssh"))
	cfg.Section("sssh").Key("local_ip").SetValue("127.0.0.1")
	cfg.Section("sssh").Key("local_port").SetValue("22")

	cfg.Section("sscodesys").Key("type").SetValue("stcp")
	cfg.Section("sscodesys").Key("sk").SetValue(getHmac("uuid", "sysd_scodesys"))
	cfg.Section("sscodesys").Key("local_ip").SetValue("127.0.0.1")
	cfg.Section("sscodesys").Key("local_port").SetValue("1217")
}

func getHmac(s string, k string) string {
	h := hmac.New(sha256.New, []byte(k))
	io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

var ver = flag.Bool("v", false, "version")

func main() {
	//返回软件版本
	flag.Parse()
	if *ver {
		fmt.Println(version)
		return
	}

	cfg := ini.Empty()

	fixFrpc(cfg)
	cfg.SaveTo(frpcFilePath)
}
