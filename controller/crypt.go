package controller

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	// HOME, _  = os.UserHomeDir()
	KeysHome      = filepath.Join(HOME, ".sshchat", "keys")
	GroupKeysHome = filepath.Join(HOME, ".sshchat", "gkeys")
)

const FILE_CIPHER_HEADER_LEN = 188

type Stream struct {
	Key    string `json:"key"`
	Author string `json:"author"`
	// Nonce string `json:"nonce"`
	cipher    *cipher.AEAD
	nonceSize int
}

func NewStreamWithRandomeKey() (stream *Stream, err error) {

	key := make([]byte, 32)
	rand.Read(key)
	keyb64 := base64.StdEncoding.EncodeToString(key)
	if err != nil {
		return
	}
	if len(key) < 32 {
		key = append(key, []byte("asfasivbniasgfbiasgbiasghiashfiashf13412$RASFWEAT!%!@%TRASFSDAT@!#%$!@$")[:32-len(key)]...)
	}
	c, err := aes.NewCipher(key)
	// if there are any errors, handle them
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	stream = new(Stream)
	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	stream.cipher = &gcm
	stream.Key = keyb64
	stream.nonceSize = (*stream.cipher).NonceSize()
	return
}

func NewStreamWithAuthor(author string, loadGroupKey bool) (stream *Stream, err error) {

	_, err = os.Stat(KeysHome)
	if err != nil {
		os.MkdirAll(KeysHome, os.ModePerm)
		err = nil
	}
	// k := stream.Key
	var key64 string
	// tmpkey := make([]byte, 32)
	saved := true
	// k, err := ioutil.ReadFile(filepath.Join(KeysHome, author+".key"))
	k := ""
	if loadGroupKey {
		k = GetGroupKey(author)
	} else {
		k = GetKey(author)

	}

	if k == "" {
		log.Println("no such author's key saved in local system!:", author, ".. so  will create new key!!!")
		key64 = NewKey()
		// // return
		// rand.Read(tmpkey)
		saved = false
		// if len(tmpkey) < 32 {
		// 	tmpkey = append(tmpkey, []byte("asfasivbniasgfbiasgbiasghiashfiashf13412$RASFWEAT!%!@%TRASFSDAT@!#%$!@$")[:32-len(tmpkey)]...)
		// }

		// key64 = base64.StdEncoding.EncodeToString(tmpkey)

	} else {
		key64 = strings.TrimSpace(string(k))
	}

	key, err := base64.StdEncoding.DecodeString(key64)

	if err != nil {
		return
	}
	if len(key) < 32 {
		key = append(key, []byte("asfasivbniasgfbiasgbiasghiashfiashf13412$RASFWEAT!%!@%TRASFSDAT@!#%$!@$")[:32-len(key)]...)
	}
	c, err := aes.NewCipher(key)
	// if there are any errors, handle them
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	stream = new(Stream)
	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	stream.cipher = &gcm
	stream.Key = key64
	stream.Author = author
	stream.nonceSize = (*stream.cipher).NonceSize()

	if !saved {
		stream.SaveKey(author, loadGroupKey)
	}
	return
}

func NewStreamWithBase64Key(keyb64 string) (stream *Stream, err error) {
	key, err := base64.StdEncoding.DecodeString(keyb64)

	if err != nil {
		return
	}
	if len(key) < 32 {
		key = append(key, []byte("asfasivbniasgfbiasgbiasghiashfiashf13412$RASFWEAT!%!@%TRASFSDAT@!#%$!@$")[:32-len(key)]...)
	}
	c, err := aes.NewCipher(key)
	// if there are any errors, handle them
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	stream = new(Stream)
	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	stream.cipher = &gcm
	stream.Key = keyb64
	stream.nonceSize = (*stream.cipher).NonceSize()

	return
}

func (stream *Stream) SaveKey(author string, grouped bool) {
	stream.Author = author
	khome := KeysHome
	if grouped {
		khome = GroupKeysHome
	}
	_, err := os.Stat(khome)
	if err != nil {
		os.MkdirAll(khome, os.ModePerm)
	}
	k := stream.Key
	author = strings.TrimSpace(author)
	author = strings.ReplaceAll(author, " ", "_")
	author = strings.ReplaceAll(author, "/", "_")

	ioutil.WriteFile(filepath.Join(khome, author+".key"), []byte(k), os.ModePerm)
}

func (stream *Stream) LoadCipherByAuthor(author string) (err error) {
	_, err = os.Stat(KeysHome)
	if err != nil {
		os.MkdirAll(KeysHome, os.ModePerm)
	}
	author = strings.TrimSpace(author)
	author = strings.ReplaceAll(author, " ", "_")
	author = strings.ReplaceAll(author, "/", "_")
	test := strings.TrimSpace(author + ".key")
	keys, err := os.ReadDir(KeysHome)
	if err != nil {
		log.Println("load keys list err:", err)
	}
	found := false
	for _, k := range keys {
		name := strings.TrimSpace(k.Name())
		// fmt.Println("+", n, author, author+".key", n == author+".key")
		if test == name {
			// fmt.Println("+", name, test)
			key, err := ioutil.ReadFile(filepath.Join(KeysHome, name))
			if err != nil {
				log.Println("load keys err:", err)
				continue
			}
			k := strings.TrimSpace(string(key))
			log.Println("[New Cipher:] ", author, " key:", k)
			stream.Author = author
			stream.ReBildCipherByKey(k)
			found = true
			break
		} else {
			// fmt.Println("com", []byte(test), []byte(name))
		}
	}
	if !found {
		return fmt.Errorf("%s: %s", "can not load cipher", author)

	}
	return
}

func (stream *Stream) ReBildCipherByKey(keyb64 string) {

	key, err := base64.StdEncoding.DecodeString(keyb64)

	if err != nil {
		return
	}
	if len(key) < 32 {
		key = append(key, []byte("asfasivbniasgfbiasgbiasghiashfiashf13412$RASFWEAT!%!@%TRASFSDAT@!#%$!@$")[:32-len(key)]...)
	}
	c, err := aes.NewCipher(key)
	// if there are any errors, handle them
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	stream = new(Stream)
	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		log.Println("generate new cipher err:", keyb64)
	}
	stream.cipher = &gcm
	stream.Key = keyb64
	stream.nonceSize = (*stream.cipher).NonceSize()

}

func (stream *Stream) En(plain []byte) (cipher []byte) {
	// nonceSize := (*stream.cipher).NonceSize()
	nonce := make([]byte, stream.nonceSize)

	// populates our nonce with a cryptographically secure
	// random sequence

	var err error
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		fmt.Println(err)
	}

	// log.Println("nonce sise:", stream.nonceSize, "no:", nonce)
	// log.Println("key:", stream.Key)
	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	cipherText := (*stream.cipher).Seal(nonce, nonce, []byte(plain), nil)
	// cipherText = append(nonce, cipherText...)
	// log.Println("2. nonce sise:", stream.nonceSize, "no:", nonce)
	// log.Println("cipher:", cipherText, len(cipherText))
	return cipherText
}

func (steam *Stream) Flow(raw []byte) (out []byte) {
	key := steam.Key
	L := len(key)
	out = make([]byte, len(raw))
	for n, c := range raw {
		kc := key[n%L]
		ec := c ^ kc
		out[n] = ec
	}
	return

}

func (stream *Stream) FlowEn(raw string) (out string) {
	outraw := stream.Flow([]byte(raw))
	return strings.ReplaceAll(strings.ReplaceAll(base64.StdEncoding.EncodeToString(outraw), "=", "_"), "+", "-")
}

func (stream *Stream) FlowDe(raw64 string) (out string) {
	// outraw := stream.En([]byte(raw))
	raw64 = strings.ReplaceAll(raw64, "_", "=")
	raw64 = strings.ReplaceAll(raw64, "-", "+")
	o, _ := base64.StdEncoding.DecodeString(raw64)
	return string(stream.Flow(o))
}

func (stream *Stream) De(cipher []byte) (plain []byte) {
	// nonce := make([]byte, (*stream.cipher).NonceSize())
	// populates our nonce with a cryptographically secure
	// random sequence
	// log.Println("\ncipher:", cipher, len(cipher))
	// log.Println("\nkey:", stream.Key)
	var err error
	// nonceSize

	if len(cipher) < stream.nonceSize {
		log.Println("cipher de: c<n", err)
	}

	nonce, cipher := cipher[:stream.nonceSize], cipher[stream.nonceSize:]
	// log.Println("\nnonce:", nonce, "\ncipher:", cipher, len(cipher))
	plaintext, err := (*stream.cipher).Open(nil, nonce, cipher, nil)
	if err != nil {
		log.Println("cipher de: open err ,", err)
		log.Println("nonce sise:", stream.nonceSize, "no:", nonce)
	}

	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	return plaintext
}

type EnHeader struct {
	NO int `json:"no"`
	L  int `json:"size"`
}

func (stream *Stream) StreamEncrypt(dst io.Writer, src io.Reader, bar ...func(updated int64)) (copied int64, err error) {
	buf := make([]byte, 8096)
	no := 0
	for {
		n, err := src.Read(buf)

		var network bytes.Buffer // Stand-in for a network connection
		enc := gob.NewEncoder(&network)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("rad from src err:" + err.Error())
			return copied, err
		}
		buf := stream.En(buf[:n])
		head := EnHeader{
			NO: no,
			L:  len(buf),
		}

		err = enc.Encode(head)
		if err != nil {
			log.Println("enc.encode err:", err)
			return copied, err
		}
		network.Write(buf)
		dstBuf := network.Bytes()
		n2, err := dst.Write(dstBuf)
		copied += int64(n2)
		// copied += int64(n2)
		if bar != nil {
			go bar[0](int64(copied))
		}
		if err != nil {
			log.Println("write encripted buf err:" + err.Error())
			return copied, err
		}
		for n2 < len(dstBuf) {
			_n, err := dst.Write(dstBuf[n2:])
			if err != nil {
				log.Println("continue write err:" + err.Error())
				return copied, err
			}
			n2 += _n
			copied += int64(_n)
			if bar != nil {
				go bar[0](int64(copied))
			}
		}

	}
	return
}
func (stream *Stream) StreamDecrypt(dst io.Writer, src io.Reader, bar ...func(uploaded int64)) (copied int64, err error) {
	var network bytes.Buffer // Stand-in for a network connection
	enc := gob.NewEncoder(&network)
	no := 0
	testhead := EnHeader{
		NO: no,
		L:  8096,
	}

	err = enc.Encode(testhead)
	if err != nil {
		log.Println("test head len err:", err)
		return
	}
	headLen := len(network.Bytes())
	// fmt.Println("hl :", headLen, "----")
	headbuf := make([]byte, headLen)
	for {
		nh, err := src.Read(headbuf)
		if err != nil {
			if err == io.EOF {
				// log.Println("Eof")
				break
			}
			log.Println("read header from src err:" + err.Error())
			return copied, err
		}
		if nh < len(headbuf) {
			log.Println("read header err, nh < len(headbuf) :", err)
			return copied, fmt.Errorf("%s:%d", "read header err, nh < len(headbuf)", nh)
		}
		// var network bytes.Buffer // Stand-in for a network connection
		dec := gob.NewDecoder(bytes.NewBuffer(headbuf))
		var thishead EnHeader

		err = dec.Decode(&thishead)
		// copied += int64(n)

		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("parse header from src err:" + err.Error())
			return copied, err
		}
		if thishead.L == 0 {
			log.Println("end")
			break
		}
		cipherBuf := make([]byte, thishead.L)
		n, err := src.Read(cipherBuf)
		if err != nil {
			log.Println("continue read cipher buf err:", err)
			return copied, err
		}

		for n < thishead.L {
			_n, err := src.Read(cipherBuf[n:])
			if err != nil {
				log.Println("continue read cipher buf err:", err)
				return copied, err
			}
			n += _n
		}

		buf := stream.De(cipherBuf[:n])
		// fmt.Println("one :", len(buf))
		n2, err := dst.Write(buf)

		if err != nil {
			log.Println("write encripted buf err:" + err.Error())
			return copied, err
		}
		copied += int64(n2)
		if bar != nil {
			go bar[0](int64(copied))
		}
		for n2 < len(buf) {
			_n, err := dst.Write(buf[n2:])
			if err != nil {
				log.Println("continue write err:" + err.Error())
				return copied, err
			}
			n2 += _n
			copied += int64(_n)
			if bar != nil {
				go bar[0](int64(copied))
			}
		}

	}
	return
}

type CipherFileHeader struct {
	From [100]byte `json:"from"`
	Len  int64     `json:"size"`
}

func (stream *Stream) EncryptFile(plainFile, cipherFile string, bar ...func(int64)) (err error) {
	if stream.Author == "" {
		return fmt.Errorf("%s", "must set author to encrypt file !!!!")
	}
	state, err := os.Stat(plainFile)
	if err != nil || state.IsDir() {
		log.Println("not file found !:" + plainFile)
		return
	}
	fp, err := os.Open(plainFile)
	if err != nil {

		log.Println("open raw file err!", err)
		return
	}
	defer fp.Close()
	wfp, err := os.OpenFile(cipherFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Println("create cipher file err!", err)
		return
	}
	defer wfp.Close()
	buf := [100]byte{}
	l := len(stream.Author)
	if l > 99 {
		l -= 10
	}

	copy(buf[:l], []byte(stream.Author))
	cipher := &CipherFileHeader{
		From: buf,
		Len:  state.Size(),
	}

	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	enc.Encode(&cipher)
	// fmt.Println(len(buffer.Bytes()))
	wfp.Write(buffer.Bytes())
	_, err = stream.StreamEncrypt(wfp, fp, bar...)
	if err != nil {
		log.Println("encript cipher file err!", err)
		return
	}
	return
}

func (stream *Stream) DecryptFile(cipherFile, plainFile string, bar ...func(int64)) (err error) {
	state, err := os.Stat(cipherFile)
	if err != nil || state.IsDir() {
		log.Println("not file found !:" + cipherFile)
		return
	}

	fp, err := os.Open(cipherFile)
	if err != nil {

		log.Println("open raw file err!", err)
		return
	}
	defer fp.Close()
	// reader := bufio.NewReader(fp)
	headline := make([]byte, FILE_CIPHER_HEADER_LEN)
	_, err = fp.Read(headline)
	if err != nil {
		log.Println("read cipherfile header err!", err)
		return
	}
	dec := gob.NewDecoder(bytes.NewBuffer(headline))
	// fmt.Println(headline)
	head := &CipherFileHeader{}
	err = dec.Decode(head)

	if head.Len == 0 {
		log.Println("parse cipherfile json header err!", err)
		return
	}
	fs := bytes.SplitN(head.From[:], []byte{0}, 2)
	from := strings.TrimSpace(string(fs[0]))
	// fmt.Println("try load :", from)
	err = stream.LoadCipherByAuthor(from)
	if err != nil {
		log.Println("load cipher err:", err, head.From)
		return
	}
	wfp, err := os.OpenFile(plainFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Println("create plain file err!", err)
		return
	}
	defer wfp.Close()
	_, err = stream.StreamDecrypt(wfp, fp, bar...)
	if err != nil {
		log.Println("decript cipher file err!", err)
		return
	}
	return
}

func VerifyKey(name, mac string) (exists, group bool) {

	_, err := os.Stat(KeysHome)
	if err != nil {
		os.MkdirAll(KeysHome, os.ModePerm)
	}
	keys, err := os.ReadDir(KeysHome)
	if err != nil {
		log.Println("load keys list err:", err)
	}
	test := name + ".key"
	// found := false
	for _, k := range keys {
		n := strings.TrimSpace(k.Name())
		// fmt.Println("+", n, author, author+".key", n == author+".key")
		if test == n {
			// fmt.Println("+", name, test)
			key, err := ioutil.ReadFile(filepath.Join(KeysHome, n))
			if err != nil {
				log.Println("load keys err:", err)
				continue
			}
			k := strings.TrimSpace(string(key))
			b := md5.Sum([]byte(k))
			res := hex.EncodeToString(b[:])
			if res == strings.TrimSpace(mac) {
				exists = true
				break
			}
		}
	}
	if exists {
		return
	}
	_, err = os.Stat(GroupKeysHome)
	if err != nil {
		os.MkdirAll(GroupKeysHome, os.ModePerm)
	}
	keys, err = os.ReadDir(GroupKeysHome)
	if err != nil {
		log.Println("load keys list err:", err)
	}
	// test = name + ".key"
	// found := false
	for _, k := range keys {
		n := strings.TrimSpace(k.Name())
		// fmt.Println("+", n, author, author+".key", n == author+".key")
		if test == n {
			// fmt.Println("+", name, test)
			key, err := ioutil.ReadFile(filepath.Join(GroupKeysHome, n))
			if err != nil {
				log.Println("load keys err:", err)
				continue
			}
			k := strings.TrimSpace(string(key))
			b := md5.Sum([]byte(k))
			res := hex.EncodeToString(b[:])
			if res == strings.TrimSpace(mac) {
				exists = true
				group = true
				break
			}
		}
	}

	return
}

func LocalKeys() (kmac []string) {

	_, err := os.Stat(KeysHome)
	if err != nil {
		os.MkdirAll(KeysHome, os.ModePerm)
	}
	keys, err := os.ReadDir(KeysHome)
	if err != nil {
		log.Println("load keys list err:", err)
	}
	// found := false
	for _, k := range keys {
		n := strings.TrimSpace(k.Name())
		key, err := ioutil.ReadFile(filepath.Join(KeysHome, n))
		if err != nil {
			log.Println("load keys err:", err)
			continue
		}
		k := strings.TrimSpace(string(key))
		b := md5.Sum([]byte(k))
		res := hex.EncodeToString(b[:])
		kmac = append(kmac, strings.SplitN(n, ".key", 2)[0]+":"+res)
	}
	return
}

func LocalGroupKeys() (kmac []string) {

	_, err := os.Stat(GroupKeysHome)
	if err != nil {
		os.MkdirAll(GroupKeysHome, os.ModePerm)
	}
	keys, err := os.ReadDir(GroupKeysHome)
	if err != nil {
		log.Println("load keys list err:", err)
	}
	// found := false
	for _, k := range keys {
		n := strings.TrimSpace(k.Name())
		key, err := ioutil.ReadFile(filepath.Join(GroupKeysHome, n))
		if err != nil {
			log.Println("load keys err:", err)
			continue
		}
		k := strings.TrimSpace(string(key))
		b := md5.Sum([]byte(k))
		res := hex.EncodeToString(b[:])
		kmac = append(kmac, strings.SplitN(n, ".key", 2)[0]+":"+res)
	}
	return
}

func GetGroupKey(name string) string {

	_, err := os.Stat(GroupKeysHome)
	if err != nil {
		os.MkdirAll(GroupKeysHome, os.ModePerm)
	}
	n := ""
	if strings.Contains(name, GROUP_TAIL) {
		n = strings.TrimSpace(name) + ".key"
	} else {
		n = strings.TrimSpace(name) + GROUP_TAIL + ".key"
	}
	key, err := ioutil.ReadFile(filepath.Join(GroupKeysHome, n))
	if err != nil {
		log.Println("load group keys err:", err, name)
		return ""
	}
	k := strings.TrimSpace(string(key))
	return k
}

func SetGroupKey(name, key string) {

	_, err := os.Stat(GroupKeysHome)
	if err != nil {
		os.MkdirAll(GroupKeysHome, os.ModePerm)
	}
	n := ""
	if strings.Contains(name, GROUP_TAIL) {
		n = strings.TrimSpace(name) + ".key"
	} else {
		n = strings.TrimSpace(name) + GROUP_TAIL + ".key"
	}
	err = ioutil.WriteFile(filepath.Join(GroupKeysHome, n), []byte(key), os.ModePerm)
	if err != nil {
		log.Println("load keys err:", err)
	}

}

func GetKey(name string) string {

	_, err := os.Stat(KeysHome)
	if err != nil {
		os.MkdirAll(KeysHome, os.ModePerm)
	}
	n := strings.TrimSpace(name) + ".key"
	key, err := ioutil.ReadFile(filepath.Join(KeysHome, n))
	if err != nil {
		log.Println("load keys err:", err)
		return ""
	}
	k := strings.TrimSpace(string(key))
	return k
}
func NewKey() string {

	tmpkey := make([]byte, 32)
	rand.Read(tmpkey)
	// saved = false
	if len(tmpkey) < 32 {
		tmpkey = append(tmpkey, []byte("asfasivbniasgfbiasgbiasghiashfiashf13412$RASFWEAT!%!@%TRASFSDAT@!#%$!@$")[:32-len(tmpkey)]...)
	}
	return base64.StdEncoding.EncodeToString(tmpkey)
}

func SetKey(name, key string) {

	_, err := os.Stat(KeysHome)
	if err != nil {
		os.MkdirAll(KeysHome, os.ModePerm)
	}
	n := strings.TrimSpace(name) + ".key"
	err = ioutil.WriteFile(filepath.Join(KeysHome, n), []byte(key), os.ModePerm)
	if err != nil {
		log.Println("load keys err:", err)
	}

}

func ToMd5(key string) string {
	b := md5.Sum([]byte(key))
	res := hex.EncodeToString(b[:])
	return res
}

func SetHome(h string) {
	HOME = h
	KeysHome = filepath.Join(HOME, ".sshchat", "keys")
	GroupKeysHome = filepath.Join(HOME, ".sshchat", "gkeys")
}

func ClearLocalCache() {
	os.RemoveAll(KeysHome)
	os.RemoveAll(GroupKeysHome)
	fmt.Println("remove keys home!!")
}
