package boltdb_go

// Cursor 接口定义了操作数据库游标的接口。
type Cursor interface {
	// First 将游标定位到当前Bucket中的第一个键值对。
	First() error
	// FirstDup 将游标定位到当前Bucket中的第一个重复的键值对。
	FirstDup() error
	// Get 返回当前游标指向的键和值。
	Get() ([]byte, []byte, error)
	// GetRange 返回当前范围内的键和值。
	GetRange() ([]byte, []byte, error)
	// Current 返回当前游标指向的键和值。
	Current() ([]byte, []byte, error)
	// Last 将游标定位到当前Bucket中的最后一个键值对。
	Last()
	// LastDup 将游标定位到当前Bucket中的最后一个重复的键值对。
	LastDup()
	// Next 将游标移动到下一个键值对。
	Next() ([]byte, []byte, error)
	// NextDup 将游标移动到下一个重复的键值对。
	NextDup() ([]byte, []byte, error)
	// NextNoDup 将游标移动到下一个不重复的键值对。
	NextNoDup() ([]byte, []byte, error)
	// Pre 将游标移动到前一个键值对。
	Pre() ([]byte, []byte, error)
	// PreDup 将游标移动到前一个重复的键值对。
	PreDup() ([]byte, []byte, error)
	// PreNoDup 将游标移动到前一个不重复的键值对。
	PreNoDup() ([]byte, []byte, error)
	// Set 将游标定位到指定的键值对。
	Set() ([]byte, []byte, error)
	// SetRange 将游标定位到指定范围的键值对。
	SetRange() ([]byte, []byte, error)
}

// cursor 结构体实现了Cursor接口，具体实现了数据库游标的操作逻辑。
type cursor struct {
	flags       int          // 标志位，用于控制游标行为
	next        *cursor      // 下一个游标，用于实现嵌套游标操作
	backup      *cursor      // 备份游标，用于实现回滚等操作
	xcursor     *xcursor     // 用于底层存储访问的游标
	transaction *transaction // 关联的事务对象
	bucketID    int          // 当前操作的Bucket ID
	bucket      *bucket      // 当前操作的Bucket
	bucketx     *bucketx     // 内部使用的Bucket扩展信息
	bucketFlag  int          // Bucket标志位，用于标识Bucket状态
	snum        int          // 页栈的数量，用于内部页跳转逻辑
	top         int          // 页栈的顶部索引，标识当前所在的页层级
	page        []*page      // 页栈，用于记录游标当前位置的页信息
	ki          []int        // 页索引栈，用于记录游标在各页中的索引位置
}
