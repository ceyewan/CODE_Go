package geecache

// PeerPicker 是必须实现的接口，用于定位特定 key 对应的 peer
type PeerPicker interface {
	// 用于根据 key 来确定相应节点 PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 是 peer 必须实现的接口
type PeerGetter interface {
	// 用于从对应的 group 查找缓存值
	Get(group string, key string) ([]byte, error)
}
