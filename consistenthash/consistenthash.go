package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 哈希函数类型
type Hash func(data []byte) uint32

// 一致性哈希算法的主要数据结构
type Map struct {
	hash     Hash            // Hash 函数
	replicas int             // 虚拟节点倍数
	keys     []int           // 哈希环
	hashMap  map[int]string  // 虚拟节点与真实节点的映射表
}

// Map 实例化函数，允许自定义虚拟节点倍数和 Hash 函数
func New(rpl int, fx Hash) *Map {
	m := &Map{
		hash:     fx,
		replicas: rpl,
		hashMap:  make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}

	return m
}

// 节点添加函数（真实/虚拟节点）
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			// 计算每一个虚拟节点的哈希值
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 将该节点哈希值加入哈希环
			m.keys = append(m.keys, hash)
			// 建立虚拟节点与真实节点的映射关系
			m.hashMap[hash] = key
		}
	}

	// 使哈希环保持有序
	sort.Ints(m.keys)
}

// 获取键为 key 的结点对应的真实节点
func (m *Map) Get(key string) string {
	if (len(m.keys)) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx % len(m.keys)]]
}