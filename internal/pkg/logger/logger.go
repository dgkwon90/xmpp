package logger

import (
	"strings"
)

/* Inject logging into xmpp library */
const (
	DisableLevel = iota
	ErrorLevel
	WarningLevel
	InfoLevel
	DebugLevel
)

type Level int

type Logger struct {
	Level Level
}

func fileName(srcPath string) string {
	paths := strings.Split(srcPath, "/")
	file := paths[len(paths)-1]
	return file
}

func (l Logger) Error(format string, args ...interface{}) {
	// if l.Level >= ErrorLevel {
	// 	_, fullFilePath, line, _ := runtime.Caller(1)
	// 	file := fileName(fullFilePath)
	// 	log.Printf("[ERROR] "+file+":"+strconv.Itoa(line)+": "+format, args...)
	// }
}

func (l Logger) Waring(format string, args ...interface{}) {
	// if l.Level >= WarningLevel {
	// 	_, fullFilePath, line, _ := runtime.Caller(1)
	// 	file := fileName(fullFilePath)
	// 	log.Printf("[WARN] "+file+":"+strconv.Itoa(line)+": "+format, args...)
	// }
}

func (l Logger) Info(format string, args ...interface{}) {
	// if l.Level >= InfoLevel {
	// 	_, fullFilePath, line, _ := runtime.Caller(1)
	// 	file := fileName(fullFilePath)
	// 	log.Printf("[INFO] "+file+":"+strconv.Itoa(line)+": "+format, args...)
	// }
}

func (l Logger) Debug(format string, args ...interface{}) {
	// if l.Level >= DebugLevel {
	// 	_, fullFilePath, line, _ := runtime.Caller(1)
	// 	file := fileName(fullFilePath)
	// 	log.Printf("[DEBUG] "+file+":"+strconv.Itoa(line)+": "+format, args...)
	// }
}
