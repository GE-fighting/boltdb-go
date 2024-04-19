package boltdb_go

// stat 结构体用于统计 B+树的各个组成部分的数量和大小。
type stat struct {
	PageSize          int // 每页的大小
	Depth             int // B+树的深度
	BranchPageCount   int // 分支页的数量
	LeafPageCount     int // 叶子页的数量
	OverflowPageCount int // 超过页面大小限制的数据页数量
	EntryCount        int // B+树中总共的键值对数量
}
