package controller

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/Qingluan/FrameUtils/utils"
	"github.com/pkg/sftp"
)

type Group struct {
	Members []string
}

func (chat *ChatRoom) IsGroupName(name string) (string, bool) {
	if strings.HasSuffix(name, GROUP_TAIL) {
		return strings.TrimSuffix(name, GROUP_TAIL), true
	} else if strings.HasSuffix(name, "_GroUP") {
		return strings.TrimSuffix(name, "_GroUP"), true
	}
	return name, false
}

func (vps *Vps) GroupCheck(name string) (exists, verified bool) {
	err := vps.WithSftp(func(client *sftp.Client) error {
		groupDir := Join(ROOT, vps.GetGroupVpsName(name))
		if f, err := client.Stat(groupDir); err != nil {

			return err
		} else {
			if !f.IsDir() {
				return nil
			}
			exists = true
		}
		gkeys := Join(groupDir, MSG_PRIVATE_KEY)
		vps.WithSftpRead(gkeys, os.O_RDONLY|os.O_CREATE, func(fp io.ReadCloser) error {
			buf, err := ioutil.ReadAll(fp)
			if err != nil {
				return err
			}
			for _, hmac := range strings.Split(string(buf), "\n") {
				if strings.Contains(hmac, ":") {
					fs := strings.SplitN(hmac, ":", 2)
					kname := strings.TrimSpace(fs[0])
					if strings.HasSuffix(kname, GROUP_TAIL) {
						kname = vps.D(kname)
					}
					if ToMd5(GetGroupKey(kname)) == strings.TrimSpace(fs[1]) {
						verified = true
						break
					}
				}
			}
			return nil
		})
		return nil
	})
	if err != nil {
		return
	}
	return
}

func (vps *Vps) CreateGroup(name string) {
	rawName := name
	name = vps.E(name)

	newgroupKey := NewKey()
	var err error
	// if vps.session == nil {

	_, session, err := vps.Connect()
	// }
	if err == nil {
		fmt.Println(utils.Green("creating group:", rawName))
	}
	dpath := Join(ROOT, MSG_TMP_FILE)
	name += GROUP_TAIL
	nghome := Join(ROOT, name)
	inits := fmt.Sprintf("mkdir -p %s ;mkdir -p %s && mkdir -p %s/%s && echo \"%s\" >  %s/%s",
		dpath,
		nghome,
		nghome,
		MSG_FILE_ROOT,
		vps.client.LocalAddr().String(),
		nghome,
		MSG_HELLO,
	)
	err = session.Run(inits)
	if err != nil {
		log.Println("create group base dir  err:", err)
		return
	}
	k := Join(nghome, "keys")
	err = vps.WithSftpWrite(k, os.O_CREATE|os.O_TRUNC|os.O_RDWR, func(fp io.WriteCloser) error {
		fp.Write([]byte(name + ":" + ToMd5(newgroupKey)))
		return nil
	})
	if err != nil {
		log.Println("create group err:", err)
		return
	}
	SetGroupKey(rawName, newgroupKey)

}

func (vps *Vps) JoinGroup(name string) {
	name = vps.GetGroupVpsName(name)
	exist, verified := vps.GroupCheck(name)
	if exist && !verified {
		// vps.SendMsg()
		vps.SendGroupMsg(name, "$want-join")
	} else {
		log.Println("exist:", exist, "verify:", verified)
	}
}

func (vps *Vps) E(raw string) string {

	return vps.steam.FlowEn(raw)
}

func (vps *Vps) D(en string) string {
	if strings.HasSuffix(en, GROUP_TAIL) {
		fs := strings.SplitN(en, GROUP_TAIL, 2)
		en := fs[0]
		return vps.steam.FlowDe(en) + GROUP_TAIL
	}
	return vps.steam.FlowDe(en)
}

func (vps *Vps) GetGroupVpsName(name string) string {
	if strings.HasSuffix(name, GROUP_TAIL) {
		return name
	}
	// if strings.HasSuffix(name, "_GroUP") {
	name = strings.TrimSuffix(name, "_GroUP")
	// }
	n := vps.E(name)
	return n + GROUP_TAIL
}

func (vps *Vps) GetGroupName(name string) string {
	n := vps.D(name)
	return strings.TrimSuffix(n, GROUP_TAIL)
}

func (vps *Vps) AllowJoinGroup(gname, uname string) {

	key := GetGroupKey(gname)
	gname = vps.GetGroupVpsName(gname)
	uname = vps.E(uname)
	if key != "" {
		vps.SendKeyTo(uname, key, gname)
		// vps.SendGroupMsg(msg.Group, "$permitted-"+key)
	}
}

func (vps *Vps) GroupList() (groups []string) {
	for _, h := range LocalGroupKeys() {
		if strings.Contains(h, ":") {
			n := strings.TrimSpace(strings.SplitN(h, ":", 2)[0])
			n = strings.TrimSuffix(n, GROUP_TAIL)
			n = vps.E(strings.TrimSuffix(n, "_GroUP")) + GROUP_TAIL
			groups = append(groups, n)
		}
	}
	return
}

func (chat *ChatRoom) CreateGroup(name string) {
	chat.vps.CreateGroup(name)
}

func (vps *Vps) RemoveGroup(name string) {
	encryptedname := vps.GetGroupVpsName(name)
	// fmt.Println("remove vps:", name, encryptedname, vps.GetGroupVpsName(encryptedname))
	exists, verify := vps.GroupCheck(encryptedname)
	if exists && verify {
		err := vps.TimerClear(3, encryptedname)
		fmt.Println("remove vps:", name, vps.GetGroupVpsName(encryptedname))
		if err != nil {
			log.Println(err)
		}
	} else {
		log.Println("you have no permitted to remove ")
	}
	// vps.Close()
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
