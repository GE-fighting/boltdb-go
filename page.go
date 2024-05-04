package boltdb_go

import "unsafe"

const maxPageSize = 0x8000
const minKeyCount = 2

var _page page

const pageHeaderSize = int(unsafe.Offsetof(_page.ptr))

// MinPageKeys 是页面中键的最小数量。
const minPageKeys = 2

// FillThreshold 是页面填充的阈值。
const fillThreshold = 250

// 页面类型的常量定义。
const (
	p_branch   = 0x01   // 分支页面
	p_leaf     = 0x02   // 叶子页面
	p_overflow = 0x04   // 溢出页面
	p_meta     = 0x08   // 元数据页面
	p_dirty    = 0x10   // 脏页面
	p_sub      = 0x40   // 子页面
	p_keep     = 0x8000 // 保留页面
	p_invalid  = ^pgno(0)
)

// MaxCommitPages 定义了最大提交页面数为64。
const MaxCommitPages = 64

// maxWriterByteCount 定义了写入器的最大字节计数为0x80000000（2^31-1），这是系统能够处理的最大字节量。
const maxWriterByteCount = 0x80000000

type pgno uint64
type txnid uint64
type indx uint16

// page 结构体表示数据库中的一个页面。
type page struct {
	id       pgno // 页面ID
	flags    int  //
	lower    indx // 页面中数据的起始位置
	upper    indx // 页面中数据的结束位置
	overflow int  // 溢出页面的数量
	ptr      int
}

// pageState 结构体定义了一个页面的状态
// 其中包含两个字段：
// head: 表示被回收的 freeDB 页面，或者在使用前的 NULL 值。
// last: 表示最后使用的记录的 ID，如果 mf_pghead 为 false，则为 0。
type pageState struct {
	head int /* Reclaimed freeDB pages, or NULL before use*/
	last int /* ID of last used record, or 0 if !mf_pghead*/
}

// meta函数尝试从page结构体中提取meta信息。
// 如果page被标记为非元数据页(即p_meta标志未设置)，函数将视为无效元数据页并返回错误。
// 参数:
// - p: 指向待检查page结构体的指针。
// 返回值:
// - *meta: 当page有效且包含meta信息时，返回指向meta结构体的指针。
// - error: 如果page被标记为非元数据页或验证meta信息时出错，则返回相应的错误；否则返回nil。
func (p *page) meta() (*meta, error) {
	// 检查page是否明确标记为非元数据页
	if (p.flags & p_meta) == 0 { // 注意这里的条件修正，当p_meta标志未设置时执行
		return nil, InValidMetaPageError
	}
	// 将page数据部分转换为meta结构体指针，假设ptr字段存放着meta数据
	m := (*meta)(unsafe.Pointer(&p.ptr))
	// 验证转换得到的meta数据是否有效
	if err := m.validate(); err != nil {
		return nil, err
	}
	// 返回有效的meta信息
	return m, nil
}

// initMeta 初始化一个页面为元数据页面
// 参数:
//
//	pageSize - 页面大小，用于设置元数据中的pageSize字段。
//
// 该方法不返回任何值。
func (p *page) initMeta(pageSize int) {
	// 初始化页面标志为p_meta
	p.flags = p_meta

	// 通过unsafe.Pointer转换，将p.ptr转换为meta类型的指针，并对其进行操作
	m := (*meta)(unsafe.Pointer(&p.ptr))

	// 设置magic数、版本号和pageSize
	m.magic = magic
	m.version = Version
	m.free.pad = uint32(pageSize)

	// 初始化页面号和两个根节点为无效值
	m.pgno = 1
	m.free.root = p_invalid
	m.main.root = p_invalid
}

// nodeCount 返回页面中的节点数量。
// 当前实现返回0，预留方法，可能在后续版本中实现。
func (p *page) nodeCount() int {
	return 0
}

// remainingSize 返回页面中剩余可用的空间大小。
func (p *page) remainingSize() int {
	return int(p.upper - p.lower)
}
