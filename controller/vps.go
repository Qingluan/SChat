package controller

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Qingluan/FrameUtils/utils"
	"github.com/Qingluan/merkur"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
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
	myenname      string
	msgto         string
	msgtoIP       string
	loginpwd      string
	state         int
	liveInterval  int
	hearted       bool
	heartInterval int
	session       *ssh.Session
	client        *ssh.Client
	sftsess       *sftp.Client
	steam         *Stream

	signal    chan int
	onMessage func(group, from, to, msg string, crypted bool, tp int, date time.Time)
}

const TIME_TMP = "2006-1-2 15:04:05 -0700 MST"
const (
	MSG_TP_NORMAL = iota
	MSG_TP_GROUP
)

var (
	HOME, _ = os.UserHomeDir()
)

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

func (vps Vps) String() string {
	return fmt.Sprintf("%s(Loc:%s Tag:%s)", vps.IP, vps.Region, vps.TAG)
}

func (vps *Vps) SetProxy(url string) {
	vps.Proxy = url
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

func (vps *Vps) WithSendFile(path string, dealStream func(networkFile io.Writer, rawFile io.Reader) (err error), groupName ...string) (err error) {
	name := filepath.Base(path)
	fpath := Join(ROOT, MSG_TMP_FILE, name)
	if vps.msgto == "" {
		return
	}

	dpath := Join(ROOT, vps.msgto, MSG_FILE_ROOT, name)
	if groupName != nil {
		gname := vps.steam.FlowEn(groupName[0])
		dpath = Join(ROOT, gname, MSG_FILE_ROOT, name)
	}
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

func (vps *Vps) WithSendFileToOwn(path string, dealStream func(networkFile io.Writer, rawFile io.Reader) (err error)) (err error) {
	name := filepath.Base(path)
	fpath := Join(ROOT, MSG_TMP_FILE, name)
	if vps.msgto == "" {
		return
	}

	dpath := Join(vps.myhome, MSG_FILE_ROOT, name)
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

func (vps *Vps) WithSftp(done func(client *sftp.Client) error) (err error) {
	if vps.sftsess == nil {
		if vps.client == nil {
			// vps.client, vps.sftsess, err =
			_, _, err = vps.Connect()
			if err != nil {
				log.Fatal("connect Server:", vps.IP, "Failed!!!")
			}
		}
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

func (vps *Vps) WithSftpReadAsLines(path string, flags ...int) (lines []string, err error) {
	flag := os.O_RDONLY
	if flags != nil {
		flag |= flags[0]
	}
	err = vps.WithSftpRead(path, flag, func(fp io.ReadCloser) error {
		buf, err := ioutil.ReadAll(fp)
		if err != nil {
			return err
		}
		for _, l := range strings.Split(string(buf), "\n") {
			lines = append(lines, strings.TrimSpace(l))
		}
		return nil
	})
	return
}

func (vps *Vps) Exists(name string) (exists bool, dir bool) {
	vps.WithSftpDir(Join(vps.myhome, MSG_FILE_ROOT), func(fs os.FileInfo) error {
		if fs.Name() == name {
			exists = true
			dir = fs.IsDir()
		}
		return nil
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
