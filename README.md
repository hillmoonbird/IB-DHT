  ## 支持特性及实现原理

### LRU 缓存

<p align="center">
<img width="500" align="center" src="images/1.jpg" />
</p>

这张图很好地表示了 LRU 算法最核心的 2 个数据结构：

- 绿色的是字典(map)，存储键和值的映射关系。这样根据某个键(key)查找对应的值(value)的复杂是O(1)，在字典中插入一条记录的复杂度也是O(1)；
- 红色的是双向链表(double linked list)实现的队列。将所有的值放到双向链表中，这样，当访问到某个值时，将其移动到队尾的复杂度是O(1)，在队尾新增一条记录以及删除一条记录的复杂度均为O(1)。

### 负载均衡

负载均衡主要是通过一致性哈希算法来实现的。

#### 步骤

一致性哈希算法将 key 映射到 $2^{32}$ 的空间中，将这个数字首尾相连，形成一个环。

- 计算节点/机器（通常使用节点的名称、编号和 IP 地址）的哈希值，放置在环上；
- 计算 key 的哈希值，放置在环上，顺时针寻找到的第一个节点，就是应选取的节点/机器。

<p align="center">
<img width="500" align="center" src="images/2.jpg" />
</p>

一致性哈希算法，在新增/删除节点时，只需要重新定位该节点附近的一小部分数据，而不需要重新定位所有的节点，这就避免了`缓存雪崩`。

#### 数据倾斜问题

如果服务器的节点过少，容易引起 key 的倾斜，造成缓存节点间负载不均。

为了解决这个问题，引入了虚拟节点的概念，一个真实节点对应多个虚拟节点。

- 第一步，计算虚拟节点的 Hash 值，放置在环上；
- 第二步，计算 key 的 Hash 值，在环上顺时针寻找到应选取的虚拟节点，例如是 peer2-1，那么就对应真实节点 peer2。

虚拟节点扩充了节点的数量，解决了节点较少的情况下数据容易倾斜的问题。而且代价非常小，只需要增加一个字典（map）维护真实节点与虚拟节点的映射关系即可。

### 防止缓存击穿

`缓存雪崩`：缓存在同一时刻全部失效，造成瞬时 DB 请求量大、压力骤增，引起雪崩。缓存雪崩通常因为缓存服务器宕机、缓存的 key 设置了相同的过期时间等引起。

`缓存击穿`：一个存在的 key，在缓存过期的一刻，同时有大量的请求，这些请求都会击穿到 DB ，造成瞬时 DB 请求量大、压力骤增。

`缓存穿透`：查询一个不存在的数据，因为不存在则不会写到缓存中，所以每次都会去请求 DB，如果瞬间流量过大，穿透到 DB，导致宕机。

要避免缓存击穿，必须保证短时间内无论向 API 发起多少次针对同一 key 的请求，向数据库服务端的请求都只有一次。

这主要是通过 Go 锁来实现的，包括 `sync.Mutex` 和 `sync.WaitGroup`。



## 提供的 API

```go
// Cache 是一个 LRU 缓存，它是非并发安全的。
type Cache struct {
	maxBytes    int64                          // 允许使用的最大内存         
	usedBytes   int64                          // 当前已使用的内存
	dList       *list.List                     // 双向链表，存储值
	cMap        map[string]*list.Element       // 字典，存储键和值的映射关系
	OnEvicted   func(key string, value Value)  // 某条记录被移除时的回调函数，可以为 nil
}
// Cache 实例化函数
func New(maxBytes int64, onEvicted func(string, Value)) *Cache
// Cache 查找函数
func (c *Cache) Get(key string) (val Value, ok bool)
// Cache 删除函数（缓存淘汰）
func (c *Cache) RemoveLRU() 
// Cache 新增/修改函数
func (c *Cache) Add(key string, val Value)
// 获取已添加数据的条数
func (c *Cache) Len() int
    

// ByteView 是用来表示缓存值的抽象的只读数据结构
type ByteView struct {
	// 选择 byte 类型是为了能够支持任意数据类型的存储
	b []byte
}
// 实现 lru.Cache 中的 Value 接口
func (v ByteView) Len() int
// 返回只读对象的拷贝，防止缓存值被外部程序修改
func (v ByteView) ByteSlice() []byte
// 以拷贝的形式将 ByteView 中的 []byte 转换为 string
func (v ByteView) String() string


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


// HTTP 调度器
type PeerPicker interface {
	// 根据传入的 key 选择相应节点 PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}
// HTTP 客户端
type PeerGetter interface {
	// 从对应的 group 查找缓存值
	Get(group string, key string) ([]byte, error)
}


// 一致性哈希算法的主要数据结构
type Map struct {
	hash     Hash            // Hash 函数
	replicas int             // 虚拟节点倍数
	keys     []int           // 哈希环
	hashMap  map[int]string  // 虚拟节点与真实节点的映射表
}
// Map 实例化函数，允许自定义虚拟节点倍数和 Hash 函数
func New(rpl int, fx Hash) *Map
// 节点添加函数（真实/虚拟节点）
func (m *Map) Add(keys ...string)
// 获取键为 key 的结点对应的真实节点
func (m *Map) Get(key string) string


// ZhongCache 最核心的数据结构，负责与用户的交互，并且控制缓存值存储和获取的流程
type Group struct {
	name       string
	getter     Getter
	mainCache  cache
	peers      PeerPicker
	loader     *singleflight.Group
}
// 实例化 Group，并且将 Group 存储在全局变量数组 groups 中
func NewGroup(name string, cacheBytes int64, getter Getter) *Group
// 获取特定名称的 Group
func GetGroup(name string) *Group
// 获取特定 Group 中键为 key 的缓存
func (g *Group) Get(key string) (ByteView, error)
// 将实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker)


// 承载节点间 HTTP 通信的核心数据结构
type HTTPPool struct {
	self        string                 // 自身地址，包括主机名/IP和端口
	basePath    string                 // 节点间通讯地址的前缀
	mut         sync.Mutex             // 互斥锁
	peers       *consistenthash.Map    // 一致性哈希的 Map，用来根据具体的 key 选择节点
	httpGetters map[string]*httpGetter // 远程节点与对应 httpGetter 之间的映射
}
// HTTPPool 的实例化函数
func NewHTTPPool(self string) *HTTPPool
// HTTPPool 日志打印
func (p *HTTPPool) Log(format string, v ...interface{})
// HTTP 服务主体
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request)
// 实例化一致性哈希算法，添加传入的节点
func (p *HTTPPool) Set(peers ...string)
// 包装了一致性哈希算法的 Get() 方法，根据具体的 key，选择节点，返回节点对应的 HTTP 客户端
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool)
// 使用 http.Get() 方式获取返回值，并转换为 []bytes 类型
func (h *httpGetter) Get(group string, key string) ([]byte, error)