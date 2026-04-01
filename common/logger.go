package common

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	warnLogger  *log.Logger
)

// InitLogger 初始化日志系统
func InitLogger(serviceName string) {
	// 确保日志目录存在
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("创建日志目录失败: %v", err)
	}

	// 设置日志文件名
	logFile := filepath.Join(logDir, serviceName+".log")

	// 打开日志文件
	logFileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("打开日志文件失败: %v", err)
		// 如果文件打开失败，只输出到终端
		setupLoggers(os.Stdout, os.Stderr)
		return
	}

	// 同时输出到终端和文件
	multiWriter := io.MultiWriter(os.Stdout, logFileHandle)
	multiErrorWriter := io.MultiWriter(os.Stderr, logFileHandle)
	
	setupLoggers(multiWriter, multiErrorWriter)
	
	// 输出启动信息
	fmt.Printf("=== %s 服务启动 ===\n", serviceName)
	fmt.Printf("日志文件: %s\n", logFile)
}

func setupLoggers(out, errOut io.Writer) {
	infoLogger = log.New(out, "", log.LstdFlags|log.Lshortfile)
	errorLogger = log.New(errOut, "", log.LstdFlags|log.Lshortfile)
	debugLogger = log.New(out, "", log.LstdFlags|log.Lshortfile)
	warnLogger = log.New(errOut, "", log.LstdFlags|log.Lshortfile)
}

// LogInfo 输出信息日志
func LogInfo(format string, v ...interface{}) {
	if infoLogger != nil {
		infoLogger.Printf("[INFO] "+format, v...)
	}
}

// LogError 输出错误日志
func LogError(format string, v ...interface{}) {
	if errorLogger != nil {
		errorLogger.Printf("[ERROR] "+format, v...)
	}
}

// LogDebug 输出调试日志
func LogDebug(format string, v ...interface{}) {
	if debugLogger != nil {
		debugLogger.Printf("[DEBUG] "+format, v...)
	}
}

// LogWarn 输出警告日志
func LogWarn(format string, v ...interface{}) {
	if warnLogger != nil {
		warnLogger.Printf("[WARN] "+format, v...)
	}
}