package registry

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type GeeRegistry struct {
	timeout time.Duration
	mu      sync.Mutex
	servers map[string]*SeverItem
}

type SeverItem struct {
	Addr  string
	start time.Time
}

// 默认超时时间设置为 5 min，任何注册的服务超过 5 min，即视为不可用状态。
const (
	defaultPath    = "/_geerpc_/registry"
	defaultTimeout = time.Minute * 5
)

func NewGeeRegistry(timeout time.Duration) *GeeRegistry {
	return &GeeRegistry{
		servers: make(map[string]*SeverItem),
		timeout: timeout,
	}
}

var DefaultGeeRegister = NewGeeRegistry(defaultTimeout)

// 添加服务实例，如果服务已经存在，则更新 start
func (r *GeeRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &SeverItem{addr, time.Now()}
	} else {
		s.start = time.Now()
	}
}

// aliveServers：返回可用的服务列表，如果存在超时的服务，则删除
func (r *GeeRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var aliveAddrs []string
	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			aliveAddrs = append(aliveAddrs, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	return aliveAddrs
}

// GeeRegistry 采用 HTTP 协议提供服务，且所有的有用信息都承载在 HTTP Header 中。
// Get：返回所有可用的服务列表，通过自定义字段 X-Geerpc-Servers 承载。
// Post：添加服务实例或发送心跳，通过自定义字段 X-Geerpc-Server 承载。
func (r *GeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		w.Header().Set("X-Geerpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		addr := req.Header.Get("X-Geerpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *GeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultGeeRegister.HandleHTTP(defaultPath)
}

// 定时向注册中心发送心跳，默认周期比注册中心设置的过期时间少 1 min。
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		duration = defaultTimeout - time.Minute*1
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		defer t.Stop()
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Geerpc-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
