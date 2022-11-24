package singleflight

import (
	"sync"
)

// 代表正在进行中或已经结束的请求
type call struct {
	wg  sync.WaitGroup  // 避免重入的锁
	val interface{}
	err error
}

// singleflight 的主数据结构，管理不同 key 的请求(call)
type Group struct {
	mut sync.Mutex
	m   map[string]*call
}

// 针对相同的 key，无论 Do 被调用多少次，函数 fx 都只会被调用一次，等待 fx 调用结束了，返回返回值或错误
func (g *Group) Do(key string, fx func() (interface{}, error)) (interface{}, error) {
	// g.mut 是保护 Group 的成员变量 m 不被并发读写而加上的锁
	g.mut.Lock()
	// 延迟初始化，提高内存使用效率
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	if c, ok := g.m[key]; ok {
		g.mut.Unlock()     
		c.wg.Wait()         // 如果请求正在进行中，则等待
		return c.val, c.err // 请求结束，返回结果
	}

	c := new(call)
	c.wg.Add(1)     // 发起请求前加锁
	g.m[key] = c    // 添加到 g.m，表明 key 已经有对应的请求在处理
	g.mut.Unlock()

	c.val, c.err = fx() // 调用 fn，发起请求
	c.wg.Done()         // 请求结束

	g.mut.Lock()
	delete(g.m, key)    // 更新 g.m
	g.mut.Unlock()

	return c.val, c.err // 返回结果
}