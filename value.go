package log

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

var (
	defaultDepth = 3
	// DefaultCaller is a Valuer that returns the file and line.
	DefaultCaller = Caller(defaultDepth)

	// DefaultTimestamp is a Valuer that returns the current wallclock time.
	DefaultTimestamp = Timestamp("2006-01-02 15:04:05.999")
)

// Valuer is returns a log value.
type Valuer func(ctx context.Context) interface{}

// Value return the function value.
func Value(ctx context.Context, v interface{}) interface{} {
	if v, ok := v.(Valuer); ok {
		return v(ctx)
	}
	return v
}

// Caller returns returns a Valuer that returns a pkg/file:line description of the caller.
func Caller(skip int) Valuer {
	return func(context.Context) interface{} {
		file := ""
		line := 0
		var pc uintptr
		// 遍历调用栈的最大索引为第11层.
		for i := 0; i < 11; i++ {
			file, line, pc = getCaller(skip + i)
			// 过滤掉所有logrus包，即可得到生成代码信息
			if !strings.HasPrefix(file, "logrus") {
				break
			}
		}
		fullFnName := runtime.FuncForPC(pc)
		fnName := ""
		if fullFnName != nil {
			fnNameStr := fullFnName.Name()
			// 取得函数名
			parts := strings.Split(fnNameStr, ".")
			fnName = parts[len(parts)-1]
		}
		return fmt.Sprintf("%s:%s():%d", file, fnName, line)
	}
}

func getCaller(skip int) (string, int, uintptr) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", 0, pc
	}
	n := 0

	// 获取包名
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			n++
			if n >= 2 {
				file = file[i+1:]
				break
			}
		}
	}
	return file, line, pc
}

// Timestamp returns a timestamp Valuer with a custom time format.
func Timestamp(layout string) Valuer {
	return func(context.Context) interface{} {
		return time.Now().Format(layout)
	}
}

func bindValues(ctx context.Context, keyvals []interface{}) {
	for i := 1; i < len(keyvals); i += 2 {
		if v, ok := keyvals[i].(Valuer); ok {
			keyvals[i] = v(ctx)
		}
	}
}

func containsValuer(keyvals []interface{}) bool {
	for i := 1; i < len(keyvals); i += 2 {
		if _, ok := keyvals[i].(Valuer); ok {
			return true
		}
	}
	return false
}
