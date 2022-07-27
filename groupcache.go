package zhongcache

import (
	"fmt"
	"log"
	"sync"

	"zhongcache/singleflight"
)

// ZhongCache 最核心的数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程
type Group struct {
	name       string
	getter     Getter
	mainCache  cache
	peers      PeerPicker
	loader     *singleflight.Group
}

// 缓存未命中时获取源数据的回调(callback)
type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义了实现 Getter 接口的函数类型
type GetterFunc func(key string) ([]byte, error)

// 接口型函数，既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}


var (
	mut      sync.RWMutex
	groups = make(map[string]*Group)
)

// 实例化 Group，并且将 Group 存储在全局变量 groups 中
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	mut.Lock()
	defer mut.Unlock()

	g := &Group{
		name:       name,
		getter:     getter,
		mainCache:  cache{cacheBytes: cacheBytes},
		loader:     &singleflight.Group{},
	}

	groups[name] = g

	return g
}

// 获取特定名称的 Group
func GetGroup(name string) *Group {
	mut.RLock()
	g := groups[name]
	mut.RUnlock()
	return g
}

// 获取特定 Group 中键为 key 的缓存
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key if required")
	}

	// 从 mainCache 中查找缓存，如果存在则返回缓存值
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[ZhongCache] hit")
		return v, nil
	}

	// 缓存不存在，则调用 load 方法
	return g.load(key)
}

// 将实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}

	g.peers = peers
}

// 使用 PickPeer() 方法选择节点
// 若非本机节点，则调用 getFromPeer() 从远程获取
// 若是本机节点或失败，则回退到 getLocally()
func (g *Group) load(key string) (val ByteView, err error) {
	// 每个 key 只获取一次（不管从本地还是远程节点）
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[ZhongCache] Failed to get from peer", err)
			}
		}
	
		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}

	return
}

// 使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{bytes}, nil
}

// 调用用户回调函数 g.getter.Get() 获取源数据，并且将源数据添加到缓存 mainCache 中
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	val := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, val)

	return val, nil
}

// 将源数据添加到缓存 mainCache 中
func (g *Group) populateCache(key string, val ByteView) {
	g.mainCache.add(key, val)
}