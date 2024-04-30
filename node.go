package boltdb_go

// nodeFlags 定义节点的类型标识。
const (
	// bigNode 标识节点为大节点。
	bigNode = 0x01
	// subNode 标识节点为子节点。
	subNode = 0x02
	// dupNode 标识节点为重复节点。
	dupNode = 0x04
)

// node 结构体表示Boltdb数据库中的一个节点。
type node struct {
	lo      int    // 节点的起始位置
	hi      int    // 节点的结束位置
	flags   int    // 节点类型标识（如bigNode、subNode、dupNode）
	keySize int    // 节点中键的大小
	data    []byte // 节点存储的数据
}

// setFlags 设置节点的类型标识。
func (n *node) setFlags(f int) {
	n.flags = f
}

// size 返回节点占用的总字节数。
// 当前实现返回0，预留方法，可能在后续版本中实现。
func (n *node) size() int {
	return 0
}

// shrink 函数用于在删除子页面上的一个节点后，紧凑主页面。
// 参数：
//
//	index int - 要删除的节点在子页面中的索引。
//
// 该函数没有返回值。
func (n *node) shrink(index int) {

}
