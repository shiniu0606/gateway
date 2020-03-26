package main 

import (
	"bufio"
	"time"
	"fmt"
	"io"
	"runtime/debug"
	"net"
	"strings"

	log "github.com/shiniu0606/gateway/log"
)

var (
	_BackendDialTimeout = 5
	_ConnReadTimeout    = time.Second * 30
)

func writeErrCode(c net.Conn, errCode []byte, httpws bool) {
	switch httpws {
	case true:
		fmt.Fprintf(c, "HTTP/1.1 %s Error\nConnection: Close", errCode)
	default:
		c.Write(errCode)
	}
}

func handleConn(c net.Conn) {
	defer func() {
		log.Info("client closed: "+ ipAddrFromRemoteAddr(c.RemoteAddr().String()))
		c.Close()
		if r := recover(); r != nil {
			log.Error("Recovered in", r, ":", string(debug.Stack()))
		}
	}()
	log.Info("client connect: "+ ipAddrFromRemoteAddr(c.RemoteAddr().String()))
	c.SetReadDeadline(time.Now().Add(_ConnReadTimeout))

	rdr := bufio.NewReader(c)

	// Read first byte
	b, err := rdr.ReadByte()
	if err != nil {
		// TODO: how to cause error to test this?
		writeErrCode(c, []byte("1101"), false)
		return
	}

	//first byte 0x90 decrypt addr
	if b == byte(0x90) {
		writeErrCode(c, []byte("1102"), false)
		return
	}

	// binary protocol get len
	blen, err := rdr.ReadByte()

	if err != nil || blen == 0 {
		writeErrCode(c, []byte("1103"), false)
		return
	}

	//read body
	p := make([]byte, blen)
	n, err := io.ReadFull(rdr, p)
	if n != int(blen) {
		// TODO: how to cause error to test this?
		writeErrCode(c, []byte("1104"), false)
		return
	}

	// decrypt addr
	addr, err := backendAddrDecrypt(p)
	if err != nil {
		writeErrCode(c, []byte("1105"), false)
		return
	}
	

	// Build tunnel
	err = tunneling(string(addr), rdr, c)
	if err != nil {
		log.Error(err)
	}
}

// tunneling to backend
func tunneling(addr string, rdr *bufio.Reader, c net.Conn) error {
	backend, err := dialTimeout("tcp", addr, time.Second*time.Duration(_BackendDialTimeout))
	if err != nil {
		// handle error
		switch err := err.(type) {
		case net.Error:
			if err.Timeout() {
				writeErrCode(c, []byte("4101"), false)
				return err
			}
		}
		writeErrCode(c, []byte("4102"), false)
		return err
	}
	defer backend.Close()

	// Start transfering data
	go pipe(c, backend, c, backend)
	pipe(backend, rdr, backend, c)

	return nil
}

// pipe upstream and downstream
func pipe(dst io.Writer, src io.Reader, dstconn, srcconn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovered in", r, ":", string(debug.Stack()))
		}
	}()

	// only close dst when done
	defer dstconn.Close()

	buf := make([]byte, 2*4096)
	for {
		srcconn.SetReadDeadline(time.Now().Add(_ConnReadTimeout))
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if ew != nil {
				break
			}
			if nr != nw {
				break
			}
		}
		if neterr, ok := er.(net.Error); ok && neterr.Timeout() {
			continue
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			break
		}
	}
}

func dialTimeout(network, address string, timeout time.Duration) (conn net.Conn, err error) {
	m := int(timeout / time.Second)
	for i := 0; i < m; i++ {
		conn, err = net.DialTimeout(network, address, timeout)
		if err == nil || !strings.Contains(err.Error(), "can't assign requested address") {
			break
		}
		time.Sleep(time.Second)
	}
	return
}

