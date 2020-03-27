package main

import (
	"testing"
	"flag"
	"time"
	"math/rand"
	"encoding/hex"
	"errors"
	"os"
	"net"
	"fmt"
	"io"
	"bytes"
	"net/url"
	"log"
	"strconv"

	"github.com/gorilla/websocket"
)

var (
	_echoServerAddr      = "127.0.0.1:62863"
	_echoWebsocketServerAddr      = "127.0.0.1:62862"
	_defaultFrontdAddr   = "127.0.0.1:4399"
	_defaultWsdAddr   = "127.0.0.1:4398"
)

func TestMain(m *testing.M){
	flag.Parse()

	// start echo server
	go servEcho()
	go servWebsocketToTcp()
	go main()

	rand.Seed(time.Now().UnixNano())
	
	// wait for servers to start
	time.Sleep(time.Second)
	os.Exit(m.Run())
}

func servWebsocketToTcp() {
	l, err := net.Listen("tcp", string(_echoWebsocketServerAddr))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + string(_echoWebsocketServerAddr))
	for {
		// Listen for an incoming connection.
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go func(c net.Conn) {
			defer c.Close()

			for {
				message,_ := ReadWsMessage(c)
				if len(message) > 0 {
					fmt.Println("message:",string(message))
				}

				SendText(c,[]byte("server "+string(message)))
			}
		}(c)
	}
}

func servEcho() {
	l, err := net.Listen("tcp", string(_echoServerAddr))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + string(_echoServerAddr))
	for {
		// Listen for an incoming connection.
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go func(c net.Conn) {
			defer c.Close()

			_, err := io.Copy(c, c)
			switch err {
			case io.EOF:
				err = nil
				return
			case nil:
				return
			}
			panic(err)
		}(c)
	}
}

func TestEchoServer(t *testing.T) {
	var conn net.Conn
	var err error
	conn, err = dialTimeout("tcp", string(_echoServerAddr), time.Second*time.Duration(_BackendDialTimeout))

	if err != nil {
		panic(err)
	}
	defer conn.Close()

	n := rand.Int() % 10
	for i := 0; i < n; i++ {
		testEchoRound(conn)
	}
}

func testEchoRound(conn net.Conn) {
	conn.SetDeadline(time.Now().Add(time.Second * 10))

	n := rand.Int()%2048 + 10
	out := randomBytes(n)
	n0, err := conn.Write(out)
	if err != nil {
		panic(err)
	}

	rcv := make([]byte, n)
	n1, err := io.ReadFull(conn, rcv)
	if err != nil && err != io.EOF {
		panic(err)
	}
	if !bytes.Equal(out[:n0], rcv[:n1]) {
		fmt.Println("out: ", n0, "in:", n1)

		fmt.Println("out: ", hex.EncodeToString(out), "in:", hex.EncodeToString(rcv))
		panic(errors.New("echo server reply is not match"))
	}
}

func randomBytes(n int) []byte {

	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i := 0; i < n; i++ {
		b[i] = byte(rand.Int())
	}

	return b
}

func encryptText(plaintext, passphrase string) (string, error) {
	return AesEncrypt(plaintext, passphrase)
}

func TestProtocolDecrypt(*testing.T) {
	b, err := encryptText(_echoServerAddr, "4d4cd0e76aecc5eca4dc322eaad3448b")
	if err != nil {
		panic(err)
	}
	head := append([]byte{0x90}, byte(len(b)))
	testProtocol(append(head, []byte(b)...))

	// test cached hitted
	testProtocol(append(head, []byte(b)...))
}

func TestWebsocketDecrypt(*testing.T) {
	b, err := encryptText(_echoWebsocketServerAddr, "4d4cd0e76aecc5eca4dc322eaad3448b")
	if err != nil {
		panic(err)
	}
	testWebsocket([]byte(b))

	// test cached hitted
	testWebsocket([]byte(b))
}

func testWebsocket(cipherAddr []byte) {
	u := url.URL{Scheme: "ws", Host: "localhost:4398", Path: "/"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	err = c.WriteMessage(websocket.TextMessage, cipherAddr)
	if err != nil {
		log.Println("write:", err)
		return
	}

	done := make(chan struct{})
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for i := 0; i < 5; i++ {
		c.WriteMessage(websocket.TextMessage, []byte("send:"+strconv.Itoa(i)))
	}

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		}
	}
}

func testProtocol(cipherAddr []byte) {
	// * test decryption
	var conn net.Conn
	var err error
	conn, err = dialTimeout("tcp", _defaultFrontdAddr, time.Second*time.Duration(_BackendDialTimeout))

	if err != nil {
		panic(err)
	}
	defer conn.Close()
	//fmt.Println(cipherAddr[:len(cipherAddr)])
	_, err = conn.Write(cipherAddr)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 5; i++ {
		testEchoRound(conn)
	}
}

func BenchmarkEcho(b *testing.B) {
	//for i := 0; i < b.N; i++ {
	//	TestEchoServer(&testing.T{})
	//}
}

func BenchmarkNoHitLatency(b *testing.B) {
	for i := 0; i < 10; i++ {
		TestProtocolDecrypt(&testing.T{})
	}
}

func BenchmarkWsNoHitLatency(b *testing.B) {
	for i := 0; i < 10; i++ {
		TestWebsocketDecrypt(&testing.T{})
	}
}
// func BenchmarkNoHitLatencyParallel(b *testing.B) {
// 	b.RunParallel(func(pb *testing.PB) {
// 		for pb.Next() {
// 			TestProtocolDecrypt(&testing.T{})
// 		}
// 	})
// }
