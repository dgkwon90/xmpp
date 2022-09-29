package main

import (
	"log"
	"runtime"
	"strconv"
	"strings"
)

/* Inject logging into xmpp library */

const (
	LOGGER_OFF = iota
	LOGGER_ERROR
	LOGGER_WARN
	LOGGER_INFO
	LOGGER_DEBUG
)

type Logger struct {
	level *int
}

func (l Logger) Error(format string, args ...interface{}) {
	if *l.level >= LOGGER_ERROR {
		_, fullFilePath, line, _ := runtime.Caller(1)
		//_ := runtime.FuncForPC(pc).Name()
		paths := strings.Split(fullFilePath, "/")
		file := paths[len(paths)-1]
		//_, err = fmt.Printf("ERROR: %s\n", msg)
		log.Printf("[ERROR] "+file+":"+strconv.Itoa(line)+": "+format, args...)
	}
}

func (l Logger) Waring(format string, args ...interface{}) {
	if *l.level >= LOGGER_WARN {
		_, fullFilePath, line, _ := runtime.Caller(1)
		//_ := runtime.FuncForPC(pc).Name()
		paths := strings.Split(fullFilePath, "/")
		file := paths[len(paths)-1]
		//_, err = fmt.Printf("ERROR: %s\n", msg)
		log.Printf("[WARN] "+file+":"+strconv.Itoa(line)+": "+format, args...)
	}
}

func (l Logger) Info(format string, args ...interface{}) {
	if *l.level >= LOGGER_INFO {
		_, fullFilePath, line, _ := runtime.Caller(1)
		//_ := runtime.FuncForPC(pc).Name()
		paths := strings.Split(fullFilePath, "/")
		file := paths[len(paths)-1]
		//_, err = fmt.Printf("ERROR: %s\n", msg)
		log.Printf("[INFO] "+file+":"+strconv.Itoa(line)+": "+format, args...)
	}
}

func (l Logger) Debug(format string, args ...interface{}) {
	if *l.level >= LOGGER_DEBUG {
		_, fullFilePath, line, _ := runtime.Caller(1)
		//_ := runtime.FuncForPC(pc).Name()
		paths := strings.Split(fullFilePath, "/")
		file := paths[len(paths)-1]
		//_, err = fmt.Printf("ERROR: %s\n", msg)
		log.Printf("[DEBUG] "+file+":"+strconv.Itoa(line)+": "+format, args...)
	}
}
