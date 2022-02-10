package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	msgpath := Join(ROOT, target, MSG_FILE)
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
	return
}
