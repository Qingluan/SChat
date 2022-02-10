package controller

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Qingluan/FrameUtils/utils"
	"github.com/pkg/sftp"
)

func (vps *Vps) SendGroupMsg(groupName string, msg string, encrypted ...bool) (err error) {
	date := time.Now().Format(TIME_TMP)
	raw := groupName

	groupName = vps.GetGroupVpsName(groupName)
	// fmt.Println("g:", groupName)
	existsGroup, _ := vps.GroupCheck(groupName)
	// fmt.Println("g3:", groupName)
	if !existsGroup {
		return fmt.Errorf("no such gorup :%s", raw)
	}
	msgpath := Join(ROOT, groupName, MSG_FILE)
	err = vps.WithSftpWrite(msgpath, os.O_RDWR|os.O_APPEND|os.O_CREATE, func(fp io.WriteCloser) error {
		d := Message{
			Date:  fmt.Sprint(date),
			Data:  msg,
			Tp:    MSG_TP_GROUP,
			From:  vps.E(vps.name),
			To:    vps.msgto,
			Group: groupName,
		}
		if encrypted != nil && encrypted[0] {
			d.Crypted = true
		}
		data, _ := json.Marshal(&d)
		_, e := fp.Write([]byte(string(data) + "\n\r"))
		return e
	})

	return

}

func (vps *Vps) CacheMsg(msg *Message) (err error) {
	if msg == nil {
		return MsgIsNULL
	}
	history := Join(vps.myhome, MSG_HISTORY)
	err = vps.WithSftpWrite(history, os.O_WRONLY|os.O_APPEND|os.O_CREATE, func(fp io.WriteCloser) error {

		data, _ := json.Marshal(msg)
		_, err = fp.Write(data)
		return nil
	})
	return
}

func (vps *Vps) SendMsg(msg string, encrypted ...bool) (err error) {
	date := time.Now().Format(TIME_TMP)
	if vps.msgto == "" {
		return fmt.Errorf("offline/no set user")
	}

	msgpath := Join(ROOT, vps.msgto, MSG_FILE)
	err = vps.WithSftpWrite(msgpath, os.O_RDWR|os.O_APPEND|os.O_CREATE, func(fp io.WriteCloser) error {
		d := Message{
			Date: fmt.Sprint(date),
			Data: msg,
			From: vps.E(vps.name),
			To:   vps.msgto,
		}
		if encrypted != nil && encrypted[0] {
			d.Crypted = true
		}
		data, _ := json.Marshal(&d)
		_, e := fp.Write([]byte(string(data) + "\n\r"))

		if e == nil {
			e = vps.CacheMsg(&d)
		}
		return e
	})
	return

}

func (vps *Vps) HeartBeat() {
	if !vps.hearted {
		for {

			last_activate := Join(vps.myhome, MSG_HEART)
			t := time.Now()
			vps.WithSftpWrite(last_activate, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, func(fp io.WriteCloser) error {

				_, er := fp.Write([]byte(t.Format(TIME_TMP)))
				return er
			})
			time.Sleep(time.Duration(vps.heartInterval) * time.Second)
			// fmt.Println("----- heart beat :", t)
		}
	}
}

func (vps *Vps) IfAlive() (out bool) {
	last_activate := Join(ROOT, vps.msgto, MSG_HEART)

	err := vps.WithSftpRead(last_activate, os.O_RDONLY, func(fp io.ReadCloser) error {
		buf, err := ioutil.ReadAll(fp)
		if err != nil {
			return err
		}
		t, err := time.Parse(TIME_TMP, string(buf))
		if err != nil {
			return err
		}
		if !time.Now().After(t.Add(60 * time.Second)) {
			out = true
		}
		return nil
	})
	if err != nil {
		out = false
	}
	return
}

func (vps *Vps) History() (msgs []*Message, err error) {

	msgpath := Join(vps.myhome, MSG_HISTORY)

	vps.WithSftpRead(msgpath, os.O_RDONLY|os.O_CREATE, func(fp io.ReadCloser) error {

		buf, err := ioutil.ReadAll(fp)
		if err != nil {
			log.Println("read msg err :", err)
			return err
		}

		// msgs = []map[string]string{}
		if len(buf) == 0 {
			return nil
		}
		for _, linebuf := range bytes.Split(buf, []byte("\n\r")) {
			onemsg := new(Message)
			// log.Println("msg :", linebuf)
			testmsg := strings.TrimSpace(string(linebuf))
			// log.Println("msg :", testmsg)
			if testmsg == "" {
				continue
			}
			err = json.Unmarshal([]byte(testmsg), onemsg)
			if err != nil {
				log.Println("msg err :", linebuf, err)
				continue
			}
			if strings.HasPrefix(onemsg.From, "${") {

			} else {
				msgs = append(msgs, onemsg)
			}

		}
		return err
	})
	return
}

func (vps *Vps) TimerClear(delay int, groupname ...string) (err error) {
	name := vps.name
	name = vps.GetVpsName()
	if groupname != nil {

		name = vps.GetGroupVpsName(groupname[0])
	}

	SecurityCheckName(name)
	buf := make([]byte, 30)
	rand.Read(buf)
	t := hex.EncodeToString(buf)
	p := Join("/tmp", t+".sh")

	vps.WithSftpWrite(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, func(fp io.WriteCloser) error {
		fp.Write([]byte(fmt.Sprintf(`
#!/bin/bash
cd %s && rm -rf  %s ;
rm $0; `, ROOT, name)))
		return nil
	})
	vps.WithSftp(func(client *sftp.Client) error {
		client.Chmod(p, 0500)
		return nil
	})

	vps.session, _ = vps.client.NewSession()
	err = vps.session.Run(fmt.Sprintf("(sleep %d ; %s) >/dev/null 2>/dev/null &", delay, p))
	if err != nil {
		log.Println("run sleep exit", err)
	}
	return
}

func (vps *Vps) Contact() (users []*User, err error) {
	vps.WithSftpDir(ROOT, func(fs os.FileInfo) error {
		if fs.Name() == vps.GetVpsName() {
			return nil
		}
		if fs.IsDir() && fs.Name() != MSG_FILE_ROOT && fs.Name() != MSG_TMP_FILE {
			userinfo := Join(ROOT, fs.Name(), MSG_HEART)
			// fmt.Println(userinfo)
			targetRoot := fs.Name()

			err = vps.WithSftpRead(userinfo, os.O_RDONLY, func(fp io.ReadCloser) error {
				buf, err := ioutil.ReadAll(fp)
				if err != nil {
					return err
				}
				ts := strings.TrimSpace(string(buf))
				_, err = time.Parse(TIME_TMP, ts)
				if err != nil {
					return err
				}

				user := &User{
					Name:       vps.D(targetRoot),
					LastActive: ts,
				}
				user.State = user.IfAlive()
				users = append(users, user)
				return nil
			})
			if err != nil {
				// fmt.Println("parse heart er:", err, userinfo)
				user := &User{
					Name:       vps.D(targetRoot),
					LastActive: time.Now().Format(TIME_TMP),
				}
				user.State = true
				users = append(users, user)
				err = nil
			}
		}

		return nil
	})
	return
}

func (vps *Vps) backgroundRecvMsgs() {
	tick := time.NewTicker(300 * time.Millisecond)
	// fmt.Println("----- start recving msg -----")
BACKEND:
	for {
		// fmt.Println("do some")
		select {
		case <-tick.C:

			// fmt.Println("recving ")
			if msgs, err := vps.RecvMsg(); err != nil {
				log.Println("[recv failed]:", err)
				time.Sleep(1 * time.Second)
				vps.Connect()
			} else {
				// fmt.Println("recved:", len(msgs))
				for _, msg := range msgs {
					// fmt.Println(msg.From)
					t, er := time.Parse(TIME_TMP, msg.Date)
					if er != nil {
						log.Println("[recv failed with date]:", msg)
						continue
					}
					if vps.onMessage != nil {
						go vps.onMessage(msg.Group, msg.From, msg.To, msg.Data, msg.Crypted, msg.Tp, t)
					} else {

					}
				}
			}

			msgs := vps.RecvGroupMsg()
			if len(msgs) > 0 {
				for _, msg := range msgs {
					// fmt.Println(msg.From)
					t, er := time.Parse(TIME_TMP, msg.Date)
					if er != nil {
						log.Println("[recv failed with date]:", msg)
						continue
					}
					if vps.onMessage != nil {
						go vps.onMessage(msg.Group, msg.From, msg.To, msg.Data, msg.Crypted, msg.Tp, t)
					} else {

					}
				}
			}

		case c := <-vps.signal:
			if c == -1 {
				break BACKEND
			}
		default:
			time.Sleep(200 * time.Millisecond)

		}
	}
	close(vps.signal)
	fmt.Println("exit recv msg in background!!")
}

func (vps *Vps) OnMessage(call func(group, from, to, msg string, crypted bool, tp int, date time.Time)) {
	vps.onMessage = call
}

func (vps *Vps) CreateMe() (canlogin bool) {
	mykeys := LocalKeys()
	canlogin = true
	var err error
	if vps.session == nil {

		vps.client, vps.session, err = vps.Connect()
	}
	if err == nil {
		fmt.Println(utils.Green("Connected", vps.myhome))
	} else {
		log.Fatal("can not connect:", err)
	}
	dpath := Join(ROOT, MSG_TMP_FILE)
	inits := fmt.Sprintf("mkdir -p %s ;mkdir -p %s && mkdir -p %s/%s && echo \"%s\" >  %s/%s",
		dpath,
		vps.myhome,
		vps.myhome,
		MSG_FILE_ROOT,
		vps.client.LocalAddr().String(),
		vps.myhome,
		MSG_HELLO,
	)
	err = vps.session.Run(inits)
	if err != nil {
		log.Fatal("init err:" + inits)
	}
	k := Join(vps.myhome, MSG_PRIVATE_KEY)
	lines, err := vps.WithSftpReadAsLines(k)
	if err == nil {
		for _, line := range lines {
			if strings.Contains(line, vps.name) {
				fs := strings.SplitN(line, ":", 2)
				if exists, _ := VerifyKey(fs[0], fs[1]); !exists {
					log.Println("login failed !!!:", fs[0])
					return false
				}
				break
			}
		}

	}
	vps.WithSftpWrite(k, os.O_CREATE|os.O_TRUNC|os.O_RDWR, func(fp io.WriteCloser) error {
		fp.Write([]byte(strings.Join(mykeys, "\n")))
		return nil
	})
	return
}

func (vps *Vps) Init() (err error) {
	vps.Security()
	if !vps.CreateMe() {
		return fmt.Errorf("login failed: user already exists but key is err!%s", "")
	}
	vps.heartInterval = 1
	go vps.HeartBeat()
	go vps.backgroundRecvMsgs()
	return
}

func (vps *Vps) Close() {
	vps.signal <- -1
	if vps.session != nil {
		vps.session.Close()
	}
}

func (vps *Vps) CloseWithClear(t int) {
	err := vps.TimerClear(t)
	if err != nil {
		log.Println(err)
	}
	vps.Close()
}

func (vps *Vps) ContactTo(name string) (ip string, err error) {
	// vps.msgto = name
	vps.SetMsgTo(name)
	vps.state = TALKER_REQ
	msgpath := Join(ROOT, vps.msgto, MSG_HELLO)
	err = vps.WithSftpRead(msgpath, os.O_RDONLY, func(fp io.ReadCloser) error {
		buf, err := ioutil.ReadAll(fp)
		if err != nil {
			return err
		}
		ip = strings.TrimSpace(string(buf))
		return nil
	})
	if err != nil {
		log.Println("contact to init check info err:", err)
		return
	}
	my, you := vps.ExchangeKeyCheck()
	if !you {
		// vps.state = TALKER_REQ
		err = vps.SendKeyTo(vps.msgto, GetKey(vps.name))
		if err != nil {
			log.Println("send req key err:", err)
			return "", err
		}

		vps.state |= TALKER_YOU_HAVE

	} else {
		vps.state |= TALKER_YOU_HAVE

	}

	if !my {

		err = vps.SendKeyReq()
		if err != nil {
			log.Println("send req key err:", err)
			return "", err
		}
	} else {
		vps.state |= TALKER_I_HAVE
	}

	if my && you {
		vps.state = TALKER_CONNECTED
	}
	vps.msgtoIP = ip

	go func() {
		for {
			time.Sleep(5 * time.Second)
			if vps.GetRawMsgTo() != "" {
				if !vps.IfAlive() {
					fmt.Println(vps.GetRawMsgTo(), vps.msgtoIP, "offline")
					vps.msgto = ""
					break
				}
			} else {
				break
			}

		}
	}()
	return ip, err
}

// func (vps )

func (vps *Vps) ExchangeKeyCheck() (my, you bool) {
	pa := Join(ROOT, vps.msgto, MSG_PRIVATE_KEY)
	err := vps.WithSftpRead(pa, os.O_RDONLY, func(fp io.ReadCloser) error {
		buf, _ := ioutil.ReadAll(fp)
		for _, l := range strings.Split(string(buf), "\n") {
			if strings.Contains(l, ":") {

				fs := strings.SplitN(l, ":", 2)
				name := strings.TrimSpace(fs[0])
				md5 := strings.TrimSpace(fs[1])
				if name == vps.name {

					if ToMd5(GetKey(vps.name)) == md5 {
						fmt.Println("You have my key:", name, md5)
						you = true
					} else {
						fmt.Println("Your key is out!:", name, md5)
					}

				} else if name == vps.msgto {
					if ToMd5(GetKey(vps.msgto)) == md5 {
						fmt.Println("I have your key:", name, md5)
						my = true
					} else {
						fmt.Println("i need your key !:", name, md5)
					}
				}

			}

		}
		return nil
	})
	if err != nil {
		return false, false
		// return false
	}
	return
}

func (vps *Vps) CloudFiles(groupName ...string) (files []string) {
	fsdir := Join(vps.myhome, MSG_FILE_ROOT)

	if groupName != nil {
		gname := vps.steam.FlowEn(groupName[0])
		fsdir = Join(ROOT, gname, MSG_FILE_ROOT)
	}

	vps.WithSftpDir(fsdir, func(fs os.FileInfo) error {
		if fs.IsDir() {
			return nil
		}
		files = append(files, fs.Name())
		return nil
	})
	return
}

func (vps *Vps) DownloadCloud(name string, dealStream func(reader io.Reader) error, groupName ...string) {
	src := Join(vps.myhome, MSG_FILE_ROOT, name)
	if groupName != nil {
		gname := vps.steam.FlowEn(groupName[0])
		src = Join(ROOT, gname, MSG_FILE_ROOT, name)
	}
	vps.WithSftpRead(src, os.O_RDONLY, func(fp io.ReadCloser) error {
		return dealStream(fp)
	})
}
