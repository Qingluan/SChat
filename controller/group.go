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

func (vps *Vps) GroupCheck(name string) (exists, verified bool) {
	err := vps.WithSftp(func(client *sftp.Client) error {
		groupDir := Join(ROOT, name+GROUP_TAIL)
		if f, err := client.Stat(groupDir); err != nil {

			return err
		} else {
			if !f.IsDir() {
				return nil
			}
			exists = true
		}
		gkeys := Join(groupDir, "keys")
		vps.WithSftpRead(gkeys, os.O_RDONLY|os.O_CREATE, func(fp io.ReadCloser) error {
			buf, err := ioutil.ReadAll(fp)
			if err != nil {
				return err
			}
			for _, hmac := range strings.Split(string(buf), "\n") {
				if strings.Contains(hmac, ":") {
					fs := strings.SplitN(hmac, ":", 2)
					if ToMd5(GetGroupKey(strings.TrimSpace(fs[0]))) == strings.TrimSpace(fs[1]) {
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
	newgroupKey := NewKey()
	var err error
	// if vps.session == nil {

	_, session, err := vps.Connect()
	// }
	if err == nil {
		fmt.Println(utils.Green("creating group", name))
	}
	dpath := Join(ROOT, "tmp_file")
	name += GROUP_TAIL
	nghome := Join(ROOT, name)
	inits := fmt.Sprintf("mkdir -p %s ;mkdir -p %s && mkdir -p %s/files && echo \"%s\" >  %s/info",
		dpath,
		nghome,
		nghome,
		vps.client.LocalAddr().String(),
		nghome,
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
	SetGroupKey(name, newgroupKey)

}

func (vps *Vps) JoinGroup(name string) {
	exist, verified := vps.GroupCheck(name)
	if exist && !verified {
		// vps.SendMsg()
		vps.SendGroupMsg(name, "$want-join")
	} else {
		log.Println("exist:", exist, "verify:", verified)
	}
}

func (vps *Vps) AllowJoinGroup(gname, uname string) {
	key := GetGroupKey(gname)
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

			groups = append(groups, n)
		}
	}
	// groupsinfo := Join(vps.myhome, "group")
	// vps.WithSftpRead(groupsinfo, os.O_CREATE|os.O_RDONLY, func(fp io.ReadCloser) error {
	// 	buf, err := ioutil.ReadAll(fp)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	for _, l := range strings.Split(string(buf), "\n") {
	// 		l = strings.TrimSpace(l)
	// 		if l != "" {
	// 			groups = append(groups, l)
	// 		}
	// 	}
	// 	return nil
	// })
	return
}
