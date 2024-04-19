package boltdb_go

// Transaction 接口定义了Boltdb数据库事务的基本操作。
type Transaction interface {
	// （待补充具体的Transaction接口方法声明）
}

// transaction 结构体实现了Transaction接口，表示Boltdb数据库中的一个事务。
type transaction struct {
	// id 为当前事务的唯一标识符。
	id int
	// flags 存储与事务相关的一些标志位信息。
	flags int
	// db 指向当前事务所属的Boltdb数据库实例。
	db *db
	// parent 指向当前事务的父级事务（如果存在）。
	parent *transaction
	// child 指向当前事务的子级事务（如果存在）。
	child *transaction
	// nextPageNumber 记录下一个待分配的页面号。
	nextPageNumber int
	// freePages 存储当前事务中已释放的页面列表。
	freePages []int
	// spillPages 存储当前事务中溢出的页面列表。
	spillPages []int
	// dirtyList 存储当前事务中被修改但尚未同步到磁盘的页面列表。
	dirtyList []int
	// reader 提供对数据库底层数据的读取访问。
	reader *reader
	// buckets 存储当前事务涉及的所有桶的引用。
	buckets []*bucket
	// bucketFlags 存储与各个桶关联的标志位信息。
	bucketFlags []int
	// TODO: 待实现bucketxs字段，用于存储与事务关联的扩展桶列表。
	//bucketxs []*bucketx
	// cursor 存储当前事务创建的所有游标对象。
	cursor []*cursor
	// Implicit from slices? TODO: MDB_dbi mt_numdbs
	// mt_dirty_room 存储当前事务的脏数据空间大小。
	mt_dirty_room int
}
