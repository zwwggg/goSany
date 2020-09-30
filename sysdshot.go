package main

import (
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	// Uncomment to store output in variable
	//"bytes"
)

func main() {
	username := "root"
	password := "sysd2020"
	hostname := "192.168.1.136"
	port := "22"

	// SSH client config
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		// Non-production only
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to host
	conn, err := ssh.Dial("tcp", hostname+":"+port, config)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// create new SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// create destination file
	// dstFile, err := os.Create("screenshot")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer dstFile.Close()

	// open source file
	srcFile, err := client.Open("/dev/fb0")
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()

	// copy source file to destination file
	// bytes, err := io.Copy(dstFile, srcFile)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("%d bytes copied\n", bytes)

	// // flush in-memory copy
	// err = dstFile.Sync()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// dstFile, _ = os.Open("screenshot")

	data, err := ioutil.ReadAll(srcFile)
	if err != nil {
		fmt.Println(err.Error())
	}

	pngfile := image.NewRGBA(image.Rect(0, 0, 1280, 768))
	pngfile.Pix = data
	for x := 0; x < 1280; x++ {
		for y := 0; y < 768; y++ {
			c := pngfile.RGBAAt(x, y)
			c.R, c.B = c.B, c.R
			pngfile.SetRGBA(x, y, c)
		}
	}

	file2, err := os.Create("screenshot.png")
	defer file2.Close()
	if err != nil {
		fmt.Println(err.Error())
	}

	png.Encode(file2, pngfile)
}
