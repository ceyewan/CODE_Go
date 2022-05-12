package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash           // 哈希函数
	replicas int            // 每个 key 对应几个虚拟节点
	keys     []int          // 哈希环
	hashMap  map[int]string // 虚拟节点映射到真实节点
}

// 创建一个 Map
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	// 默认哈希函数，但是也可以自行实现
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加节点，我们多了设备了，那么就得添加节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			// 虚拟节点 i-key
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 节点增加
			m.keys = append(m.keys, hash)
			// 虚拟节点到真实节点的映射
			m.hashMap[hash] = key
		}
	}
	// 保持节点有序，便于后续查找
	sort.Ints(m.keys)
}

// 返回获取 key 这个数据应该去哪里找
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	// 求出我们需要的值的哈希值
	hash := int(m.hash([]byte(key)))
	// 找到需要值后面阿那个节点（可能是 n 那么转化为 0）
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	// 返回真实节点
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
