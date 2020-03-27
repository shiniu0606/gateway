package main

import (
	"time"
	"net"
	"io"
	"bufio"
	"strings"
	"strconv"
	"bytes"
	"errors"
	"runtime/debug"
	"crypto/sha1"
	"encoding/base64"
    "encoding/binary"

	log "github.com/shiniu0606/gateway/log"
)

var keyGUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

const (
	// Frame header byte 0 bits from Section 5.2 of RFC 6455
	finalBit = 1 << 7
	rsv1Bit  = 1 << 6
	rsv2Bit  = 1 << 5
	rsv3Bit  = 1 << 4

	// Frame header byte 1 bits from Section 5.2 of RFC 6455
	maskBit = 1 << 7

	maxFrameHeaderSize         = 2 + 8 + 4 // Fixed header + length + mask

)

func websockethandle(c net.Conn) {
	defer func() {
		//log.Info("client closed: "+ c.RemoteAddr().String())
		c.Close()
		if r := recover(); r != nil {
			log.Error("Recovered in", r, ":", string(debug.Stack()))
		}
	}()
	c.SetReadDeadline(time.Now().Add(_ConnReadTimeout))

	rdr := bufio.NewReader(c)
	if !handshake(c) {
        return
	}
	//握手后的第一条消息，传输加密的backend地址
	_,addr,err := readwsframe(c)

	if err != nil {
		if err != io.EOF {
			log.Info("x", err)
		}
		return
	}
	log.Info("ws send addr: ",addr)
	// decrypt addr
	decaddr, err := backendAddrDecrypt(addr)
	if err != nil {
		return
	}
	log.Info("ws send addr: ",decaddr)
	// Build tunnel
	err = tunneling(string(decaddr), rdr, c)
	if err != nil {
		log.Error(err)
	}
}

func writewsframe(fin bool, opcode byte,data []byte,c net.Conn) error {
	return nil
}

//读取一个完整消息
func readwsmessage(c net.Conn) (massage []byte,err error) {
	data := make([]byte, 0)
	for {
		final, message, err := readwsframe(c)
		if final {
			data = append(data, message...)
			break
		} else {
			data = append(data, message...)
		}
		if err != nil {
			return data, err
		}
	}

	return data, nil
}

//ws拆包
func readwsframe(c net.Conn) (fin bool,data []byte,err error) {
	var buf     []byte
	var mask    byte
	buf = make([]byte, 2)
	_, err = io.ReadFull(c, buf)

	if err != nil {
		return true,nil,err
	}

	fin = buf[0]&0x80 != 0

	opcode := buf[0] & 0xf
	if opcode == 8 {
		return true,nil,errors.New("client want close connect")
	}

	mask = buf[1] >> 7

	if rsv := buf[0] & (rsv1Bit | rsv2Bit | rsv3Bit); rsv != 0 {
		return true,nil,errors.New("unexpected reserved bits 0x" + strconv.FormatInt(int64(rsv), 16))
	}

	payload := buf[1] & 0x7f
	var length    uint64	
	var l        uint16
	var mKey    []byte
	// if length < 126 then payload mean the length
	// if length == 126 then the next 8bit mean the length
	// if length == 127 then the next 64bit mean the length
	switch {
	case payload < 126:
		length = uint64(payload)

	case payload == 126:
		buf = make([]byte, 2)
		io.ReadFull(c, buf)
		binary.Read(bytes.NewReader(buf), binary.BigEndian, &l)
		length = uint64(l)

	case payload == 127:
		buf = make([]byte, 8)
		io.ReadFull(c, buf)
		binary.Read(bytes.NewReader(buf), binary.BigEndian, &length)
	}

	if mask == 1 {
		mKey = make([]byte, 4)
		io.ReadFull(c, mKey)
	}

	content := make([]byte, length)
	io.ReadFull(c, content)

	if mask == 1 {
		for i, v := range content {
			content[i] = v ^ mKey[i % 4]
		}
		//fmt.Print("mask", mKey)
	}
	log.Info(string(content))

	return fin,content,nil
}

func handshake(c net.Conn) bool {
	reader := bufio.NewReader(c)
	key := ""
    str := ""
    for {
        line, _, err := reader.ReadLine()
        if err != nil {
            log.Error("Handshake err:",err)
            return false
        }
        if len(line) == 0 {
            break
        }
        str = string(line)
        if strings.HasPrefix(str, "Sec-WebSocket-Key") {
            if len(line)>= 43 {
                key = str[19:43]
            }
        }
	}
	
	if key == "" {
        return false
	}
	
	sha := sha1.New()
    sha.Write([]byte(key))
	sha.Write(keyGUID)
    key = base64.StdEncoding.EncodeToString(sha.Sum(nil))
    header := "HTTP/1.1 101 Switching Protocols\r\n" +
    "Connection: Upgrade\r\n" +
    "Sec-WebSocket-Version: 13\r\n" +
    "Sec-WebSocket-Accept: " + key + "\r\n" +
	"Upgrade: websocket\r\n\r\n"
	
	//upgrade 
	c.Write([]byte(header))

	return true
}


func listenWsServe() {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(_WebsocketPort))
	if err != nil {
		log.Fatal(err)
	}

	defer l.Close()
	var tempDelay time.Duration
	for {
		conn, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			log.Fatal(err)
		}
		tempDelay = 0
		go websockethandle(conn)
	}
}
