package main

import (
	"log"
	"path/filepath"
	"runtime"
	"strings"
)

func logger(msg any) {
	pc, file, line, ok := runtime.Caller(1)

	if !ok {
		file = "?"
		line = 0
	}

	fn := runtime.FuncForPC(pc)
	var fnName string
	if fn == nil {
		fnName = "?()"
	} else {
		dotName := filepath.Ext(fn.Name())
		fnName = strings.TrimLeft(dotName, ".") + "()"
	}

	log.Printf("%s:%d %s: %s", filepath.Base(file), line, fnName, msg)
}
