package controller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

func (vps *Vps) SendKeyReq() (err error) {
	date := time.Now().Format(TIME_TMP)
	if vps.msgto == "" {
		return fmt.Errorf("offline/no set user")
	}
	msgpath := Join(ROOT, vps.msgto, MSG_FILE)
	err = vps.WithSftpWrite(msgpath, os.O_RDWR|os.O_APPEND|os.O_CREATE, func(fp io.WriteCloser) error {
		d := Message{
			Date: fmt.Sprint(date),
			Data: "",
			From: "${no-key}:" + vps.name,
		}
		data, _ := json.Marshal(&d)
		_, e := fp.Write([]byte(string(data) + "\n\r"))
		return e
	})
	L("pull %s(%s)'s key", vps.msgto, vps.D(vps.msgto))
	return
}

func (vps *Vps) SendKeyTo(target, key string, grouped ...string) (err error) {
	// fmt.Println("test:", vps.name, "send key to", target)
	date := time.Now().Format(TIME_TMP)
	// if vps.msgto == "" {
	// 	return fmt.Errorf("offline/no set user")
	// }
	kk := "${key}:"
	group := ""
	if grouped != nil {
		kk = "${group-key}:"
		group = grouped[0]
	}
	msgpath := Join(ROOT, vps.E(target), MSG_FILE)
	err = vps.WithSftpWrite(msgpath, os.O_RDWR|os.O_APPEND|os.O_CREATE, func(fp io.WriteCloser) error {
		d := Message{
			Date:  fmt.Sprint(date),
			Data:  key,
			Group: group,
			From:  kk + vps.name,
		}
		data, _ := json.Marshal(&d)
		_, e := fp.Write([]byte(string(data) + "\n\r"))
		return e
	})
	L("i share my key to: %s(%s)", vps.E(target), target)
	return
}

func (vps *Vps) TryRestoreKey(key string) bool {
	stearm, err := NewStreamWithBase64Key(base64.StdEncoding.EncodeToString([]byte(vps.IP + key)))
	if err != nil {
		log.Fatal("restore init kjey err :!!!!")
	}
	remotekeyName := Join(TMP, stearm.FlowEn(vps.name))
	// var plainkey []byte

	vps.loginpwd = key
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
