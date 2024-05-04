package boltdb_go

// bucket 结构体实现了 Bucket 接口，具体存储桶的实现细节。
type Bucket struct {
	pad       uint32 // 用于内存对齐的填充项
	flags     uint16 // 标志位，用于表示存储桶的特性或状态
	depth     uint16 // 存储桶的深度，表示数据在页结构中的层次
	branches  pgno   // 分支页的数量，用于表示存储桶的分层结构
	leafs     pgno   // 叶子页的数量，存储实际数据的页面数
	overflows pgno   // 超出页的数量，用于处理数据量过大无法容纳在一个页面的情况
	entries   uint64 // 存储桶中的条目数量，即存储的数据项数量
	root      pgno   // 根节点的ID，指向存储桶的顶层页面
}
