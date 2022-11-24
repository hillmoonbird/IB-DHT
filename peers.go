package zhongcache

// HTTP 调度器
type PeerPicker interface {
	// 根据传入的 key 选择相应节点 PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// HTTP 客户端
type PeerGetter interface {
	// 从对应 group 中查找缓存值
	Get(group string, key string) ([]byte, error)
}