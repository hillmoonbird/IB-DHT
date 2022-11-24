package zhongcache

import (
	"sync"

	"zhongcache/lru"
)

// 添加了并发特性的 lru.Cache
type cache struct {
	mut         sync.Mutex
	lru         *lru.Cache
	cacheBytes  int64
}

// 通过互斥锁实现并发的 add()方法
func (c *cache) add(key string, val ByteView) {
	c.mut.Lock()
	defer c.mut.Unlock()

	// 延迟初始化：对象的创建延迟至第一次使用时
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}

	c.lru.Add(key, val)
}

// 通过互斥锁实现并发的 get() 方法
func (c *cache) get(key string) (val ByteView, ok bool) {
	c.mut.Lock()
	defer c.mut.Unlock()

	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}

	return
}