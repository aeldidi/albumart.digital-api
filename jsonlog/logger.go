package jsonlog

import (
	"encoding/json"
	"io"
	"log"
	"strings"
	"time"
)

type Filter struct {
	// A list of log levels to support, in increasing order of severity.
	Levels []string
	// The writer which the logs will be sent to after filtering.
	Writer io.Writer
	// The current log level. All logs with levels below the current one
	// will not be logged.
	Level string
}

func (f *Filter) shouldLog(line []byte) (shouldLog bool, hasLevel bool, level int) {
	hasPrefix := false
	defaultLevel := 0
	prefixLevel := 0
	for i, level := range f.Levels {
		if f.Level == level {
			defaultLevel = i
		}

		if strings.HasPrefix(string(line), level) {
			hasPrefix = true
			prefixLevel = i
			continue
		}
	}

	return prefixLevel >= defaultLevel, hasPrefix, prefixLevel
}

type logTime time.Time

func (lt logTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(lt).Format(time.RFC3339) + `"`), nil
}

type logMessage struct {
	Level     string  `json:"level"`
	Timestamp logTime `json:"timestamp"`
	Message   string  `json:"message"`
}

func (f *Filter) Write(p []byte) (int, error) {
	shouldLog, hasLevel, level := f.shouldLog(p)
	if !shouldLog {
		return len(p), nil
	}

	levelStr := f.Levels[level]
	if hasLevel {
		p = p[len(f.Levels[level])+1:]
	} else {
		levelStr = f.Levels[0]
	}

	if p[len(p)-1] == '\n' {
		p = p[:len(p)-1]
	}

	msg := logMessage{
		Level:     levelStr,
		Timestamp: logTime(time.Now()),
		Message:   string(p),
	}
	err := json.NewEncoder(f.Writer).Encode(&msg)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func SetLevel(level string) {
	if l, ok := log.Default().Writer().(*Filter); ok {
		l.Level = level
	}
}

// Used to output structured logs.
type Logger map[string]interface{}

func Level(level string) Logger {
	return Logger{"level": level}
}

func (l Logger) Field(key string, value interface{}) Logger {
	l[key] = value
	return l
}

func (l Logger) Msg(msg string) {
	l["message"] = msg
	l.Send()
}

func (l Logger) Send() {
	l["timestamp"] = logTime(time.Now())
	if logger, ok := log.Default().Writer().(*Filter); ok {
		level := []byte(l["level"].(string))
		if shouldLog, _, _ := logger.shouldLog(level); !shouldLog {
			return
		}
		_ = json.NewEncoder(logger.Writer).Encode(l)
		return
	}

	_ = json.NewEncoder(log.Default().Writer()).Encode(l)
}
