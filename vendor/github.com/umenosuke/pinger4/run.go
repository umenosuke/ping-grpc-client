package pinger4

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/umenosuke/labelinglog"
)

// Run is Pinger start
func (thisPinger *Pinger) Run(ctx context.Context) error {
	thisPinger.isStarted.Lock()
	if thisPinger.isStarted.flg {
		defer thisPinger.isStarted.Unlock()

		msg := "Pinger has already started"
		thisPinger.logger.Log(labelinglog.FlgWarn, msg)
		return errors.New(msg)
	}
	thisPinger.isStarted.flg = true
	thisPinger.isStarted.Unlock()

	defer thisPinger.logger.Log(labelinglog.FlgNotice, "finish Pinger")
	thisPinger.logger.Log(labelinglog.FlgNotice, "start Pinger")

	timeoutLimit := thisPinger.config.IntervalMillisec * responseListNum / 2
	if timeoutLimit < thisPinger.config.TimeoutMillisec {
		thisPinger.logger.Log(labelinglog.FlgWarn, "timeout too long, change timeout ["+strconv.FormatInt(thisPinger.config.TimeoutMillisec, 10)+" to "+strconv.FormatInt(timeoutLimit, 10)+"]")
		thisPinger.config.TimeoutMillisec = timeoutLimit
	}

	childCtx, childCtxCancel := context.WithCancel(context.Background())
	defer childCtxCancel()
	thisPinger.cancelFunc = childCtxCancel

	wgChild := sync.WaitGroup{}
	{
		thisPinger.logger.Log(labelinglog.FlgDebug, "start statistics")
		wgChild.Add(1)
		go thisPinger.statistics(childCtx, &wgChild)

		thisPinger.logger.Log(labelinglog.FlgDebug, "start listener")
		wgChild.Add(1)
		go thisPinger.listener(childCtx, &wgChild)

		thisPinger.logger.Log(labelinglog.FlgDebug, "start broker")
		wgChild.Add(1)
		go thisPinger.broker(childCtx, &wgChild)
	}

	if thisPinger.config.DebugEnable {
		wgChild.Add(1)
		go thisPinger.debugStatus(childCtx, &wgChild)
	}

	resError := error(nil)
	select {
	case <-ctx.Done():
		thisPinger.logger.Log(labelinglog.FlgDebug, "stop request from parent")
	case <-childCtx.Done():
		thisPinger.logger.Log(labelinglog.FlgDebug, "stop request from child")
		msg := "may be fatal error (´・ω・`)"
		thisPinger.logger.Log(labelinglog.FlgError, msg)
		resError = errors.New(msg)
	}

	thisPinger.logger.Log(labelinglog.FlgDebug, "stop request to all chlid")
	{
		thisPinger.cancelFunc()

		c := make(chan struct{})
		go (func() {
			wgChild.Wait()
			close(c)
		})()

		thisPinger.logger.Log(labelinglog.FlgNotice, "waiting for termination ("+strconv.FormatInt(terminateTimeOutSec, 10)+"sec)")
		select {
		case <-c:
			thisPinger.logger.Log(labelinglog.FlgNotice, "terminated successfully")
		case <-time.After(time.Duration(terminateTimeOutSec) * time.Second):
			msg := "forced termination"
			thisPinger.logger.Log(labelinglog.FlgError, msg)
			if resError == nil {
				resError = errors.New(msg)
			}
		}

		thisPinger.logger.Log(labelinglog.FlgDebug, "chIcmpResponse        "+strconv.Itoa(len(thisPinger.chIcmpResponse)))
		thisPinger.logger.Log(labelinglog.FlgDebug, "chIcmpResponseTimeout "+strconv.Itoa(len(thisPinger.chIcmpResponseTimeout)))
		thisPinger.logger.Log(labelinglog.FlgDebug, "chIcmpResult          "+strconv.Itoa(len(thisPinger.chIcmpResult)))
	}

	return resError
}
