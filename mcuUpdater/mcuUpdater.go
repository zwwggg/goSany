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

	//升级命令 McuUpdateStatus
	mcuUpdateTypeError    = 0 //传输错误
	mcuUpdateTypeDownload = 1 //请求下载
	mcuUpdateTypeUpload   = 2 //请求上传
	mcuUpdateTypeSend     = 3 //数据传输
	mcuUpdateTypeComplete = 4 //传输完成

	//
	sizeOfSection = 128

	targetDevice  = "127.0.0.1"
	currentDevice = "127.0.0.1"
)

type McuUpdater struct {
	McuUpdateStatus uint8
	File            *os.File
	FileMaxPackage  int64
	udpConn         *net.UDPConn
	udpaddr         *net.UDPAddr
	TimerReq        *time.Ticker
	IsFinished      bool
	updatePercent   int64
}

func NewMcuUpdater() *McuUpdater {
	var (
		err     error
		fs      os.FileInfo
		mcuFile string
	)
	m := &McuUpdater{}
	m.McuUpdateStatus = mcuUpdateTypeComplete
	if m.File != nil {
		m.File.Close()
	}
	if *updateFileName != "" {
		mcuFile = *updateFileName
	} else {
		mcuFile = "MCU_APP.bin"
	}
	m.File, err = os.Open(mcuFile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	fs, err = os.Stat(mcuFile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	m.FileMaxPackage = (fs.Size() + sizeOfSection - 1) / sizeOfSection
	m.updatePercent = 0
	m.IsFinished = false
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

func (m *McuUpdater) Send(buffer []byte) {
	if m.udpConn != nil && m.udpaddr != nil {
		m.udpConn.WriteToUDP(m.fixPayload(buffer), m.udpaddr)
	}
}

func (m *McuUpdater) RequestDownload() {
	m.TimerReq = time.NewTicker(500 * time.Millisecond)

	go func() {
		for {
			select {
			case <-m.TimerReq.C:
				if m.udpConn != nil && m.udpaddr != nil {
					fmt.Println("request download")
					m.udpConn.WriteToUDP([]byte("uart"), m.udpaddr)
					m.Send([]byte{mcuUpdateTypeDownload << 5})
				}
			}
		}
	}()

}

func (m *McuUpdater) sendPackage(num int64) (bool, error) {
	if m.File == nil {
		return false, errors.New("no valid file, please check...")
	}

	buffer := make([]byte, sizeOfSection)

	offset := num * sizeOfSection
	m.File.Seek(offset, 0)
	n, err := m.File.Read(buffer)
	if err != nil {
		log.Fatalln(err.Error())
	}

	packageNum := []byte{byte(num >> 8), byte(num)}
	buffer = append(packageNum, buffer...)
	buffer = append([]byte{mcuUpdateTypeSend << 5}, buffer...)
	m.Send(buffer)

	//send data
	// log.Printf("%x\n", buffer)

	if n == 0 {
		fmt.Println("at the end of file")
		return true, nil
	}

	if n != sizeOfSection {
		m.File.Close()
		return true, nil
	}

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
	defer m.File.Close()

	fmt.Println("udp listening ...")

	m.RequestDownload()

	//udp不需要Accept
	for {
		m.handleConnection(m.udpConn)

		if m.IsFinished {
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
	case mcuUpdateTypeDownload:
		if m.McuUpdateStatus == mcuUpdateTypeComplete || m.McuUpdateStatus == mcuUpdateTypeError {
			fmt.Println("mcu_update_type_download")
			m.McuUpdateStatus = mcuUpdateTypeDownload
			m.TimerReq.Stop()
		}
	case mcuUpdateTypeUpload:
		fmt.Println("mcu_update_type_upload")
	case mcuUpdateTypeSend:
		if m.McuUpdateStatus != mcuUpdateTypeSend && m.McuUpdateStatus != mcuUpdateTypeDownload {
			return
		} else {
			// log.Println("mcu_update_type_send")
			m.McuUpdateStatus = mcuUpdateTypeSend

			num := int64(buf[2]) | (int64(buf[1]) << 8)
			isFinish, _ := m.sendPackage(num)

			currentPercent := (num + 1) * 100 / m.FileMaxPackage
			if currentPercent != m.updatePercent {
				m.updatePercent = currentPercent
				fmt.Printf("\rmcu update percent:%d%%", currentPercent)
			}

			if isFinish {
				m.Send([]byte{mcuUpdateTypeComplete << 5})
				m.IsFinished = true
				fmt.Println("\nfinished ...")
			}
		}
	case mcuUpdateTypeComplete:
	case mcuUpdateTypeError:
		m.McuUpdateStatus = uint8(updateType)
	default:

	}
}
