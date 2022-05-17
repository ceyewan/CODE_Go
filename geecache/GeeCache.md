设计一个分布式缓存系统，需要考虑**资源控制**、**淘汰策略**、**并发**、**分布式节点通信**等各个方面的问题。而且，针对不同的应用场景，还需要在不同的特性之间权衡，例如，是否需要支持缓存更新？还是假定缓存在淘汰之前是不允许改变的。不同的权衡对应着不同的实现。

`GeeCache` 基本上模仿了 [groupcache](https://github.com/golang/groupcache) 的实现。

### LRU 缓存淘汰策略

淘汰策略主要有 FIFO(First In First Out)、LFU(Least Frequently Used) 和 LRU(Least Recently Used) 。LFU 需要记录访问次数，LRU 只需要使用时间的远近，通过离链表头/尾部的距离来体现。

LRU 需要用到两个主要的数据结构，map 和 list 我们通过 map 将 key 映射到 list 的某个节点，然后通过 list 决定淘汰策略。

```go
package lru

import "container/list"

type Cache struct {
	maxBytes  int64                         // 允许使用的最大内存
	nbytes    int64                         // 已经使用的内存
	ll        *list.List                    // 双向链表
	cache     map[string]*list.Element      // 字符串到节点的映射
	OnEvicted func(key string, value Value) // 某条记录被移除时的回调函数
}

// 存储到 list 中的数据，包括 key 和 value
type entry struct {
	key   string
	value Value
}

// Value 只有一个接口，谁实现了接口，谁就可以是 Value
// 也可以认为是一种泛型编程思想
type Value interface {
	Len() int
}
```

接下来我们需要实现 lru 的 New 方法，

```go
// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}
```

 我们接下来分别实现增删改查等功能，

```go
// list 里面的 Value 其实也是一个接口，那么我们需要使用 .(*entry) 来断言
// 其实不用也没什么关系，但是谁知道呢，用着用着里面存的不一定就是 *entry 了
// 这里其实移动的是节点，而不是节点内的数字，因此没有必要修改 cache
func (c *Cache) Get(key string) (value Value, ok bool) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		kv := element.Value.(*entry) // 接口断言
		return kv.value, true
	}
	return
}

func (c *Cache) RemoveOldest() {
	element := c.ll.Back()
	if element != nil {
		c.ll.Remove(element)
		kv := element.Value.(*entry)
		delete(c.cache, kv.key) // 如果 entry 中不存 key 那么就需要遍历 cache
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		kv := element.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		element := c.ll.PushFront(&entry{key, value})
		c.cache[key] = element
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
    // for 循环，删一个不一定够的
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
```

这里提一嘴回调函数，我感觉这个函数就是用来测试用的，因为我们知道，remove 是没有返回值的，因此，我们可以通过在 remove 的时候处理回调函数，从而用来测试。

```go
type String string

func (s String) Len() int {
    return len(s)
}

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lru := New(int64(10), callback)
	lru.Add("key1", String("123456"))
	lru.Add("k2", String("k2"))
	lru.Add("k3", String("k3"))
	lru.Add("k4", String("k4"))

	expect := []string{"key1", "k2"}

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equal to %s", expect)
	}
}
// 写好之后 IDE 应该会有 run test 按钮
// 也可以在命令行中执行 go test [-run] [xxxTest](测试特定函数)
```

### 单机并发缓存

我们抽象一个只读数据结构 `ByteView` 用来表示缓存值，是 GeeCache 主要的数据结构之一。为什么要用它来表示我们的缓存内容呢？因为所有其他的数据类型都是 byte 组成的，包括图片视频，这个适用度最广。

```go
type ByteView struct {
	b []byte
}

// 实现 Len() 方法，就能作为 lru 的 Value 用了
func (v ByteView) Len() int {
	return len(v.b)
}

// 切片方法，返回的是 copy ，所以不会被修改，保持只读属性
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// 转字符串输出方法
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
```

接下来就是为 lru 添加并发属性，这也很简单，

```go
type cache struct {
	mu         sync.Mutex // 互斥锁
	lru        *lru.Cache // lru
	cacheBytes int64      // lru 的大小
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
    // 当我们要用的时候再创建，这种方法称为延迟初始化
    // 一个对象的延迟初始化意味着该对象的创建将会延迟至第一次使用该对象时
    // 主要用于提高性能，并减少程序内存要求。
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok // 这也是断言，返回的 v 为 ByteView 类型
	}
	return
}
```

接下来，我们写主体结构 Group ，这是 geecache 最核心的数据结构，负责与用户交互，并且控制缓存值存储和获取的流程。

```
                          	是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
```

1. 回调 Getter 

如果缓存不存在，我们调用一个回调函数(callback)，得到源数据，至于从何而得，这是用户决定是事情。****

```go
// 定义了一个接口，只包含一个方法
type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义了一个函数类型，和 get 的参数、返回值是一样的
type GetterFunc func(key string) ([]byte, error)

// GetterFunc 定义 Get 方式，并在 Get 方法中调用自己，这样就实现了接口 Getter
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}
// GetterFunc 是一个实现了接口的函数类型，简称为接口型函数
```

接口型函数只能应用于接口内部只定义了一个方法的情况。

定义一个函数类型 F，并且实现接口 A 的方法，然后在这个方法中调用自己。这是 Go 语言中将其他函数（参数返回值定义与 F 一致）转换为接口 A 的常用技巧。

2. Group 的定义

```go
// Group 有名字，回调函数，cache
type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

// 这里声明的一个读写锁和 Groups （很多个 cache ，分布式
var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// 创建一个 group 实例，需要加全局锁，因为涉及到修改 Groups
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// 通过 name 拿到某个 Group 
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// 从某个 Group 中拿到数据
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	// 拿到了
	if v, ok := g.mainCache.get(key); ok {
		log.Println("GeeCache hit")
		return v, nil
	}
    // 没有从 cache 中拿到
	return g.load(key)
}

// 调用 getLocally ，那么 load 有什么存在的必要吗？
func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

// 调用回调函数的 Get 去获取数据（至于从哪里，不管
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
    // 获取的数据 copy yi'ge
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
```

### HTTP 服务端

首先创建一个结构体 `HTTPPool` 用于承载节点间 HTTP 通信的核心数据结构（包括服务端和客户端）。现在，我们先实现服务端。

```go
const defaultBasePath = "/_geecache/"

// HTTPPool 为 HTTP 节点池实现 PeerPicker
type HTTPPool struct {
	self     string // 用于记录自己的主机号和端口
	basePath string // 作为节点通讯地址的前缀
}

// New 一个 HTTPPool
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}
```

接下来，实现最为核心的 ServeHTTP 方法，

```go
// Log 打印日志信息
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 处理所有的 http 请求
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 访问路径前缀必须合理
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> 格式
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// groupname 和 key
	groupName := parts[0]
	key := parts[1]
	// 拿到对应的 group
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	// 从 group 中拿到数据
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 发送数据
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}
```

那么此时，我们的服务端就写好了，接收对应的 url ，然后查找数据，返回。

### 一致性哈希

对于分布式缓存来说，当一个节点接收到请求，如果该节点并没有存储缓存值，那么它面临的难题是，从谁那获取数据？自己，还是节点1, 2, 3, 4… 。假设包括自己在内一共有 10 个节点，当一个节点接收到请求时，随机选择一个节点，由该节点从数据源获取数据。

我们使用简单的对 10 取余也可以，但是，节点数量变化了怎么办，少了一个节点，变成对 9 取余，那么之前的缓存全部失效，容易造成缓存雪崩。

> 缓存雪崩：缓存在同一时刻全部失效，造成瞬时DB请求量大、压力骤增，引起雪崩。常因为缓存服务器宕机，或缓存设置了相同的过期时间引起。

我们使用一致性哈希算法，将所有的 key 映射到 32 位数值上，然后将我们的节点也映射到数值上，然后 key 从后面找最近的一个节点，用这个节点来处理操作。但是，如果一个节点失效，他的任务会全部交给他后面的那个节点，造成任务倾斜。我们可以用虚拟节点来处理，每个真实节点有多个虚拟节点在长度为 $2^{32}$    的圆环上，如果失效了，那么这些虚拟节点上的任务就会各自顺移到后面的那个虚拟节点，因为哈希算法的随机性，我们可以认为这个节点的任务被均摊到了其他节点上。

```go
type Hash func(data []byte) uint32

type Map struct {
	hash     Hash           // 哈希函数
	replicas int            // 每个 key 对应几个虚拟节点
	keys     []int          // 哈希环 // 所有的虚拟节点的值
	hashMap  map[int]string // 虚拟节点映射到真实节点
}

// New 创建一个 Map
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
```

当我们增加机器的时候，那么就需要添加节点

```go
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
```

当我们需要查找数据时，确定去哪个节点找：

```go
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
```

### 分布式节点

```go
// PeerPicker 是必须实现的接口，用于定位特定 key 对应的 peer
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 是 peer 必须实现的接口
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
```

这里我们抽象了两个接口，PeerPicker 的 PickPeer 方法根据传入的 key 值选择相应的节点 PeeGetter 。

因为 PeerGetter 就是通过 group 和 key 来查找缓存值的，并且我们的节点需要实现这个功能。

接下来我们为 HTTPPool 实现客户端功能，首先我们创建 httpGetter 实现 PeerGetter 接口。

```go
type httpGetter struct {
	baseURL string // 将要访问的远程节点的地址
}

// 通过 group 和 key 拿到我们的数据
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	// u = http://example.com/_geecache/group/key
	// 这里我们使用了 http.Get, 谁来响应？实现了 serveHTTP 的 HTTPPool
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
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

var _ PeerGetter = (*httpGetter)(nil)
```

那么我们的 HTTPPool 需要实现响应，那么我们为他添加节点选择功能，

```go
const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)
// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	self        string                 // 用来记录自己的地址（ip和端口）
	basePath    string                 // 作为节点间通讯地址的前缀，默认 "url/_geecache/"
	mu          sync.Mutex             // 锁
	peers       *consistenthash.Map    // 一致性哈希的 Map 根据 key 选择节点
	httpGetters map[string]*httpGetter // 映射远程节点与对应的 httpGetter
}
```

新增 peers ，类型是一致性哈希算法的 Map ，用来根据具体的 key 选择节点。

新增成员 httpGetters ，映射远程节点与对应的 httpGetter 。

然后我们实现 PeerPicker 接口，

```go
// 实例化一致性哈希算法，添加传入的节点
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
        // 为每一个节点创建了一个 http 客户端
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// 包装了一致性哈希的 Get 方法，根据具体的 key 选择节点，返回节点对应的 http 客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)
```

至此，HTTPPool 既具备了提供 HTTP 服务的能力，也具备了根据具体的 key 创建 HTTP 客户端从远程获取缓存值的能力。

最后，我们将功能集成在主流程中，

```go
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
}

// 将实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 使用 PickPeer 方法选择节点，如果不是本地节点，那么调用 getFromPeer 从远程获取。或者，调用 getLocally 
func (g *Group) load(key string) (value ByteView, err error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[GeeCache] Failed to get from peer", err)
		}
	}

	return g.getLocally(key)
}

// getFromPeer 使用实现了 PeerPicker 接口的 httpGetter 访问远程节点，获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}
```

接下来，我们使用 main 函数测试

```go
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// 启动缓存服务器，创建 HTTPPool 添加节点信息，注册到 gee 中，启动 HTTP 服务
func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	peers := geecache.NewHTTPPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 启动一个 API 服务，与用户交互，用户感知
func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		go startAPIServer(apiAddr, gee)
	}
	startCacheServer(addrMap[port], []string(addrs), gee)
}
```

我们下一个小脚本，封装我们的测试命令，

```shell
#!/bin/bash
trap "rm server;kill 0" EXIT

go build -o server
./server -port=8001 &
./server -port=8002 &
./server -port=8003 -api=1 &

sleep 2
echo ">>> start test"
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &

wait
```

### 防止缓存击穿

> **缓存雪崩**：缓存在同一时刻全部失效，造成瞬时DB请求量大、压力骤增，引起雪崩。缓存雪崩通常因为缓存服务器宕机、缓存的 key 设置了相同的过期时间等引起。
>
> **缓存击穿**：一个存在的key，在缓存过期的一刻，同时有大量的请求，这些请求都会击穿到 DB ，造成瞬时DB请求量大、压力骤增。
>
> **缓存穿透**：查询一个不存在的数据，因为不存在则不会写到缓存中，所以每次都会去请求 DB，如果瞬间流量过大，穿透到 DB，导致宕机。

我们要控制一个时间只有对某个元素的一次访问，我们可以看到我们上面对 key=Tom 执行了三次请求，这样容易造成缓存击穿和穿透。

我们可以通过实现一个名为 singleflight 的 package 来解决这个问题，

```go
// 代表正在进行中或已经结束的请求
type call struct {
	wg  sync.WaitGroup // 避免重复写入
	val interface{}
	err error
}

// 管理不同的 key 的请求（key）
type Group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call
}
```

实现 Do 方法，

```go
// Do 的作用是针对相同的 key ，保证 fn 只被调用一个
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock() // 保护 Group 的 m 不被并发读写
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
```

我们修改 geecache.go 中的 Group ，添加成员变量 loader 并更新 NewGroup 。修改 load 函数，将原来的 load 逻辑使用 g.load.Do 包裹起来，确保并发场景下针对相同的 key ，load 只会调用一次。

```go
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	// use singleflight.Group to make sure that
	// each key is only fetched once
	loader *singleflight.Group
}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
    // ...
	g := &Group{
        // ...
		loader:    &singleflight.Group{},
	}
	return g
}

func (g *Group) load(key string) (value ByteView, err error) {
	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}

		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}
```

执行上面的测试函数，如果并发度不够高，可能仍会看到向 8001 请求三次的场景。这种情况下三次请求是串行执行的，并没有触发 `singleflight` 的锁机制工作，可以加大并发数量再测试。

### 总流程分析

我们看 main 函数首先创建了一个 group ，主要就是 getter 类型，用从 map 中取数据模拟从远端数据库中取数据。然后 startCacheServer 就是创建 peers ，我们有三个节点，创建每个节点的时候需要 Set(peers) 将所有节点都添加到一致性哈希中。然后 startAPIServer 就是调用 gee.Get() 处理请求就好了。

那么我们就来看 gee.Get(key) ，首先从 maincache 中找，如果没有，就调用 g.load(key) 。在 g.load(key) 中，调用 g.loader.Do(key) 防止缓存击穿。首先调用 g.PickPeer(key) 确定应该使用节点 peer 来获取数据，那么就调用 g.getFromPeer(peer, key) ；否则，调用 g.GetLocally(key) 。GetLocally 其实就是实现了从数据库中取数据的过程，然后添加到 maincache 中。

说实话，我现在也只是大致明白了流程，很多细节没有搞懂，接口，接口型函数。虽然这个项目不大，对我来说还是很难理解的，毕竟在此之前我只写过单文件代码。
