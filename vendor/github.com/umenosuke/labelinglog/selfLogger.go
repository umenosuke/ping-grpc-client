package labelinglog

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Use for logger's own log output.
type tSelfLogger struct {
	sync.Mutex
	writer io.Writer
	prefix string
	flg    LogLevel
}

func (thisLogger *tSelfLogger) log(timestamp string, fileName string, msg string) {
	thisLogger.Lock()
	defer thisLogger.Unlock()
	fmt.Fprintln(thisLogger.writer, timestamp+thisLogger.prefix+" "+fileName+": "+msg)
}

var selfLogger = &tSelfLogger{
	writer: os.Stderr,
	prefix: "[logger][FATAL]",
	flg:    FlgsetAll,
}
