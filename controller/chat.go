package controller

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type ChatRoom struct {
	IP       string
	nowMsgTo string
	MyName   string
	vps      *Vps
	stream   *Stream
	recvMsg  chan *Message
	watch    func(msg *Message)
}

/*NewChatRoom
HOME default is '~'
*/
func NewChatRoom(sshstr string, HomePath ...string) (chat *ChatRoom, err error) {
	chat = new(ChatRoom)
	if HomePath != nil {
		fp, err := os.Stat(HomePath[0])
		if err == nil && fp.IsDir() {
			SetHome(HomePath[0])
		}
	}
	chat.vps = Parse(sshstr)
	SecurityCheckName(chat.vps.name)
	chat.IP = chat.vps.IP
	chat.stream, err = NewStreamWithAuthor(chat.vps.name, false)
	chat.recvMsg = make(chan *Message, 1024)
	chat.MyName = chat.vps.name
	go func() {
		chat.vps.OnMessage(func(group, from, msg string, crypted bool, tp int, date time.Time) {
			var m *Message
			if crypted {
				// log.Println("key:", chat.stream.Key)
				// err := chat.stream.LoadCipherByAuthor(from)
				grouped := false
				author := from
				if tp == MSG_TP_GROUP {
					grouped = true
					author = group
				}
				stream, err := NewStreamWithAuthor(author, grouped)
				if err != nil {
					log.Println("chat recv msg err: ", err)
					return
				}
				// log.Println("key 2:", chat.stream.Key)
				cipher, err := base64.RawStdEncoding.DecodeString(msg)
				if err != nil {
					log.Println("chat recv msg base64 err: ", err)
					return
				}
				// log.Println("key 3:", chat.stream.Key)

				realMsg := stream.De(cipher)
				// log.Println("key 4:", chat.stream.Key)

				m = &Message{
					Date:  date.Format(TIME_TMP),
					Data:  string(realMsg),
					From:  from,
					Group: group,
					Tp:    tp,
				}

			} else {
				m = &Message{
					Date:  date.Format(TIME_TMP),
					Data:  string(msg),
					From:  from,
					Tp:    tp,
					Group: group,
				}
			}
			if chat.watch != nil {
				go chat.watch(m)
			}
			chat.recvMsg <- m
		})
	}()
	return chat, err
}

func (chat *ChatRoom) TalkTo(name string) {
	chat.vps.ContactTo(name)
	chat.nowMsgTo = name

}

func (chat *ChatRoom) Write(msg string) {
	// fmt.Println("run write")
	e := chat.stream.En([]byte(msg))
	estr := base64.RawStdEncoding.EncodeToString(e)
	chat.vps.SendMsg(estr, true)
}

func (chat *ChatRoom) WriteGroup(gname, msg string) {
	// fmt.Println("run write")
	key := GetGroupKey(gname)
	if key != "" {
		steam, err := NewStreamWithAuthor(gname, true)
		if err != nil {
			log.Println("load gkey err:", err)
			return
		}
		e := steam.En([]byte(msg))
		estr := base64.RawStdEncoding.EncodeToString(e)
		err = chat.vps.SendGroupMsg(gname, estr, true)
		if err != nil {
			log.Println("write group err:", err)
		}
	} else {
		fmt.Println("can not found grou key:", gname)
	}
}

func (chat *ChatRoom) CreateGroup(name string) {
	chat.vps.CreateGroup(name)
}
func (chat *ChatRoom) RemoveGroup(name string) {
	chat.vps.RemoveGroup(name)
}

func (chat *ChatRoom) JoinGroup(name string) {
	chat.vps.JoinGroup(name)
}

func (chat *ChatRoom) Permitt(gname, uname string) {
	chat.vps.AllowJoinGroup(gname, uname)
}

func (chat *ChatRoom) CloseWithClear(t int) {
	if chat.vps != nil {
		chat.vps.CloseWithClear(t)
	}
}

func (chat *ChatRoom) Read() *Message {

	// msg := new(Message)
	msg := <-chat.recvMsg
	return msg
}

func (chat *ChatRoom) Login() error {
	return chat.vps.Init()
}

func (chat *ChatRoom) SetWacher(call func(msg *Message)) {
	chat.watch = call
}

func (chat *ChatRoom) Contact() (users []*User) {
	users, err := chat.vps.Contact()
	if err != nil {
		log.Fatal("get contact err :", err)
	}
	return users
}

func (chat *ChatRoom) History() {
	msgs, err := chat.vps.History()
	if err != nil {
		log.Println("read history :", err)
		return
	}
	for _, msg := range msgs {
		if msg.Crypted {
			// log.Println("key:", chat.stream.Key)
			// err := chat.stream.LoadCipherByAuthor(from)
			goruped := false
			if msg.Tp == MSG_TP_GROUP {
				goruped = true
			}
			stream, err := NewStreamWithAuthor(msg.From, goruped)
			if err != nil {
				log.Println("chat recv msg err: ", err)
				return
			}
			// log.Println("key 2:", chat.stream.Key)
			cipher, err := base64.RawStdEncoding.DecodeString(msg.Data)
			if err != nil {
				log.Println("chat recv msg base64 err: ", err)
				return
			}
			// log.Println("key 3:", chat.stream.Key)

			realMsg := stream.De(cipher)
			// log.Println("key 4:", chat.stream.Key)

			m := &Message{
				Date: msg.Date,
				Data: "[history] " + string(realMsg),
				From: msg.From,
			}

			if chat.watch != nil {
				go chat.watch(m)
			}
			chat.recvMsg <- m

		} else {
			m := &Message{
				Date: msg.Date,
				Data: "[history] " + msg.Data,
				From: msg.From,
			}

			if chat.watch != nil {
				go chat.watch(m)
			}
			chat.recvMsg <- m
		}
	}
}

func (chat *ChatRoom) SendFile(path string, groupName ...string) (err error) {
	name := filepath.Base(path)
	f, err := os.Stat(path)
	if err != nil || f.IsDir() {
		log.Println(path, "not exists / is a dir !!")
		return
	}

	if chat.vps.msgto == "" {
		log.Println("no target !")
		return
	}
	grouped := false
	author := chat.nowMsgTo

	if groupName != nil {
		grouped = true
		author = groupName[0]
		if exists, verified := chat.vps.GroupCheck(author); !exists || !verified {
			log.Println("Group Verify failed exists:", exists, "key exists:", verified)
			return
		}
	}

	err = chat.vps.WithSendFile(path, func(networkFile io.Writer, rawFile io.Reader) (err error) {

		stream, err := NewStreamWithAuthor(author, grouped)
		if err != nil {
			log.Println("load straem err:", err)
			return err
		}
		stream.StreamEncrypt(networkFile, rawFile, func(updated int64) {
			if updated%(1024*1024) == 0 && updated != 0 {
				log.Println("encrypted upload "+name+" :", updated/1024/1024, "MB")
			}
		})
		return nil
	}, groupName...)
	if err != nil {
		return err
	}
	log.Println("encrypted upload " + name + " ok")
	return nil
}

func (chat *ChatRoom) GroupVerify(name string) bool {
	if exists, verified := chat.vps.GroupCheck(name); !exists || !verified {
		log.Println("Group Verify failed exists:", exists, "key exists:", verified)
		return false
	}
	return true
}

func (chat *ChatRoom) GetFile(name string, groupName ...string) (err error) {
	dirs := "Downloads"

	grouped := false
	author := chat.vps.name

	if groupName != nil {
		grouped = true
		author = groupName[0]
		if exists, verified := chat.vps.GroupCheck(author); !exists || !verified {
			log.Println("Group Verify failed exists:", exists, "key exists:", verified)
			return
		}
	}

	chat.vps.DownloadCloud(name, func(networkFile io.Reader) (err error) {
		stream, err := NewStreamWithAuthor(author, grouped)
		if err != nil {
			log.Println("load straem err:", err)
			return err
		}
		if _, err := os.Stat(dirs); err != nil {
			os.MkdirAll(dirs, os.ModePerm)
		}
		fpath := filepath.Join(dirs, name)
		fp, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		if err != nil {
			log.Println("create local file err:", err)
			return
		}
		defer fp.Close()
		stream.StreamDecrypt(fp, networkFile, func(downloaded int64) {
			if downloaded%(1024*1024) == 0 && downloaded != 0 {
				log.Println("encrypted download "+name+" :", downloaded/1024/1024, "MB")
			}
		})
		return nil
	})
	log.Println("encrypted download " + name + " ok")
	return nil
}

func (chat *ChatRoom) CloudFiles(groupName ...string) (fs []string) {
	if groupName != nil {
		if !chat.GroupVerify(groupName[0]) {
			return
		}
	}
	return chat.vps.CloudFiles(groupName...)
}
