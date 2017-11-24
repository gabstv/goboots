package goboots

import (
	"bytes"
	"fmt"
	"runtime"
)

func deepError(id int, str string, v ...interface{}) error {
	nc := 15
	var b bytes.Buffer
	for i := 0; i < nc; i++ {
		_, file, line, ok := runtime.Caller(i + 2)
		if ok {
			b.WriteString(fmt.Sprintf("on `%s` @ %v\n", file, line))
		} else {
			break
		}
	}
	b.WriteString(fmt.Sprintf(str, v...))
	e := AppError{}
	e.Message = b.String()
	e.Id = id
	return &e
}

func deepErrorStr(str string, v ...interface{}) error {
	return deepError(9999, str, v...)
}

func deepErr(err error) error {
	if err == nil {
		return nil
	}
	return deepError(9999, err.Error())
}
