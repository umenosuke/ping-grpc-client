package labelinglog

import "io"

// SetEnableLevel enables only the output of the specified log level.
func (thisLabelingLogger *LabelingLogger) SetEnableLevel(targetLevelFlgs LogLevel) {
	for _, logger := range thisLabelingLogger.loggers {
		logger.isEnable = targetLevelFlgs&logger.flg != 0
	}
}

// SetIoWriter changes the output destination of the specified log level.
func (thisLabelingLogger *LabelingLogger) SetIoWriter(targetLevelFlgs LogLevel, writer io.Writer) {
	for _, logger := range thisLabelingLogger.loggers {
		logger.Lock()
		logger.writer = writer
		logger.Unlock()
	}
}

// DisableFilename disables the output of the file name of the log caller.
func (thisLabelingLogger *LabelingLogger) DisableFilename() {
	thisLabelingLogger.enableFileame = false
}

// EnableFilename enables output of the file name of the caller of the log.
func (thisLabelingLogger *LabelingLogger) EnableFilename() {
	thisLabelingLogger.enableFileame = true
}

// DisableTimestamp disables log timestamp output.
func (thisLabelingLogger *LabelingLogger) DisableTimestamp() {
	thisLabelingLogger.enableTimestamp = false
}

// EnableTimestamp enables log timestamp output.
func (thisLabelingLogger *LabelingLogger) EnableTimestamp() {
	thisLabelingLogger.enableTimestamp = true
}
