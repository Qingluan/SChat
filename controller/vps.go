package controller

import (
	"bufio"
	"bytes"
	"context"
	"crypto/x509"
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

type Vps struct {
	IP      string
	USER    string
	PWD     string
	TAG     string
	Region  string
	Proxy   string
	name    string
	myhome  string
	msgto   string
	msgtoIP string
	hearted bool
	session *ssh.Session
	client  *ssh.Client
	sftsess *sftp.Client

	signal    chan int
	onMessage func(from, msg string, date time.Time)
}

const ROOT = "/tmp/SecureRoom"

var (
	HOME, _ = os.UserHomeDir()
)

type Message struct {
	Date string `json:"date"`
	Data string `json:"data"`
	From string `json:"from"`
}

// type Vultr struct {
// 	API     string //
// 	Servers map[string]Vps
// }

func (vps Vps) String() string {
	return fmt.Sprintf("%s(Loc:%s Tag:%s)", vps.IP, vps.Region, vps.TAG)
}
func (vps *Vps) Connect() (client *ssh.Client, sess *ssh.Session, err error) {
	var sshConfig *ssh.ClientConfig
	var signer ssh.Signer
	// var err error
	// if vps.PWD == "" {

	privateKeyPath := filepath.Join(HOME, ".ssh", "id_rsa")
	pemBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		log.Fatal("Reading private key file failed %v", err)
	}
	// create signer
	signer, err = signerFromPem(pemBytes, []byte(""))
	if err != nil {
		log.Fatal(err)
	}

	sshConfig = &ssh.ClientConfig{
		User: vps.USER,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
			ssh.Password(vps.PWD),
		},
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

func (vps *Vps) Init() (err error) {
	if vps.session == nil {

		vps.client, vps.session, err = vps.Connect()
	}
	if err == nil {
		fmt.Println(utils.Green("Connected"))
	}
	err = vps.session.Run("mkdir -p " + vps.myhome + " ; echo \"" + vps.client.LocalAddr().String() + "\" > " + vps.myhome + "/info")
	go vps.HeartBeat()
	go vps.backgroundRecvMsgs()
	return
}

func (vps *Vps) Close() {
	vps.signal <- -1

}

func (vps *Vps) ContactTo(name string) (ip string, err error) {
	vps.msgto = name
	msgpath := filepath.Join(ROOT, vps.msgto, "info")
	err = vps.WithSftpRead(msgpath, os.O_RDONLY, func(fp io.ReadCloser) error {
		buf, err := ioutil.ReadAll(fp)
		if err != nil {
			return err
		}
		ip = strings.TrimSpace(string(buf))
		return nil
	})
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

func (vps *Vps) SendMsg(msg string) (err error) {
	date := time.Now().Format("2006-1-2 15:04:05(-0700)")
	if vps.msgto == "" {

		return fmt.Errorf("offline/no set user")
	}
	msgpath := filepath.Join(ROOT, vps.msgto, "message.txt")
	err = vps.WithSftpWrite(msgpath, os.O_RDWR|os.O_APPEND|os.O_CREATE, func(fp io.WriteCloser) error {
		d := Message{
			Date: fmt.Sprint(date),
			Data: msg,
			From: vps.name,
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

			time.Sleep(5 * time.Second)
			last_activate := filepath.Join(vps.myhome, "heart")
			t := time.Now()
			vps.WithSftpWrite(last_activate, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, func(fp io.WriteCloser) error {

				_, er := fp.Write([]byte(t.Format("2006-1-2 15:04:05 -0700 MST")))
				return er
			})
			// fmt.Println("----- heart beat :", t)
		}
	}
}

func (vps *Vps) IfAlive() (out bool) {
	last_activate := filepath.Join(ROOT, vps.msgto, "heart")

	err := vps.WithSftpRead(last_activate, os.O_RDONLY, func(fp io.ReadCloser) error {
		buf, err := ioutil.ReadAll(fp)
		if err != nil {
			return err
		}
		t, err := time.Parse("2006-1-2 15:04:05 -0700 MST", string(buf))
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

func (vps *Vps) RecvMsg() (msgs []*Message, err error) {
	// date := time.Now().Format("2006-1-2 15:04:05(-0700)")
	// fmt.Println(date)

	msgpath := filepath.Join(vps.myhome, "message.txt")
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
			msgs = append(msgs, onemsg)
		}
		return err
	})
	if len(msgs) > 0 {
		msgpath = filepath.Join(vps.myhome, "message.history.txt")
		err = vps.WithSftpWrite(msgpath, os.O_RDWR|os.O_CREATE|os.O_APPEND, func(fp io.WriteCloser) error {

			strings := ""
			for _, m := range msgs {
				data, _ := json.Marshal(&m)
				strings += string(data) + "\n\r"
			}
			_, err := fp.Write([]byte(strings))
			return err
		})
		if err == nil {
			err = vps.sftsess.Remove(filepath.Join(vps.myhome, "message.txt"))
		}

	}

	return msgs, err

}

func (vps *Vps) backgroundRecvMsgs() {
	tick := time.NewTicker(300 * time.Millisecond)
BACKEND:
	for {

		select {
		case <-tick.C:
			if msgs, err := vps.RecvMsg(); err != nil {
				log.Println("[recv failed]:", err)
				time.Sleep(1 * time.Second)
				vps.Connect()
			} else {
				for _, msg := range msgs {
					t, er := time.Parse("2006-1-2 15:04:05(-0700)", msg.Date)
					if er != nil {
						log.Println("[recv failed with date]:", msg)
						continue
					}
					if vps.onMessage != nil {
						go vps.onMessage(msg.From, msg.Data, t)
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

func (vps *Vps) OnMessage(call func(from, msg string, date time.Time)) {
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
		if err := sess.Run("rm " + filepath.Join("/tmp", file)); err != nil {
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
	v.myhome = filepath.Join(ROOT, name)
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
