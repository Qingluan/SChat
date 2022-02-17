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
	CL := 36
	// Create a colored image of the given width and height.
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	// i := 0
	l := len(vps.myhome)
	// last := uint8(0)
	for y := 0; y < height; y++ {
		// fs := []uint8{}
		for x := 0; x < width; x++ {
			xn := x / CL
			yn := y / CL
			c := vps.myhome[(yn*width+xn)%l]
			R := uint8(((xn + yn) ^ int(c)) % 256)
			G := uint8(((xn+yn)<<1 ^ int(c)) % 256)
			B := uint8(((xn+yn)<<2 ^ int(c)) % 256)
			// fmt.Println("x:", xn, "y:", yn, "c:", R)
			// fs = append(fs, R)
			img.Set(x, y, color.NRGBA{
				// Y: R,
				R: R,
				G: G,
				B: B,
				A: 255,
			})
		}
		// for _, i := range fs {
		// 	fmt.Print(i, " ")
		// }
		// fmt.Println()
	}

	f := bytes.NewBuffer([]byte{})

	if err := png.Encode(f, img); err != nil {
		return nil, err
	}

	return f.Bytes(), nil
}

func (chat *ChatRoom) GetTalkerSIcon(name ...string) (buf []byte, err error) {
	if chat.nowMsgTo == "" && name == nil {
		return nil, fmt.Errorf("no talker setting !!")
	}
	author := chat.nowMsgTo
	iconPath := Join(ROOT, chat.vps.msgto, MSG_FILE_ROOT, MSG_ICON)

	if name != nil {
		author = name[0]
		iconPath = Join(ROOT, chat.vps.E(name[0]), MSG_FILE_ROOT, MSG_ICON)
	}

	buffer := bytes.NewBuffer([]byte{})
	err = chat.vps.WithSftpRead(iconPath, os.O_RDONLY, func(fp io.ReadCloser) error {
		stream, err := NewStreamWithAuthorNoSave(author, false)
		if err != nil {
			log.Println("load straem err:", err)
			return err
		}
		stream.StreamDecrypt(buffer, fp, func(updated int64) {
			if updated%(1024*1024) == 0 && updated != 0 {
				log.Println("encrypted upload "+iconPath+" :", updated/1024/1024, "MB")
			}
		})
		return nil
	})
	if err != nil {
		log.Println("Get talker's icon err:", err)
		return nil, err
	}

	return buffer.Bytes(), nil
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
		// log.Println("updalote end :", localPath)
		err = chat.vps.WithSendFileToOwn(localPath, func(networkFile io.Writer, rawFile io.Reader) (err error) {

			stream, err := NewStreamWithAuthor(chat.MyName, false)
			if err != nil {
				log.Println("load straem err:", err)
				return err
			}
			_, err = stream.StreamEncrypt(networkFile, rawFile, func(updated int64) {
				if updated%(1024*1024) == 0 && updated != 0 {
					log.Println("encrypted upload "+path+" :", updated/1024/1024, "MB")
				}
			})
			// fmt.Println("upload :", n)
			return err
		})
		if err != nil {
			fmt.Println("upload png err:", err)
			return err
		}
		log.Println("encrypted upload " + path + " ok")
		return nil
	} else {
		log.Println("not png end :", path)
	}
	return
}

func (chat *ChatRoom) GetMyIcon() (buf []byte, err error) {

	grouped := false
	author := chat.MyName
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
			err = chat.SetMyIcon(localPath)

			fmt.Println("no icon , so i generate one !!!", localPath)
			if err != nil {
				log.Println("upload my icon err:", err)
			}

			// os.Remove(localPath)
			return buf, err
		}

	}
	L("icon found!:%s | by [%s]", MSG_ICON, author)

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
	path := filepath.Join(HOME, ".sshchat", chat.MyName+".icon")
	if _, err := os.Stat(path); err != nil {
		chat.UpdateMyIconWithPath()
	}
	return path
}

func (chat *ChatRoom) UpdateMyIconWithPath() string {
	buf, err := chat.GetMyIcon()
	if err != nil {
		log.Println("get mycion err:", err)
		return ""
	}
	path := filepath.Join(HOME, ".sshchat", chat.MyName+".icon")
	ioutil.WriteFile(path, buf, os.ModePerm)
	return path
}

func (chat *ChatRoom) UpdateTalkerIconWithPath(name ...string) string {
	if chat.nowMsgTo == "" && name == nil {
		log.Println("no talker setting !!")
		return ""
	}
	path := filepath.Join(HOME, ".sshchat", chat.nowMsgTo+".icon")
	if name != nil {
		path = filepath.Join(HOME, ".sshchat", name[0]+".icon")
	}

	buf, err := chat.GetTalkerSIcon(name...)
	if err != nil {
		log.Println(err)
		return ""
	}
	err = ioutil.WriteFile(path, buf, os.ModePerm)
	if err != nil {
		log.Println(err)

	}
	return path
}

func (chat *ChatRoom) GetTalkerSIconPath(name ...string) (string, error) {
	if chat.nowMsgTo == "" && name == nil {
		return "", fmt.Errorf("no talker setting !!%s", "")
	}
	path := filepath.Join(HOME, ".sshchat", chat.nowMsgTo+".icon")
	if name != nil {
		path = filepath.Join(HOME, ".sshchat", name[0]+".icon")
		if !chat.ExistsUser(name[0]) {
			return "", fmt.Errorf("no such user:%s", name)
		}
	}
	if _, err := os.Stat(path); err != nil {
		p := chat.UpdateTalkerIconWithPath(name...)
		return p, nil
	}
	return path, nil

}
