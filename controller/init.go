package controller

import (
	"encoding/base64"
	"fmt"
	"log"
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
	MSG_ICON = steam.FlowEn(MSG_ICON)
	GROUP_TAIL = steam.FlowEn(GROUP_TAIL)
	MSG_TMP_FILE = steam.FlowEn(MSG_TMP_FILE)
	vps.myhome = Join(ROOT, steam.FlowEn(name))
	vps.myenname = steam.FlowEn(name)
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

func (chat *ChatRoom) Login(restoresKey ...string) (logined bool) {
	if restoresKey != nil && restoresKey[0] != "" {
		chat.RestoreKeyFromServer(restoresKey[0])
	}
	var err error
	chat.stream, err = NewStreamWithAuthor(chat.vps.name, false)
	if err != nil {
		log.Println("chat logined failed: ", err)
		return
	}
	if err := chat.vps.Init(); err != nil {
		log.Println("chat logined failed: ", err)
		return
	} else {
		if restoresKey != nil && restoresKey[0] != "" {
			fmt.Println("share key in remote:", chat.SaveKeyToServer(restoresKey[0]))
		}
		return true
	}

}

func (vps *Vps) Init() (err error) {
	vps.Security()
	if !vps.CreateMe() {
		return fmt.Errorf("login failed: user already exists but key is err!%s", "")
	}
	vps.heartInterval = 1
	vps.liveInterval = 1200
	go vps.HeartBeat()
	go vps.backgroundRecvMsgs()
	return
}
