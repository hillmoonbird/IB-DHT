package zhongcache

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"zhongcache/consistenthash"
)

const (
	defaultBasePath = "/_zhongcache/" // 访问路径的默认前缀
	defaultReplicas = 50              // 默认虚拟节点倍数
)

// 承载节点间 HTTP 通信的核心数据结构
type HTTPPool struct {
	self        string                 // 自身地址，包括主机名/IP和端口
	basePath    string                 // 节点间通讯地址的前缀
	mut         sync.Mutex             // 互斥锁
	peers       *consistenthash.Map    // 一致性哈希的 Map，用来根据具体的 key 选择节点
	httpGetters map[string]*httpGetter // 远程节点与对应 httpGetter 之间的映射
}

// HTTPPool 的实例化函数
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// HTTPPool 日志打印
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// HTTP 服务主体
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 若请求报文的访问路径前缀不适配，则报错
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	// 打印日志
	p.Log("%s %s", r.Method, r.URL.Path)

	// 切分请求报文的 URL
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// 解析出组名和键
	groupName := parts[0]
	key := parts[1]

	// 获取 group 对象
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group:"+groupName, http.StatusNotFound)
	}

	// 获取 group 中 key 对应的缓存
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// 将得到的缓存写入响应报文
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// 实例化一致性哈希算法，添加传入的节点
func (p *HTTPPool) Set(peers ...string) {
	p.mut.Lock()
	defer p.mut.Unlock()

	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		// 为每一个节点创建一个 HTTP 客户端 httpGetter
		p.httpGetters[peer] = &httpGetter{peer + p.basePath}
	}
}

// 包装了一致性哈希算法的 Get() 方法，根据具体的 key，选择节点，返回节点对应的 HTTP 客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mut.Lock()
	defer p.mut.Unlock()

	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}

	return nil, false
}

// 让编译器检查类型 HTTPPool 是否实现了接口 PeerPicker
var _ PeerPicker = (*HTTPPool)(nil)

// HTTP 客户端类
type httpGetter struct {
	baseURL string //将要访问的远程节点的地址
}

// 使用 http.Get() 方式获取返回值，并转换为 []bytes 类型
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)

	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	// 确保 http 事务完成，相关资源得以回收
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

// 让编译器检查类型 httpGetter 是否实现了接口 PeerGetter
var _ PeerGetter = (*httpGetter)(nil)
