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

// MaxCommitPages 定义了最大提交页面数为64。
const MaxCommitPages = 64

// maxWriterByteCount 定义了写入器的最大字节计数为0x80000000（2^31-1），这是系统能够处理的最大字节量。
const maxWriterByteCount = 0x80000000

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

// pageState 结构体定义了一个页面的状态
// 其中包含两个字段：
// head: 表示被回收的 freeDB 页面，或者在使用前的 NULL 值。
// last: 表示最后使用的记录的 ID，如果 mf_pghead 为 false，则为 0。
type pageState struct {
	head int /* Reclaimed freeDB pages, or NULL before use*/
	last int /* ID of last used record, or 0 if !mf_pghead*/
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
