package controller

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func (vps *Vps) Security() {
	sshstr := vps.IP
	name := vps.name

	key := base64.StdEncoding.EncodeToString([]byte(sshstr))
	// fmt.Println("[debug]:", key)
	steam, err := NewStreamWithBase64Key(key)
	if err != nil {
		log.Fatal("base Login Ssh as crypter err ")
	}
	base_root := steam.FlowEn(ROOT)

	ROOT = Join("/tmp", base_root)
	MSG_HISTORY = steam.FlowEn(MSG_HISTORY)
	MSG_FILE = steam.FlowEn(MSG_FILE)
	MSG_HELLO = steam.FlowEn(MSG_HELLO)
	MSG_FILE_ROOT = steam.FlowEn(MSG_FILE_ROOT)
	MSG_HEART = steam.FlowEn(MSG_HEART)
	MSG_PRIVATE_KEY = steam.FlowEn(MSG_PRIVATE_KEY)
	GROUP_TAIL = steam.FlowEn(GROUP_TAIL)
	MSG_TMP_FILE = steam.FlowEn(MSG_TMP_FILE)
	vps.myhome = Join(ROOT, steam.FlowEn(name))
	vps.steam = steam
}

func (chat *ChatRoom) Init(sshstr string) (err error) {
	chat.vps = Parse(sshstr)
	SecurityCheckName(chat.vps.name)
	chat.IP = chat.vps.IP
	chat.recvMsg = make(chan *Message, 1024)
	chat.MyName = chat.vps.name
	return
}

func (vps *Vps) SetMsgTo(name string) {
	vps.msgto = vps.steam.FlowEn(name)
}

func (vps *Vps) GetRawMsgTo() string {
	return vps.steam.FlowDe(vps.msgto)
}

func (vps *Vps) GetVpsName() string {
	return vps.steam.FlowEn(vps.name)
}

func (vps *Vps) TryRestoreKey(key string) bool {
	stearm, err := NewStreamWithBase64Key(base64.StdEncoding.EncodeToString([]byte(vps.IP + key)))
	if err != nil {
		log.Fatal("restore init kjey err :!!!!")
	}
	remotekeyName := Join(TMP, stearm.FlowEn(vps.name))
	// var plainkey []byte
	err = vps.WithSftpRead(remotekeyName, os.O_RDONLY, func(fp io.ReadCloser) error {
		buf, err := ioutil.ReadAll(fp)
		if err != nil {
			return err
		}
		fs := strings.Fields(string(buf))
		if len(fs) == 0 {
			return fmt.Errorf("no key found in remote !!!")
		}
		plainkey := stearm.FlowDe(strings.TrimSpace(fs[0]))
		if len(plainkey) > 0 {
			SetKey(vps.name, string(plainkey))
		}
		return nil
	})
	if err != nil {
		log.Println("restore failed !:", err)
		return false
	}
	return true
}

func (vps *Vps) SaveKeyToServer(key string) (ok bool) {
	stearm, err := NewStreamWithBase64Key(base64.StdEncoding.EncodeToString([]byte(vps.IP + key)))
	if err != nil {
		log.Fatal("restore init kjey err :!!!!")
	}
	remotekeyName := Join(TMP, stearm.FlowEn(vps.name))
	// remotekeyName := stearm.FlowEn(vps.name)
	vps.WithSftpWrite(remotekeyName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, func(fp io.WriteCloser) error {
		myKey := GetKey(vps.name)
		randomKey := NewKey()
		entrys := stearm.FlowEn(myKey)
		fp.Write([]byte(entrys + " " + randomKey))
		ok = true
		return nil
	})
	return
}
