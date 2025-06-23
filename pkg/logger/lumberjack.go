// Package logger lumberjack日志轮转支持
// 由于lumberjack不在go.mod中，这里添加一个简化的实现
package logger

import (
	"io"
	"os"
)

// SimpleLumberjack 简化的日志轮转实现
// 在正式版本中应该使用 gopkg.in/natefinch/lumberjack.v2
type SimpleLumberjack struct {
	Filename   string
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool
}

// Write 实现io.Writer接口
func (l *SimpleLumberjack) Write(p []byte) (n int, err error) {
	// 简化实现：直接写入文件
	// 在实际使用中应该替换为真实的lumberjack
	file, err := os.OpenFile(l.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	
	return file.Write(p)
}

// 确保实现了io.Writer接口
var _ io.Writer = (*SimpleLumberjack)(nil) 