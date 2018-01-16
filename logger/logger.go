package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level int

const (
	INVALID Level = iota
	TRACE
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

func (l Level) String() string {
	switch l {
	case INVALID:
		return "INVALID"
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return ""
	}
}

func (l Level) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

type key int

const (
	level key = iota
	values
)

type Pair struct {
	Key   string
	Value interface{}
}

type Formatter interface {
	Log(vals ...interface{})
}

type FormatterFunc func(vals ...interface{})

func (ff FormatterFunc) Log(vals ...interface{}) {
	ff(vals...)
}

var impl Formatter = DefaultLogger{os.Stdout}
var o sync.Once

func Config(i Formatter) {
	o.Do(func() {
		impl = i
	})
}

func WithLevel(c context.Context, l Level) context.Context {
	return context.WithValue(c, level, l)
}

func WithValues(c context.Context, vals ...Pair) context.Context {
	return context.WithValue(c, values, vals)
}

func Log(c context.Context, l Level, message string, vals ...Pair) {
	curLevelVal := c.Value(level)
	if curLevel, ok := curLevelVal.(Level); !ok || curLevel == INVALID || curLevel > l {
		return
	}
	outVals := []interface{}{"time", time.Now().UTC(), "level", l, "message", message}

	if curVals, ok := c.Value(values).([]Pair); ok {
		for _, v := range curVals {
			outVals = append(outVals, v.Key)
			outVals = append(outVals, v.Value)
		}
	}

	for _, v := range vals {
		outVals = append(outVals, v.Key)
		outVals = append(outVals, v.Value)
	}
	impl.Log(outVals...)
}

type DefaultLogger struct {
	io.Writer
}

func (dl DefaultLogger) Log(vals ...interface{}) {
	if dl.Writer == nil {
		return
	}
	var out bytes.Buffer
	out.WriteString("{")
	for i := 0; i < len(vals); i += 2 {
		out.WriteString(fmt.Sprintf(`%s:%s`, dl.toJSONString(vals[i]), dl.toJSONString(vals[i+1])))
		if i < len(vals)-2 {
			out.WriteRune(',')
		}
	}
	out.WriteString("}\n")
	dl.Write(out.Bytes())
}

func (dl DefaultLogger) toJSONString(i interface{}) string {
	out, err := json.Marshal(i)
	if err != nil {
		out, _ = json.Marshal(err.Error())
	}
	return string(out)
}
