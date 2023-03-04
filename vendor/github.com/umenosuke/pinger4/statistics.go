package pinger4

import (
	"context"
	"sync"

	"github.com/umenosuke/labelinglog"
)

func (thisPinger *Pinger) statistics(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer thisPinger.logger.Log(labelinglog.FlgDebug, "finish")

	wgChild := sync.WaitGroup{}
	defer wgChild.Wait()
	defer thisPinger.cancelFunc()

	for {
		select {
		case <-ctx.Done():
			thisPinger.logger.Log(labelinglog.FlgDebug, "stop request received")
			return
		case result := <-thisPinger.chIcmpResult:
			switch result.ResultType {
			case IcmpResultTypeReceive:
				thisPinger.addResult(result.IcmpTargetID, 1)
			case IcmpResultTypeReceiveAfterTimeout:
			case IcmpResultTypeTTLExceeded:
				thisPinger.addResult(result.IcmpTargetID, 0)
			case IcmpResultTypeTimeout:
				thisPinger.addResult(result.IcmpTargetID, 0)
			default:
				thisPinger.addResult(result.IcmpTargetID, 0)
			}
			for _, ch := range thisPinger.chIcmpResultsSubscriber {
				select {
				case ch <- result:
				default:
					thisPinger.logger.Log(labelinglog.FlgWarn, "busy results subscriber skip")
				}
			}
		}
	}
}

func (thisPinger *Pinger) addResult(targetID BinIPv4Address, res int64) {
	target := thisPinger.statisticsData.targets[targetID]
	target.Lock()
	defer target.Unlock()

	target.Res[target.Index] = res
	target.Index++
	if target.Index >= thisPinger.config.StatisticsCountsNum {
		target.Index = 0
	}
}
