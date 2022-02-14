package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
)

func (vps *Vps) RecvMsg() (msgs []*Message, err error) {

	msgpath := Join(vps.myhome, MSG_FILE)

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
				log.Println("msg err :", string(linebuf), err)
				continue
			}

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
				} else if strings.HasPrefix(onemsg.From, "${group-key}:") {
					reqName := strings.TrimSpace(strings.SplitN(onemsg.From, "${group-key}:", 2)[1])
					vps.state |= TALKER_I_HAVE
					gname := vps.GetGroupName(onemsg.Group)
					fmt.Println("i got group key :", onemsg.Data, "from", reqName, "in group:", gname)

					go SetGroupKey(gname, onemsg.Data)
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
		msgpath = Join(vps.myhome, MSG_HISTORY)
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
			err = vps.sftsess.Remove(Join(vps.myhome, MSG_FILE))
		} else {
			log.Println("overmessge history err:", err)
		}
	}
	return msgs, err
}

func (vps *Vps) RecvGroupMsg() (msgs []*Message) {
	groups := vps.GroupList()
	waiter := sync.WaitGroup{}
	rwlock := sync.RWMutex{}
	for n, group := range groups {
		waiter.Add(1)
		go func(gname string, wait *sync.WaitGroup) {
			defer wait.Done()
			dealSpecialMessage := false
			msgpath := Join(ROOT, vps.GetGroupVpsName(gname), MSG_FILE)

			tmpsmsgs := []*Message{}
			err := vps.WithSftpRead(msgpath, os.O_RDONLY|os.O_CREATE, func(fp io.ReadCloser) error {

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
					found := false
					for _, n := range strings.Split(onemsg.Readed, ",") {
						if n == vps.name {
							found = true
							break
						}
					}
					if !found {
						tmpsmsgs = append(tmpsmsgs, onemsg)

					}

				}
				return err
			})
			if err != nil {
				// log.Println("recv gmsg:"+gname, "err:", err)
			}
			if len(tmpsmsgs) > 0 || dealSpecialMessage {
				msgpath = Join(ROOT, vps.GetGroupVpsName(gname), MSG_FILE)
				err := vps.WithSftpWrite(msgpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, func(fp io.WriteCloser) error {
					msg_string := ""
					for _, m := range tmpsmsgs {
						if m.Readed == "" {
							m.Readed += vps.name
						} else {
							m.Readed += "," + vps.name
						}
						data, _ := json.Marshal(&m)
						msg_string += string(data) + "\n\r"
					}

					rwlock.Lock()
					msgs = append(msgs, tmpsmsgs...)
					rwlock.Unlock()
					_, err := fp.Write([]byte(msg_string))
					return err
				})
				if err == nil {
					// 	// vps.sftsess.Remove(Join(ROOT, gname+GROUP_TAIL, MSG_FILE))
					// 	// if err != nil {
					// 	// 	log.Println("remove old group msg err:", err)
					// 	// }

					// } else {
					// 	log.Println("overmessge history err:", err)
				}
			}
		}(group, &waiter)
		if n > 0 && n%20 == 0 {
			waiter.Wait()
		}
	}
	waiter.Wait()
	return msgs

	// return msgs, err
}
