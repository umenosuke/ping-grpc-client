package labelinglog

import (
	"io"
)

// LogLevel is used to specify output and configuration targets.
type LogLevel uint16

// Flg** is a log level bit flag.
const (
	FlgFatal  LogLevel = 1 << 0
	FlgError  LogLevel = 1 << 1
	FlgWarn   LogLevel = 1 << 2
	FlgNotice LogLevel = 1 << 3
	FlgInfo   LogLevel = 1 << 4
	FlgDebug  LogLevel = 1 << 5
)

// Flgset** is a preset log level bit flag set.
const (
	FlgsetAll    LogLevel = 0xffff
	FlgsetCommon LogLevel = FlgFatal | FlgError | FlgWarn | FlgNotice
)

// LabelingLogger is the main
type LabelingLogger struct {
	loggers         []*tLogger
	enableFileame   bool
	enableTimestamp bool
}

// New returns an initialized LabelingLogger
func New(prefix string, writer io.Writer) *LabelingLogger {
	loggers := make([]*tLogger, 0)
	loggers = append(loggers, &tLogger{
		isEnable: true,
		writer:   writer,
		prefix:   "[" + prefix + "][FATAL] ",
		flg:      FlgFatal,
	})
	loggers = append(loggers, &tLogger{
		isEnable: true,
		writer:   writer,
		prefix:   "[" + prefix + "][ERROR] ",
		flg:      FlgError,
	})
	loggers = append(loggers, &tLogger{
		isEnable: true,
		writer:   writer,
		prefix:   "[" + prefix + "][WARN]  ",
		flg:      FlgWarn,
	})
	loggers = append(loggers, &tLogger{
		isEnable: true,
		writer:   writer,
		prefix:   "[" + prefix + "][NOTICE]",
		flg:      FlgNotice,
	})
	loggers = append(loggers, &tLogger{
		isEnable: true,
		writer:   writer,
		prefix:   "[" + prefix + "][INFO]  ",
		flg:      FlgInfo,
	})
	loggers = append(loggers, &tLogger{
		isEnable: true,
		writer:   writer,
		prefix:   "[" + prefix + "][DEBUG] ",
		flg:      FlgDebug,
	})

	return &LabelingLogger{
		loggers:         loggers,
		enableFileame:   true,
		enableTimestamp: true,
	}
}
