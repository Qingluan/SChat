package controller

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Qingluan/FrameUtils/utils"
	"github.com/Qingluan/merkur"
	"github.com/machinebox/progress"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	NO_TALKER       = 0x000
	TALKER_REQ      = 0x001
	TALKER_I_HAVE   = 0x010
	TALKER_YOU_HAVE = 0x100

	TALKER_CONNECTED = 0x111
	TALKER_ERR       = -1
)

type Vps struct {
	IP            string
	USER          string
	PWD           string
	TAG           string
	Region        string
	Proxy         string
	name          string
	myhome        string
	msgto         string
	msgtoIP       string
	state         int
	hearted       bool
	heartInterval int
	session       *ssh.Session
	client        *ssh.Client
	sftsess       *sftp.Client

	signal    chan int
	onMessage func(from, msg string, crypted bool, date time.Time)
}

const ROOT = "/tmp/SecureRoom"
const TIME_TMP = "2006-1-2 15:04:05 -0700 MST"

var (
	HOME, _ = os.UserHomeDir()
)

type User struct {
	Name       string `json:"name"`
	State      bool   `json:"login"`
	LastActive string `json:"last"`
}
type Message struct {
	Date    string `json:"date"`
	Data    string `json:"data"`
	From    string `json:"from"`
	Crypted bool   `json:"crypt"`
}

func SecurityCheckName(name string) {
	for _, c := range []string{" ", "/", "$", "&", "!", "@", "#", "%", "^", "&", "*", "+", "=", "`", "~", ";", ":", "\\", "|"} {
		if strings.Contains(name, c) {
			log.Fatal(fmt.Sprintf("!!!! name can not include '%s'", c))
		}

	}
}

func Join(root string, files ...string) string {
	fs := root
	if !strings.HasSuffix(root, "/") {
		fs += "/"
	}
	L := len(files)
	for n, f := range files {
		f = strings.TrimSpace(f)
		if strings.HasPrefix(f, "./") {
			fs += f[2:]
		} else if strings.HasPrefix(f, "/") {
			fs = f
		} else {
			fs += f
		}
		if n < L-1 && !strings.HasSuffix(root, "/") {
			fs += "/"
		}
	}
	return fs
}

func (u *User) Last() (t time.Time) {
	t, err := time.Parse(TIME_TMP, u.LastActive)
	if err != nil {
		log.Println("parse last active time failed !", u.LastActive, err)
		t, _ = time.Parse(TIME_TMP, TIME_TMP)
	}
	return
}

func (u *User) Acivte() string {
	return fmt.Sprintf("%s", time.Now().Sub(u.Last()))
}

func (u *User) IfAlive() bool {
	return !time.Now().After(u.Last().Add(15 * time.Second))
}

func (vps Vps) String() string {
	return fmt.Sprintf("%s(Loc:%s Tag:%s)", vps.IP, vps.Region, vps.TAG)
}
func (vps *Vps) Connect() (client *ssh.Client, sess *ssh.Session, err error) {
	var sshConfig *ssh.ClientConfig
	var signer ssh.Signer
	foundPri := false
	// var err error
	// if vps.PWD == "" {

	privateKeyPath := filepath.Join(HOME, ".ssh", "id_rsa")
	pemBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		// log.Printf("Reading private key file failed %v", err)
	} else {
		signer, err = signerFromPem(pemBytes, []byte(""))
		if err != nil {
			log.Fatal(err)
		}
		foundPri = true

	}
	// create signer
	Auth := []ssh.AuthMethod{}
	if foundPri {
		Auth = append(Auth, ssh.PublicKeys(signer))
	}
	Auth = append(Auth, ssh.Password(vps.PWD))

	sshConfig = &ssh.ClientConfig{
		User: vps.USER,
		Auth: Auth,
	}
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	ip := vps.IP
	if !strings.Contains(ip, ":") {
		ip += ":22"
	}
	if vps.Proxy != "" {
		if dialer := merkur.NewProxyDialer(vps.Proxy); dialer != nil {
			if conn, err := dialer.Dial("tcp", ip); err == nil {
				conn, chans, reqs, err := ssh.NewClientConn(conn, ip, sshConfig)
				if err != nil {
					return nil, nil, err
				}
				log.Println(utils.Green("Use Proxy:", vps.Proxy))
				client = ssh.NewClient(conn, chans, reqs)
			} else {
				return nil, nil, err
			}
		} else {
			return nil, nil, fmt.Errorf("%v", "no proxy dialer create ok!")
		}
	} else {
		client, err = ssh.Dial("tcp", ip, sshConfig)
	}
	if err != nil {
		return nil, nil, err
	}
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}
	vps.session = session
	vps.client = client
	return client, session, nil
}

func (vps *Vps) Init(mykeys []string) (err error) {
	if vps.session == nil {

		vps.client, vps.session, err = vps.Connect()
	}
	if err == nil {
		fmt.Println(utils.Green("Connected", vps.myhome))
	}
	dpath := Join(ROOT, "tmp_file")
	inits := fmt.Sprintf("mkdir -p %s ;mkdir -p %s && mkdir -p %s/files && echo \"%s\" >  %s/info",
		dpath,
		vps.myhome,
		vps.myhome,
		vps.client.LocalAddr().String(),
		vps.myhome,
	)
	err = vps.session.Run(inits)
	k := Join(vps.myhome, "keys")
	vps.WithSftpWrite(k, os.O_CREATE|os.O_TRUNC|os.O_RDWR, func(fp io.WriteCloser) error {
		fp.Write([]byte(strings.Join(mykeys, "\n")))
		return nil
	})
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
	vps.msgto = name
	vps.state = TALKER_REQ
	msgpath := Join(ROOT, vps.msgto, "info")
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
			if vps.msgto != "" {
				if !vps.IfAlive() {
					fmt.Println(vps.msgto, vps.msgtoIP, "offline")
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
	pa := Join(ROOT, vps.msgto, "keys")
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

func (vps *Vps) CloudFiles() (files []string) {
	fsdir := Join(vps.myhome, "files")
	vps.WithSftpDir(fsdir, func(fs os.FileInfo) error {
		if fs.IsDir() {
			return nil
		}
		files = append(files, fs.Name())
		return nil
	})
	return
}

func (vps *Vps) DownloadCloud(name string, dealStream func(reader io.Reader) error) {
	src := Join(vps.myhome, "files", name)
	vps.WithSftpRead(src, os.O_RDONLY, func(fp io.ReadCloser) error {
		return dealStream(fp)
	})
}

func (vps *Vps) SendKeyReq() (err error) {
	date := time.Now().Format(TIME_TMP)
	if vps.msgto == "" {
		return fmt.Errorf("offline/no set user")
	}
	msgpath := Join(ROOT, vps.msgto, "message.txt")
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

func (vps *Vps) SendKeyTo(target, key string) (err error) {
	// fmt.Println("test:", vps.name, "send key to", target)
	date := time.Now().Format(TIME_TMP)
	// if vps.msgto == "" {
	// 	return fmt.Errorf("offline/no set user")
	// }
	msgpath := Join(ROOT, target, "message.txt")
	err = vps.WithSftpWrite(msgpath, os.O_RDWR|os.O_APPEND|os.O_CREATE, func(fp io.WriteCloser) error {
		d := Message{
			Date: fmt.Sprint(date),
			Data: key,
			From: "${key}:" + vps.name,
		}
		data, _ := json.Marshal(&d)
		_, e := fp.Write([]byte(string(data) + "\n\r"))
		return e
	})
	return
}

func (vps *Vps) WithSendFile(path string, dealStream func(networkFile io.Writer, rawFile io.Reader) (err error)) (err error) {
	name := filepath.Base(path)
	fpath := Join(ROOT, "tmp_file", name)
	if vps.msgto == "" {
		return
	}
	dpath := Join(ROOT, vps.msgto, "files", name)

	err = vps.WithSftpWrite(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, func(fp io.WriteCloser) error {
		readfp, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return err
		}
		defer readfp.Close()
		err = dealStream(fp, readfp)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Println("send file to tmp err:", err)
	}
	// err = vps.session.Run(fmt.Sprintf("mv %s %s", fpath, dpath))
	err = vps.WithSftp(func(client *sftp.Client) error {
		return client.Rename(fpath, dpath)
	})
	return
}

func (vps *Vps) SendMsg(msg string, encrypted ...bool) (err error) {
	date := time.Now().Format(TIME_TMP)
	if vps.msgto == "" {
		return fmt.Errorf("offline/no set user")
	}
	if vps.state != TALKER_CONNECTED {
		log.Println("not talker state ... wait auth...:", vps.state)
	}
	msgpath := Join(ROOT, vps.msgto, "message.txt")
	err = vps.WithSftpWrite(msgpath, os.O_RDWR|os.O_APPEND|os.O_CREATE, func(fp io.WriteCloser) error {
		d := Message{
			Date: fmt.Sprint(date),
			Data: msg,
			From: vps.name,
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

func (vps *Vps) HeartBeat() {
	if !vps.hearted {
		for {

			last_activate := Join(vps.myhome, "heart")
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
	last_activate := Join(ROOT, vps.msgto, "heart")

	err := vps.WithSftpRead(last_activate, os.O_RDONLY, func(fp io.ReadCloser) error {
		buf, err := ioutil.ReadAll(fp)
		if err != nil {
			return err
		}
		t, err := time.Parse(TIME_TMP, string(buf))
		if err != nil {
			return err
		}
		if !time.Now().After(t.Add(10 * time.Second)) {
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

	msgpath := Join(vps.myhome, "message.history.txt")

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

func (vps *Vps) TimerClear(delay int) (err error) {
	SecurityCheckName(vps.name)
	buf := make([]byte, 30)
	rand.Read(buf)
	t := hex.EncodeToString(buf)
	p := Join("/tmp", t+".sh")
	vps.WithSftpWrite(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, func(fp io.WriteCloser) error {
		fp.Write([]byte(fmt.Sprintf(`
#!/bin/bash
cd /tmp/SecureRoom ;
rm -rf  %s
rm $0; `, vps.name)))
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

func (vps *Vps) RecvMsg() (msgs []*Message, err error) {

	msgpath := Join(vps.myhome, "message.txt")

	dealSpecialMessage := false
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

			// switch vps.state {
			// case TALKER_CONNECTED:
			// default:
			if strings.HasPrefix(onemsg.From, "${") {
				if strings.HasPrefix(onemsg.From, "${no-key}:") {
					reqName := strings.TrimSpace(strings.SplitN(onemsg.From, "${no-key}:", 2)[1])
					if key := GetKey(vps.name); key != "" {
						// fmt.Println("kkk:", reqName)
						go vps.SendKeyTo(reqName, key)
						dealSpecialMessage = true
					}
				} else if strings.HasPrefix(onemsg.From, "${key}:") {
					reqName := strings.TrimSpace(strings.SplitN(onemsg.From, "${key}:", 2)[1])
					vps.state |= TALKER_I_HAVE
					// fmt.Println("i got your key :", reqName, onemsg.Data)
					go SetKey(reqName, onemsg.Data)
					dealSpecialMessage = true
				}
			} else {
				msgs = append(msgs, onemsg)

			}
			// }

		}
		return err
	})
	if len(msgs) > 0 || dealSpecialMessage {
		msgpath = Join(vps.myhome, "message.history.txt")
		err = vps.WithSftpWrite(msgpath, os.O_RDWR|os.O_CREATE|os.O_APPEND, func(fp io.WriteCloser) error {
			msg_string := ""
			for _, m := range msgs {
				if strings.HasPrefix(m.From, "${") {
					continue
				}
				data, _ := json.Marshal(&m)
				msg_string += string(data) + "\n\r"
			}
			_, err := fp.Write([]byte(msg_string))
			return err
		})
		if err == nil {
			err = vps.sftsess.Remove(Join(vps.myhome, "message.txt"))
		} else {
			log.Println("overmessge history err:", err)
		}
	}
	return msgs, err
}

func (vps *Vps) Contact() (users []*User, err error) {
	vps.WithSftpDir(ROOT, func(fs os.FileInfo) error {
		if fs.Name() == vps.name {
			return nil
		}
		if fs.IsDir() && fs.Name() != "files" && fs.Name() != "tmp_file" {
			userinfo := Join(ROOT, fs.Name(), "heart")
			// fmt.Println(userinfo)
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
					Name:       fs.Name(),
					LastActive: ts,
				}
				user.State = user.IfAlive()
				users = append(users, user)
				return nil
			})
			if err != nil {
				fmt.Println("parse heart er:", err, userinfo)
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
						go vps.onMessage(msg.From, msg.Data, msg.Crypted, t)
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

func (vps *Vps) OnMessage(call func(from, msg string, crypted bool, date time.Time)) {
	vps.onMessage = call
}

func (vps *Vps) WithSftp(done func(client *sftp.Client) error) (err error) {
	if vps.sftsess == nil {
		if sftpChannel, err := sftp.NewClient(vps.client); err != nil {
			// log.Println(err)
			return err
		} else {
			vps.sftsess = sftpChannel
		}

	}

	err = done(vps.sftsess)

	return
}

func (vps *Vps) WithSftpRead(path string, flags int, done func(fp io.ReadCloser) error) (err error) {
	err = vps.WithSftp(func(client *sftp.Client) (e error) {

		fp, err := client.OpenFile(path, flags)
		if err != nil {
			return fmt.Errorf("can not open file:%s", path)
		}
		err = done(fp)
		defer fp.Close()
		return err
	})
	return
}

func (vps *Vps) WithSftpDir(path string, done func(fs os.FileInfo) error) (err error) {
	err = vps.WithSftp(func(client *sftp.Client) (e error) {

		fss, err := client.ReadDir(path)
		if err != nil {
			return fmt.Errorf("can not open file:%s", path)
		}
		for _, fs := range fss {
			err = done(fs)
			if err != nil {
				return fmt.Errorf("parse one file in dir err:%s | %s", path, err)
			}
		}
		// defer fp.Close()
		return err
	})
	return
}

func (vps *Vps) WithSftpWrite(path string, flags int, done func(fp io.WriteCloser) error) (err error) {
	err = vps.WithSftp(func(client *sftp.Client) (e error) {

		fp, err := client.OpenFile(path, flags)
		if err != nil {
			return fmt.Errorf("can not open file:%s", path)
		}
		err = done(fp)
		defer fp.Close()
		return err
	})
	return
}

func (vps Vps) Rm(file string) bool {
	if _, sess, err := vps.Connect(); err != nil {
		return false
	} else {
		if err := sess.Run("rm " + Join("/tmp", file)); err != nil {
			return false
		} else {
			return true
		}
	}
}

func (vps Vps) Shell() bool {
	if conn, session, err := vps.Connect(); err != nil {
		log.Fatal(err)
		return false
	} else {
		if runtime.GOOS != "windows" {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
			ctx, cancel := context.WithCancel(context.Background())

			// session, err := conn.NewSession()
			// if err != nil {
			// 	return fmt.Errorf("cannot open new session: %v", err)
			// }

			run := func(ctx context.Context, conn *ssh.Client, session *ssh.Session) error {
				defer session.Close()

				go func() {
					<-ctx.Done()
					conn.Close()
				}()

				// fd := int(os.Stdin.Fd())
				fd := int(os.Stdout.Fd())

				state, err := terminal.MakeRaw(fd)
				if err != nil {
					return fmt.Errorf("terminal make raw: %s", err)
				}
				defer terminal.Restore(fd, state)

				w, h, err := terminal.GetSize(fd)
				if err != nil {
					return fmt.Errorf("terminal get size: %s", err)
				}

				modes := ssh.TerminalModes{
					ssh.ECHO:          1,
					ssh.TTY_OP_ISPEED: 14400,
					ssh.TTY_OP_OSPEED: 14400,
				}

				term := os.Getenv("TERM")
				if term == "" {
					term = "xterm-256color"
				}
				if err := session.RequestPty(term, h, w, modes); err != nil {
					return fmt.Errorf("session xterm: %s", err)
				}

				session.Stdout = os.Stdout
				session.Stderr = os.Stderr
				session.Stdin = os.Stdin

				if err := session.Shell(); err != nil {
					return fmt.Errorf("session shell: %s", err)
				}

				if err := session.Wait(); err != nil {
					if e, ok := err.(*ssh.ExitError); ok {
						switch e.ExitStatus() {
						case 130:
							return nil
						}
					}
					return fmt.Errorf("ssh: %s", err)
				}
				return nil
			}

			go func() {
				if err := run(ctx, conn, session); err != nil {
					log.Print(err)
				}
				cancel()
			}()

			select {
			case <-sig:
				cancel()
			case <-ctx.Done():
			}

		} else {
			// session.Stdout = os.Stdout
			defer session.Close()

			// StdinPipe for commands
			stdin, err := session.StdinPipe()
			if err != nil {
				log.Fatal(err)
			}

			// Uncomment to store output in variable
			//var b bytes.Buffer
			//session.Stdout = &b
			//session.Stderr = &b

			// Enable system stdout
			// Comment these if you uncomment to store in variable
			session.Stdout = os.Stdout
			session.Stderr = os.Stderr

			// Start remote shell
			err = session.Shell()
			if err != nil {
				log.Fatal(err)
			}

			// send the commands
			buffer := bufio.NewReader(os.Stdin)
			for {
				// nowCwd := fmt
				time.Sleep(1 * time.Second)
				fmt.Printf("%s >", utils.Green(vps))
				line, _, _ := buffer.ReadLine()

				_, err = fmt.Fprintf(stdin, "%s\n", line)
				if err != nil {
					log.Fatal(err)
				}
			}

			// Wait for session to finish
			// err = session.Wait()
			// if err != nil {
			// 	log.Fatal(err)
			// }

			// Uncomment to store in variable
			//fmt.Println(b.String())

		}
		return true
	}
}

func (vps Vps) Upload(file string, canexcute bool) error {
	if cli, _, err := vps.Connect(); err != nil {
		log.Fatal(err)
		return err
	} else {
		if sftpChannel, err := sftp.NewClient(cli); err != nil {

			log.Println(err)
			return err

		} else {
			fileName := filepath.Base(file)
			sftpChannel.Remove("/tmp/" + fileName)
			fp, err := sftpChannel.OpenFile("/tmp/"+fileName, os.O_APPEND|os.O_CREATE|os.O_RDWR)

			if err != nil {

				log.Println(err)
				return err
			}
			localState, err := os.Stat(file)
			if err != nil {

				log.Println(err)
				return err
			}

			startAt := int64(0)
			defer fp.Close()
			if state, err := fp.Stat(); err == nil {
				startAt = state.Size()
				if startAt == localState.Size() {
					log.Println("Already upload !")
					return nil
				}
				if startAt != 0 {
					log.Println("Continued at:", float64(startAt)/float64(1024)/float64(1024), "MB")
				}
			}
			localFp, err := os.OpenFile(file, os.O_RDONLY, os.ModePerm)
			if err != nil {
				log.Println(err)
				return err
			}
			defer localFp.Close()
			_, err = localFp.Seek(startAt, os.SEEK_SET)
			if err != nil {
				log.Println(err)
				return err
			}
			// ctx := context.Background()

			// get a reader and the total expected number of bytes
			// s := `Now that's what I call progress`
			size := localState.Size()
			r := progress.NewReader(localFp)
			// Start a goroutine printing progress
			go func() {
				ctx := context.Background()
				progressChan := progress.NewTicker(ctx, r, size, 5*time.Second)
				for p := range progressChan {
					fmt.Printf("\r[%.3f%%] %.3f MB %s %v remaining...", p.Percent(), float64(p.Size())/float64(1024)/float64(1024), fileName, p.Remaining().Round(time.Second))
				}
				fmt.Println("\rdownload is completed")
			}()

			io.Copy(fp, r)
			if canexcute {
				fp.Chmod(os.ModeExclusive)
			}
		}

	}
	return nil
}

func Parse(sshStr string) *Vps {
	name := strings.SplitN(sshStr, "://", 2)[0]
	tail := strings.SplitN(sshStr, "://", 2)[1]
	v := &Vps{
		USER:   "root",
		Region: "Unkonw",
		TAG:    "Unkonw",
	}
	if strings.Contains(tail, "/") {
		ip := strings.SplitN(tail, "/", 2)[0]
		v.IP = ip
		auth := strings.SplitN(tail, "/", 2)[1]
		if strings.Contains(auth, ":") {
			fs := strings.SplitN(auth, ":", 2)
			v.USER = fs[0]
			v.PWD = fs[1]
		} else {
			if strings.TrimSpace(auth) != "" {
				v.USER = auth
			}
		}
	} else {
		v.IP = tail
	}
	v.name = name
	v.myhome = Join(ROOT, name)
	v.signal = make(chan int, 10)
	return v
}

func signerFromPem(pemBytes []byte, password []byte) (ssh.Signer, error) {

	// read pem block
	err := errors.New("Pem decode failed, no key found")
	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil {
		return nil, err
	}

	// handle encrypted key
	if x509.IsEncryptedPEMBlock(pemBlock) {
		// decrypt PEM
		pemBlock.Bytes, err = x509.DecryptPEMBlock(pemBlock, []byte(password))
		if err != nil {
			return nil, fmt.Errorf("Decrypting PEM block failed %v", err)
		}

		// get RSA, EC or DSA key
		key, err := parsePemBlock(pemBlock)
		if err != nil {
			return nil, err
		}

		// generate signer instance from key
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			return nil, fmt.Errorf("Creating signer from encrypted key failed %v", err)
		}

		return signer, nil
	} else {
		// generate signer instance from plain key
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing plain private key failed %v", err)
		}

		return signer, nil
	}
}

func parsePemBlock(block *pem.Block) (interface{}, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing PKCS private key failed %v", err)
		} else {
			return key, nil
		}
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing EC private key failed %v", err)
		} else {
			return key, nil
		}
	case "DSA PRIVATE KEY":
		key, err := ssh.ParseDSAPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing DSA private key failed %v", err)
		} else {
			return key, nil
		}
	default:
		return nil, fmt.Errorf("Parsing private key failed, unsupported key type %q", block.Type)
	}
}
