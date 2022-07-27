package zhongcache

// ByteView 是用来表示缓存值的抽象的只读数据结构
type ByteView struct {
	// 选择 byte 类型是为了能够支持任意数据类型的存储
	b []byte
}

// 实现 lru.Cache 中的 Value 接口
func (v ByteView) Len() int {
	return len(v.b)
}

// 返回只读对象的拷贝，防止缓存值被外部程序修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

// 以拷贝的形式将 ByteView 中的 []byte 转换为 string
func (v ByteView) String() string {
	return string(v.b)
}
