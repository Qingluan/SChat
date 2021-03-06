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
	IP         string
	nowMsgTo   string
	MyName     string
	vps        *Vps
	stream     *Stream
	baseStream *Stream
	recvMsg    chan *Message
	watch      func(msg *Message)
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
	err = chat.Init(sshstr)
	if err != nil {
		return
	}
	go func() {
		chat.vps.OnMessage(func(group, from, to, msg string, crypted bool, tp int, date time.Time) {
			m, err := chat.ReadEncryptedMessage(msg, from, to, group, date, tp, crypted)
			if err != nil {
				log.Println("[Read encrypted err]:", err)
				return
			}
			if chat.watch != nil {
				go chat.watch(m)
			}
			chat.recvMsg <- m
		})
	}()
	return chat, err
}

func (chat *ChatRoom) ReadEncryptedMessage(msg, from, to, group string, date time.Time, tp int, crypted bool) (m *Message, err error) {
	if crypted {
		// log.Println("key:", chat.stream.Key)
		// err := chat.stream.LoadCipherByAuthor(from)
		grouped := false
		author := chat.vps.D(from)
		gname := group
		if tp == MSG_TP_GROUP {
			grouped = true
			author = chat.vps.GetGroupName(group)
		}

		if gname != "" {
			gname = chat.vps.GetGroupName(gname)
		}
		stream, err := NewStreamWithAuthor(author, grouped)
		if err != nil {
			log.Println("chat recv msg err: ", err)
			return nil, err
		}
		// log.Println("key 2:", chat.stream.Key)
		cipher, err := base64.RawStdEncoding.DecodeString(msg)
		if err != nil {
			log.Println("chat recv msg base64 err: ", err)
			return nil, err
		}
		// log.Println("key 3:", chat.stream.Key)

		realMsg := stream.De(cipher)
		// log.Println("key 4:", chat.stream.Key)

		m = &Message{
			Date:  date.Format(TIME_TMP),
			Data:  string(realMsg),
			From:  chat.vps.D(from),
			Group: gname,
			To:    chat.vps.D(to),
			Tp:    tp,
		}

	} else {
		m = &Message{
			Date:  date.Format(TIME_TMP),
			Data:  string(msg),
			From:  chat.vps.D(from),
			Tp:    tp,
			To:    chat.vps.D(to),
			Group: chat.vps.GetGroupName(group),
		}
	}
	return m, nil
}

func (chat *ChatRoom) TalkTo(name string) (icon string) {
	chat.vps.ContactTo(name)
	chat.nowMsgTo = name
	icon, err := chat.GetTalkerSIconPath()
	if err != nil {
		log.Println("fetch talker's icon err:", err)
	}
	return
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
	// gname = chat.vps.GetGroupVpsName(gname)
	if key != "" {
		// fmt.Println("key:", key)
		steam, err := NewStreamWithAuthor(gname, true)
		if err != nil {
			log.Println("load gkey err:", err)
			return
		}
		// fmt.Println("key 2:", key, gname)
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

func (chat *ChatRoom) GetTalker() string {
	return chat.vps.GetRawMsgTo()
}

func (chat *ChatRoom) SaveKeyToServer(key string) bool {
	return chat.vps.SaveKeyToServer(key)
}

func (chat *ChatRoom) RestoreKeyFromServer(key string) bool {
	return chat.vps.TryRestoreKey(key)
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
		fmt.Println("load kye by :", author, "grouped:", grouped)

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
		log.Println("send file err:", err)
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
	author := chat.MyName
	fmt.Println("author:", author)

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
		fmt.Println("load kye by :", author, "grouped:", grouped)
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

func (chat *ChatRoom) History() {
	msgs, err := chat.vps.History()
	if err != nil {
		log.Println("read history :", err)
		return
	}
	for _, msg := range msgs {
		date, _ := time.Parse(TIME_TMP, msg.Date)

		m, err := chat.ReadEncryptedMessage(msg.Data, msg.From, msg.To, msg.Group, date, msg.Tp, msg.Crypted)
		if err != nil {
			log.Println("[history read encrypted msg err]:", err)
			continue
		}
		// fmt.Println("rec", m)
		m.Data = fmt.Sprintf("[history]: %s", m.Data)
		if chat.watch != nil {
			go chat.watch(m)
		}
		chat.recvMsg <- m
	}
}

func (chat *ChatRoom) GetMyLocalHome() string {
	return HOME
}

func (chat *ChatRoom) ClearLocalCache() {
	ClearLocalCache()
}
