package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

const (
	//mcu升级标识符
	prtclTypeMcuUpdate = 0x58

	//升级命令 status
	mcuUpdateTypeError    = 0 //传输错误
	mcuUpdateTypeDownload = 1 //请求下载
	mcuUpdateTypeUpload   = 2 //请求上传
	mcuUpdateTypeSend     = 3 //数据传输
	mcuUpdateTypeComplete = 4 //传输完成

	//
	sizeOfDownloadSection = 256
	sizeOfUploadSection   = 128
	sizeOfUploadCnt       = (256 - 32) * 1024 / sizeOfUploadSection

	targetDevice  = "127.0.0.1"
	currentDevice = "127.0.0.1"
)

type McuUpdater struct {
	status         uint8        //状态
	mode           string       //模式，download, upload
	file           *os.File     //打开或创建的文件
	fileMaxPackage int16        //最大文件包数
	udpConn        *net.UDPConn //udp连接
	udpaddr        *net.UDPAddr //udp地址
	timerReq       *time.Ticker //定时请求
	timerUpload    *time.Ticker //定时上传请求
	isFinished     bool         //是否完成
	updatePercent  int16        //完成率
	updateCnt      int16        //完成包数
}

func NewMcuUpdater(filename, mode string) *McuUpdater {
	var (
		err error
		fs  os.FileInfo
	)

	m := &McuUpdater{}
	if filename == "" || mode == "" {
		return m
	}

	m.status = mcuUpdateTypeComplete
	m.mode = mode
	if m.file != nil {
		m.file.Close()
	}

	if m.mode == "download" {
		m.file, err = os.Open(filename)
	} else if m.mode == "upload" {
		m.file, err = os.Create(filename)
	}
	if err != nil {
		log.Fatalln(err.Error())
	}

	fs, err = os.Stat(filename)
	if err != nil {
		log.Fatalln(err.Error())
	}

	if m.mode == "download" {
		m.fileMaxPackage = int16((fs.Size() + sizeOfDownloadSection - 1) / sizeOfDownloadSection)
	} else if m.mode == "upload" {
		m.fileMaxPackage = sizeOfUploadCnt
	}
	m.updatePercent = 0
	m.isFinished = false
	m.udpaddr, _ = net.ResolveUDPAddr("udp4", targetDevice+":10319")
	return m
}

func (m *McuUpdater) fixPayload(b []byte) []byte {
	payload := make([]byte, 5)
	l := len(b)
	payload[0] = 0x5a
	payload[1] = 0xa5
	payload[2] = prtclTypeMcuUpdate
	payload[3] = byte(l >> 8)
	payload[4] = byte(l)
	payload = append(payload, b...)
	crc := CheckSum(payload)
	payload = append(payload, byte(crc>>8))
	payload = append(payload, byte(crc))

	return payload
}

func (m *McuUpdater) send(buffer []byte) {
	if m.udpConn != nil && m.udpaddr != nil {
		m.udpConn.WriteToUDP(m.fixPayload(buffer), m.udpaddr)
	}
}

func (m *McuUpdater) sendUploadRequest(num int16) {
	buf := make([]byte, 3)
	buf[0] = mcuUpdateTypeSend << 5
	buf[1] = byte(num >> 8)
	buf[2] = byte(num & 0xff)
	m.send(buf)
}

func (m *McuUpdater) udpRequest() {
	m.timerReq = time.NewTicker(500 * time.Millisecond)
	m.timerUpload = time.NewTicker(100 * time.Millisecond)
	m.timerUpload.Stop()

	go func() {
		for {
			select {
			case <-m.timerReq.C:
				if m.udpConn != nil && m.udpaddr != nil {
					m.udpConn.WriteToUDP([]byte("uart"), m.udpaddr)
					if m.mode == "download" {
						fmt.Println("request download")
						m.send([]byte{mcuUpdateTypeDownload << 5})
					} else if m.mode == "upload" {
						fmt.Println("request upload")
						m.send([]byte{mcuUpdateTypeUpload << 5})
					}
				}
			}
		}
	}()

	if m.mode == "upload" {
		go func() {
			for {
				select {
				case <-m.timerUpload.C:
					if m.udpConn != nil && m.udpaddr != nil {
						m.sendUploadRequest(m.updateCnt)
					}
				}
			}
		}()
	}
}

func (m *McuUpdater) sendDownloadPackage(num int16) (bool, error) {
	if m.file == nil {
		return false, errors.New("no valid file, please check...")
	}

	if num >= m.fileMaxPackage {
		// m.file.Close()
		return true, nil
	}

	buffer := make([]byte, sizeOfDownloadSection)

	offset := int64(num * sizeOfDownloadSection)
	m.file.Seek(offset, 0)
	n, _ := m.file.Read(buffer)
	if n == 0 {
		fmt.Println("out the range of file")
	}

	packageNum := []byte{byte(num >> 8), byte(num)}
	buffer = append(packageNum, buffer...)
	buffer = append([]byte{mcuUpdateTypeSend << 5}, buffer...)
	m.send(buffer)

	//send data
	// log.Printf("%x\n", buffer)
	return false, nil
}

// udp 服务端
func (m *McuUpdater) RunUdpServer() {
	udpAddr, _ := net.ResolveUDPAddr("udp4", currentDevice+":10316")

	//监听端口
	var err error
	m.udpConn, err = net.ListenUDP("udp4", udpAddr)
	if err != nil {
		fmt.Println(err)
	}
	defer m.udpConn.Close()
	defer m.file.Close()

	fmt.Println("udp listening ...")

	m.udpRequest()

	//udp不需要Accept
	for {
		m.handleConnection(m.udpConn)

		if m.isFinished {
			break
		}
	}
}

// 处理接收
func (m *McuUpdater) handleConnection(udpConn *net.UDPConn) {
	// 读取数据
	buf := make([]byte, 1024)
	var l int
	var err error
	l, _, err = udpConn.ReadFromUDP(buf)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// 数据有效性验证
	buf = buf[:l]
	if buf[0] != 0x5a || buf[1] != 0xa5 || len(buf) < 7 {
		return
	}

	// fmt.Printf("%02x\n", buf)

	//数据长度验证
	payloadLen := int(buf[4]) + int(buf[3])<<8
	if payloadLen != (len(buf) - 7) {
		return
	}

	//数据校验验证
	recvCrc := uint16(buf[l-1]) + uint16(buf[l-2])<<8
	crc := CheckSum(buf[:l-2])
	if crc != recvCrc {
		return
	}

	//数据帧类型验证
	if buf[2] != byte(prtclTypeMcuUpdate) {
		return
	}
	buf = buf[5 : l-2]
	// fmt.Printf("recv valid payload: %x, len: %d\n", buf, l)

	updateType := int(buf[0] >> 5)
	switch updateType {
	case mcuUpdateTypeSend:
		m.status = mcuUpdateTypeSend
		num := int16(buf[2]) | (int16(buf[1]) << 8)

		if m.mode == "download" {
			m.timerReq.Stop()
			m.updateCnt = num
			isFinish, _ := m.sendDownloadPackage(m.updateCnt)

			//进度展示
			currentPercent := int16((int64(m.updateCnt) + 1) * 100 / int64(m.fileMaxPackage))
			if currentPercent != m.updatePercent {
				m.updatePercent = currentPercent
				//fmt.Printf("\rmcu update num: %d, maxnum: %d, percent:%d%%", num, m.fileMaxPackage, currentPercent)
			}
			fmt.Printf("\rmcu update num: %d, maxnum: %d, percent:%d%%", num, m.fileMaxPackage, currentPercent)

			if isFinish {
				m.send([]byte{mcuUpdateTypeComplete << 5})
			}
		} else if m.mode == "upload" {
			if len(buf) != (sizeOfUploadSection+3) && num != 0 {
				fmt.Println("data package length incorrect")
				break
			}
			m.timerReq.Stop()

			//mcu返回的第一帧不包含数据，不做处理
			if len(buf) == (sizeOfUploadSection + 3) {
				m.timerUpload.Stop()
				m.updateCnt++
				m.file.Write(buf[3:])
			}

			//进度展示
			currentPercent := int16((int64(m.updateCnt) * 100) / sizeOfUploadCnt)
			if currentPercent != m.updatePercent {
				m.updatePercent = currentPercent
				fmt.Printf("\rmcu upload percent:%d%%", currentPercent)
			}

			//数据接收完成
			if m.updateCnt == sizeOfUploadCnt {
				m.send([]byte{mcuUpdateTypeComplete << 5})
			}

			//请求下一帧
			m.sendUploadRequest(m.updateCnt)
			m.timerUpload.Reset(100 * time.Millisecond)
		}
	case mcuUpdateTypeComplete:
		m.isFinished = true
		m.file.Close()
		if m.mode == "download" {
			fmt.Println("\ndownload finished ...")
		} else if m.mode == "upload" {
			fmt.Println("\nupload finished ...")
		}

	case mcuUpdateTypeError:
		m.status = uint8(updateType)
	default:
	}
}
