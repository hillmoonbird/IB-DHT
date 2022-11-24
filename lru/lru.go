package lru

import (
	"container/list"
)


// Cache 是一个 LRU 缓存，非并发安全
type Cache struct {
	maxBytes    int64                          // 允许使用的最大内存         
	usedBytes   int64                          // 当前已使用的内存
	dList       *list.List                     // 双向链表，存储值
	cMap        map[string]*list.Element       // 字典，存储键和值的映射关系
	OnEvicted   func(key string, value Value)  // 某条记录被移除时的回调函数，可以为 nil
}


// 键值对 Entry 是双向链表节点的数据类型
// 链表中仍保存每个值对应的 key，淘汰队首节点时，可以根据 key 从字典中删除对应的映射
type Entry struct {
	key    string
	value  Value
}


// 为了通用性，我们允许值是实现了 Value 接口的任意类型
// 该接口只包含了一个方法 Len() int，用于返回值所占用的内存大小
type Value interface {
	Len() int
}


// Cache 实例化函数
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		dList:     list.New(),
		cMap:      make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}


// Cache 查找函数
func (c *Cache) Get(key string) (val Value, ok bool) {
	if ele, ok := c.cMap[key]; ok {
		// 在字典中找到对应双向链表节点后，将该节点移动到队尾
		c.dList.MoveToBack(ele)
		kv := ele.Value.(*Entry)
		return kv.value, true
	}
	return
}


// Cache 删除函数（缓存淘汰）
func (c *Cache) RemoveLRU() {
	ele := c.dList.Front()
	if ele != nil {
		// 取到队首节点，从链表中删除
		c.dList.Remove(ele)
		kv := ele.Value.(*Entry)
		// 从字典中删除该节点的映射关系
		delete(c.cMap, kv.key)
		// 更新当前所用的内存
		c.usedBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}


// Cache 新增/修改函数
func (c *Cache) Add(key string, val Value) {
	if ele, ok := c.cMap[key]; ok {
		// 若键存在，则更新对应节点的值，并将该节点移到队尾
		c.dList.MoveToBack(ele)
		kv := ele.Value.(*Entry)
		c.usedBytes += int64(val.Len()) - int64(kv.value.Len())
		kv.value = val
	} else {
		// 若键不存在，则添加一个新节点到队尾
		ele := c.dList.PushBack(&Entry{key, val})
		c.cMap[key] = ele
		c.usedBytes += int64(len(key)) + int64(val.Len())
	}
	// 若已用内存超过设定的最大值，则移除 LRU 节点
	for c.maxBytes != 0 && c.usedBytes > c.maxBytes {
		c.RemoveLRU()
	}
}


// 获取已添加数据的条数
func (c *Cache) Len() int {
	return c.dList.Len()
}