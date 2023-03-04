package pinger4

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/umenosuke/labelinglog"
)

func (thisPinger *Pinger) debugStatus(ctx context.Context, wg *sync.WaitGroup) {
	thisPinger.logger.Log(labelinglog.FlgDebug, "start DEBUG_STATS")
	defer wg.Done()
	defer thisPinger.logger.Log(labelinglog.FlgDebug, "finish DEBUG_STATS")

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(thisPinger.config.DebugPrintIntervalSec) * time.Second):
			timeouterCount := atomic.LoadInt64(&thisPinger.status.timeouterCounter)
			thisPinger.logger.Log(labelinglog.FlgDebug, "waiting timeouter "+strconv.FormatInt(timeouterCount, 10))

			resultDropCount := atomic.LoadInt64(&thisPinger.status.resultDropCounter)
			thisPinger.logger.Log(labelinglog.FlgDebug, "total drop result "+strconv.FormatInt(resultDropCount, 10))
		}
	}
}
