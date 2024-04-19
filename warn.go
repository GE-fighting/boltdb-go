package boltdb_go

import (
	"fmt"
	"os"
)

// warn函数：输出警告信息
// 参数：
//
//	v ...interface{}：变长参数，表示要输出的警告信息内容
//
// 返回值：无
func warn(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...) // 将警告信息输出到标准错误流
}

// warnf函数：格式化输出警告信息
// 参数：
//
//	msg string：格式化字符串，表示要输出的警告信息模板
//	v ...interface{}：变长参数，用于填充格式化字符串中的占位符
//
// 返回值：无
func warnf(msg string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, v...) // 将格式化后的警告信息输出到标准错误流
}
