package main 

import (
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"strconv"
	"net"
	"strings"
	"os/signal"

	_ "net/http/pprof"
	log "github.com/shiniu0606/gateway/log"
)

const (
	// max open file should at least be
	_MaxOpenfile              = uint64(1024 * 1024 * 1024)
	_MaxBackendAddrCacheCount = 1024 * 1024
)

var (
	_SecretPassphase []byte
)

var (
	_BackendAddrCacheMutex sync.Mutex
	_BackendAddrCache      atomic.Value
)

var (
	_DefaultPort        = 4399
	_WebsocketPort	    = 4398
)

type backendAddrMap map[string]string

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	os.Setenv("GOTRACEBACK", "crash")

	log.InitLog(AppConfigs.InfoLogPath,AppConfigs.ErrorLogPath)

	_BackendAddrCache.Store(make(backendAddrMap))

	//hack too many open files 1024
	var lim syscall.Rlimit
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim)
	if lim.Cur < _MaxOpenfile || lim.Max < _MaxOpenfile {
		lim.Cur = _MaxOpenfile
		lim.Max = _MaxOpenfile
		syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lim)
	}

	_SecretPassphase = []byte(AppConfigs.Secret)

	listenPort, err := strconv.Atoi(AppConfigs.DefaultPort)
	if err == nil && listenPort > 0 && listenPort <= 65535 {
		_DefaultPort = listenPort
	}

	listenPort, err = strconv.Atoi(AppConfigs.WebScoketPort)
	if err == nil && listenPort > 0 && listenPort <= 65535 {
		_WebsocketPort = listenPort
	}

	//pprofPort, err := strconv.Atoi(os.Getenv("PPROF_PORT"))
	//if err == nil && pprofPort > 0 && pprofPort <= 65535 {
	//	go func() {
	//		http.ListenAndServe(":"+strconv.Itoa(pprofPort), nil)
	//	}()
	//}
	go listenWsServe()
	log.Info("listenAndServe start :",AppConfigs.DefaultPort)
	listenAndServe()

	// catchs system signal
	chSig := make(chan os.Signal)
	signal.Notify(chSig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTERM)
	sig := <-chSig
	log.Info("siginal:", sig)
}

func listenAndServe() {
	l, err := net.Listen("tcp", ":"+strconv.Itoa(_DefaultPort))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}
}

func backendAddrDecrypt(key []byte) (string, error) {
	// Try to check cache
	m1 := _BackendAddrCache.Load().(backendAddrMap)
	k1 := string(key)
	addr, ok := m1[k1]
	if ok {
		return addr, nil
	}

	// Try to decrypt it (AES)
	log.Info("_SecretPassphase:"+string(_SecretPassphase))
	log.Info("key:"+string(key))
	addr, err := AesDecrypt(string(key), string(_SecretPassphase))
	if err != nil {
		return "127.0.0.1:8888", err
	}

	backendAddrList(k1, addr)
	return addr, nil
}

func backendAddrList(key string, val string) {
	_BackendAddrCacheMutex.Lock()
	defer _BackendAddrCacheMutex.Unlock()

	m1 := _BackendAddrCache.Load().(backendAddrMap)
	// double check
	if _, ok := m1[key]; ok {
		return
	}

	m2 := make(backendAddrMap)
	// flush cache if there is way too many
	if len(m1) < _MaxBackendAddrCacheCount {
		// copy-on-write
		for k, v := range m1 {
			m2[k] = v // copy all data from the current object to the new one
		}
	}
	m2[key] = val
	_BackendAddrCache.Store(m2) // atomically replace the current object with the new one
}

func ipAddrFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}
