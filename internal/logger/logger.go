// Package logger 日志包
// 提供统一格式的日志记录功能
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// 日志级别常量
const (
	DEBUG = iota // 调试级别
	INFO         // 信息级别
	WARN         // 警告级别
	ERROR        // 错误级别
	FATAL        // 致命错误级别
)

// Logger 日志记录器
// 支持分级日志、线程安全、可配置输出
type Logger struct {
	logger *log.Logger // 标准库日志实例
	level  int         // 日志级别
	mu     sync.Mutex  // 互斥锁，保证线程安全
}

var (
	defaultLogger *Logger // 默认日志实例
	once          sync.Once
)

// Init 初始化日志系统
// 参数: level 日志级别(debug/info/warn/error/fatal), output 输出位置
func Init(level string, output io.Writer) {
	once.Do(func() {
		// 解析日志级别
		lvl := INFO
		switch level {
		case "debug":
			lvl = DEBUG
		case "warn":
			lvl = WARN
		case "error":
			lvl = ERROR
		case "fatal":
			lvl = FATAL
		}
		
		// 默认输出到标准输出
		if output == nil {
			output = os.Stdout
		}
		
		// 创建日志实例
		defaultLogger = &Logger{
			logger: log.New(output, "", 0),
			level:  lvl,
		}
	})
}

// Get 获取默认日志实例
// 如果未初始化，则使用默认配置创建
func Get() *Logger {
	if defaultLogger == nil {
		Init("info", nil)
	}
	return defaultLogger
}

// Debug 记录调试级别日志
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, "DEBUG", format, v...)
}

// Info 记录信息级别日志
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(INFO, "INFO", format, v...)
}

// Warn 记录警告级别日志
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(WARN, "WARN", format, v...)
}

// Error 记录错误级别日志
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(ERROR, "ERROR", format, v...)
}

// Fatal 记录致命错误日志并退出程序
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.log(FATAL, "FATAL", format, v...)
	os.Exit(1)
}

// log 内部日志方法
// 参数: level 日志级别, levelStr 级别字符串, format 格式, v 参数
func (l *Logger) log(level int, levelStr, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// 低于设定级别的日志不记录
	if level < l.level {
		return
	}
	
	// 格式化消息
	msg := fmt.Sprintf(format, v...)
	// 添加时间戳
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	// 输出日志
	l.logger.Printf("[%s] [%s] %s", timestamp, levelStr, msg)
}

// ============ 全局便捷函数 ============

// Debug 调试日志
func Debug(format string, v ...interface{}) {
	Get().Debug(format, v...)
}

// Info 信息日志
func Info(format string, v ...interface{}) {
	Get().Info(format, v...)
}

// Warn 警告日志
func Warn(format string, v ...interface{}) {
	Get().Warn(format, v...)
}

// Error 错误日志
func Error(format string, v ...interface{}) {
	Get().Error(format, v...)
}

// Fatal 致命错误日志
func Fatal(format string, v ...interface{}) {
	Get().Fatal(format, v...)
}
