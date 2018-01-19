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
	OFF Level = iota
	TRACE
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

func (l Level) String() string {
	switch l {
	case OFF:
		return "OFF"
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

type Logger interface {
	Log(vals ...interface{}) error
}

type LoggerFunc func(vals ...interface{}) error

func (lf LoggerFunc) Log(vals ...interface{}) error {
	return lf(vals...)
}

var impl Logger = DefaultLogger{Writer: os.Stdout}
var rw sync.RWMutex

func Config(i Logger) {
	rw.Lock()
	defer rw.Unlock()
	impl = i
}

func WithLevel(c context.Context, l Level) context.Context {
	return context.WithValue(c, level, l)
}

func WithValues(c context.Context, vals ...Pair) context.Context {
	//if there are any existing pairs, copy them into this vals as well
	var pairs []Pair
	if curVals, ok := c.Value(values).([]Pair); ok {
		pairs = append(pairs, curVals...)
	}
	pairs = append(pairs, vals...)

	return context.WithValue(c, values, pairs)
}

func Log(c context.Context, l Level, message string, vals ...Pair) {
	curLevelVal := c.Value(level)
	if curLevel, ok := curLevelVal.(Level); !ok || curLevel == OFF || curLevel > l {
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
	rw.RLock()
	defer rw.RUnlock()
	impl.Log(outVals...)
}

type Formatter interface {
	Format(vals ...interface{}) string
}

type FormatterFunc func(vals ...interface{}) string

func (ff FormatterFunc) Format(vals ...interface{}) string {
	return ff(vals...)
}

type DefaultLogger struct {
	io.Writer
	Formatter Formatter
}

func (dl DefaultLogger) Log(vals ...interface{}) error {
	if dl.Writer == nil {
		return nil
	}
	var out string
	if dl.Formatter == nil {
		out = jsonOutput(vals...)
	} else {
		out = dl.Formatter.Format(vals...)
	}
	_, err := dl.Writer.Write([]byte(out))
	return err
}

func jsonOutput(vals ...interface{}) string {
	var out bytes.Buffer
	out.WriteString("{")
	for i := 0; i < len(vals); i += 2 {
		out.WriteString(fmt.Sprintf(`%s:%s`, toJSONString(vals[i]), toJSONString(vals[i+1])))
		if i < len(vals)-2 {
			out.WriteRune(',')
		}
	}
	out.WriteString("}\n")
	return out.String()
}

func toJSONString(i interface{}) string {
	out, err := json.Marshal(i)
	if err != nil {
		out, _ = json.Marshal(err.Error())
	}
	return string(out)
}
