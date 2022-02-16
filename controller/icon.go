package controller

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func (vps *Vps) GenerateIcon() (buf []byte, err error) {
	const width, height = 180, 180
	// Create a colored image of the given width and height.
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	// i := 0
	l := len(vps.myhome)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := vps.myhome[(y*width+x)%l]
			img.Set(x, y, color.NRGBA{
				R: uint8(((x + y) ^ int(c)) % 256),
				G: uint8(((x+y)<<1 ^ int(c)) % 256),
				B: uint8(((x+y)<<2 ^ int(c)) % 256),
				A: 255,
			})
		}
	}

	f := bytes.NewBuffer([]byte{})

	if err := png.Encode(f, img); err != nil {
		return nil, err
	}

	return f.Bytes(), nil
}

func (chat *ChatRoom) SetMyIcon(path string) (err error) {
	if strings.HasSuffix(path, ".png") {
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		localPath := filepath.Join(os.TempDir(), MSG_ICON)
		err = ioutil.WriteFile(localPath, buf, os.ModePerm)
		if err != nil {
			return err
		}
		err = chat.vps.WithSendFileToOwn(localPath, func(networkFile io.Writer, rawFile io.Reader) (err error) {

			stream, err := NewStreamWithAuthor(chat.vps.E(chat.MyName), false)
			if err != nil {
				log.Println("load straem err:", err)
				return err
			}
			stream.StreamEncrypt(networkFile, rawFile, func(updated int64) {
				if updated%(1024*1024) == 0 && updated != 0 {
					log.Println("encrypted upload "+path+" :", updated/1024/1024, "MB")
				}
			})
			return nil
		})
		if err != nil {
			return err
		}
		log.Println("encrypted upload " + path + " ok")
		return nil
	}
	return
}

func (chat *ChatRoom) GetMyIcon() (buf []byte, err error) {

	grouped := false
	author := chat.vps.myenname
	buffer := bytes.NewBuffer([]byte{})

	if exists, _ := chat.vps.Exists(MSG_ICON); !exists {
		if buf, err := chat.vps.GenerateIcon(); err != nil {
			return nil, err
		} else {
			localPath := filepath.Join(os.TempDir(), "init.png")
			err = ioutil.WriteFile(localPath, buf, os.ModePerm)
			if err != nil {
				return nil, err
			}
			fmt.Println("no icon , so i generate one !!!")
			return buf, nil
		}

	}

	err = chat.vps.DownloadCloud(MSG_ICON, func(networkFile io.Reader) (err error) {
		stream, err := NewStreamWithAuthor(author, grouped)
		if err != nil {
			log.Println("load straem err:", err)
			return err
		}

		if err != nil {
			log.Println("create local file err:", err)
			return
		}

		stream.StreamDecrypt(buffer, networkFile, func(downloaded int64) {
			if downloaded%(1024*1024) == 0 && downloaded != 0 {
				log.Println("encrypted download "+MSG_ICON+" :", downloaded/1024/1024, "MB")
			}
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (chat *ChatRoom) GetMyIconWithPath() string {
	buf, err := chat.GetMyIcon()
	if err != nil {
		log.Println("get mycion err:", err)
		return ""
	}
	path := filepath.Join(HOME, ".sshchat", "my.icon")
	ioutil.WriteFile(path, buf, os.ModePerm)
	return path
}
