package labelinglog

import (
	"bufio"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// Log outputs messages at the specified log level.
func (thisLabelingLogger *LabelingLogger) Log(targetLevelFlgs LogLevel, msg string) {
	if !thisLabelingLogger.isActive(targetLevelFlgs) {
		return
	}

	var timestamp string
	if thisLabelingLogger.enableTimestamp {
		timestamp = time.Now().Format("2006/01/02 15:04:05.000") + " "
	} else {
		timestamp = ""
	}

	var fileName string
	if thisLabelingLogger.enableFileame {
		_, file, line, ok := runtime.Caller(1)
		if ok {
			s := strings.Split(file, "/")
			fileName = fmt.Sprintf("%s line %3d", s[len(s)-1], line) + " "
		} else {
			fileName = "unknown "
		}
	} else {
		fileName = ""
	}

	for _, logger := range thisLabelingLogger.loggers {
		if logger.isEnable {
			if targetLevelFlgs&logger.flg != 0 {
				logger.logSub(timestamp, fileName, msg)
			}
		}
	}
}

// LogMultiLines outputs multi-line messages at the specified log level.
func (thisLabelingLogger *LabelingLogger) LogMultiLines(targetLevelFlgs LogLevel, msg string) {
	if !thisLabelingLogger.isActive(targetLevelFlgs) {
		return
	}

	var timestamp string
	if thisLabelingLogger.enableTimestamp {
		timestamp = time.Now().Format("2006/01/02 15:04:05.000") + " "
	} else {
		timestamp = ""
	}

	var fileName string
	if thisLabelingLogger.enableFileame {
		_, file, line, ok := runtime.Caller(1)
		if ok {
			s := strings.Split(file, "/")
			fileName = fmt.Sprintf("%s line %3d", s[len(s)-1], line) + " "
		} else {
			fileName = "unknown "
		}
	} else {
		fileName = ""
	}

	scanner := bufio.NewScanner(strings.NewReader(msg))
	for scanner.Scan() {
		for _, logger := range thisLabelingLogger.loggers {
			if logger.isEnable {
				if targetLevelFlgs&logger.flg != 0 {
					logger.logSub(timestamp, fileName, scanner.Text())
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		selfLogger.log(timestamp, fileName, err.Error())
	}
}

func (thisLabelingLogger *LabelingLogger) isActive(targetLevelFlgs LogLevel) bool {
	for _, logger := range thisLabelingLogger.loggers {
		if logger.isEnable {
			if targetLevelFlgs&logger.flg != 0 {
				return true
			}
		}
	}
	return false
}
