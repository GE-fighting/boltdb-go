package boltdb_go

// MinPageKeys 是页面中键的最小数量。
const MinPageKeys = 2

// FillThreshold 是页面填充的阈值。
const FillThreshold = 250

// 页面类型的常量定义。
const (
	BranchPage   = 0x01   // 分支页面
	LeafPage     = 0x02   // 叶子页面
	OverflowPage = 0x04   // 溢出页面
	MetaPage     = 0x08   // 元数据页面
	DirtyPage    = 0x10   // 脏页面
	SubPage      = 0x40   // 子页面
	KeepPage     = 0x8000 // 保留页面
)

// page 结构体表示数据库中的一个页面。
type page struct {
	header struct {
		id                int   // 页面ID
		next              *page // 下一个页面指针
		lower             int   // 页面中数据的起始位置
		upper             int   // 页面中数据的结束位置
		overflowPageCount int   // 溢出页面的数量
	}
	metadata []byte // 页面的元数据
}

// nodeCount 返回页面中的节点数量。
// 当前实现返回0，预留方法，可能在后续版本中实现。
func (p *page) nodeCount() int {
	return 0
}

// remainingSize 返回页面中剩余可用的空间大小。
func (p *page) remainingSize() int {
	return p.header.upper - p.header.lower
}
