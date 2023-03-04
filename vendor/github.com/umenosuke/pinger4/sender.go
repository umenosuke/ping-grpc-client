package pinger4

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"github.com/umenosuke/labelinglog"
)

func (thisPinger *Pinger) broker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer thisPinger.logger.Log(labelinglog.FlgDebug, "finish")

	wgChild := sync.WaitGroup{}
	defer wgChild.Wait()
	defer thisPinger.cancelFunc()

	childCtx, childCtxCancel := context.WithCancel(context.Background())
	defer childCtxCancel()

	chIcmpRequestQueueEntry := make(chan icmpRequestQueueEntry, thisPinger.config.MaxWorkers*2)

	requestQueue := make([]icmpRequestQueueEntry, 0, len(thisPinger.targetsOrder))
	for _, targetID := range thisPinger.targetsOrder {
		requestQueue = append(requestQueue, icmpRequestQueueEntry{
			icmpTargetID: targetID,
			seq:          0,
		})
	}

	maxWorkers := thisPinger.config.MaxWorkers
	queueLen := len(requestQueue)
	if queueLen == 0 {
		thisPinger.logger.Log(labelinglog.FlgError, "target IP list is empty")
		return
	} else if maxWorkers > int64(queueLen) {
		maxWorkers = int64(queueLen)
	}

	for i := int64(0); i < maxWorkers; i++ {
		thisPinger.logger.Log(labelinglog.FlgDebug, "(id "+strconv.FormatInt(i, 10)+")"+" start sendIcmpInterval")
		wgChild.Add(1)
		go thisPinger.sendIcmpInterval(childCtx, &wgChild, chIcmpRequestQueueEntry, i)
	}

	i := 0
	for {
		select {
		case <-ctx.Done():
			thisPinger.logger.Log(labelinglog.FlgDebug, "stop request received")
			return
		case chIcmpRequestQueueEntry <- requestQueue[i]:
			requestQueue[i].seq = (requestQueue[i].seq + 1) & 0xffff
			i++
			if i >= queueLen {
				i = 0
			}
		}
	}
}

func (thisPinger *Pinger) sendIcmpInterval(ctx context.Context, wg *sync.WaitGroup, chIcmpRequestQueueEntry chan icmpRequestQueueEntry, myID int64) {
	defer wg.Done()
	defer thisPinger.logger.Log(labelinglog.FlgDebug, "(id "+strconv.FormatInt(myID, 10)+")"+" finish")

	wgChild := sync.WaitGroup{}
	defer wgChild.Wait()
	defer thisPinger.cancelFunc()

	conn, err := net.ListenPacket("ip4:icmp", thisPinger.config.SourceIPAddress)
	if err != nil {
		thisPinger.logger.Log(labelinglog.FlgError, "(id "+strconv.FormatInt(myID, 10)+") "+err.Error())
		return
	}
	defer conn.Close()

	var request icmpRequestQueueEntry

	if thisPinger.config.StartSendIcmpSmoothing {
		select {
		case <-ctx.Done():
			thisPinger.logger.Log(labelinglog.FlgDebug, "(id "+strconv.FormatInt(myID, 10)+") "+"stop request received")
			return
		case <-time.After(time.Duration(rand.Int63n(thisPinger.config.IntervalMillisec)) * time.Millisecond):
		}
	}

	ticker := time.NewTicker(time.Duration(thisPinger.config.IntervalMillisec) * time.Millisecond)
	defer ticker.Stop()

	data := &tICMPData{}
	wmbd := bytes.NewBuffer(make([]byte, 12))
	wmb := &icmp.Echo{
		ID:   thisPinger.icmpID,
		Seq:  0,
		Data: nil,
	}
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: wmb,
	}
	for {
		select {
		case <-ctx.Done():
			thisPinger.logger.Log(labelinglog.FlgDebug, "(id "+strconv.FormatInt(myID, 10)+") "+"stop request received")
			return
		case request = <-chIcmpRequestQueueEntry:
		}

		nowNanosec := time.Now().UnixNano()

		data.IcmpTargetID = uint32(request.icmpTargetID)
		data.SendTimeNanosec = nowNanosec
		wmbd.Reset()
		binary.Write(wmbd, binary.LittleEndian, data)

		wmb.Seq = request.seq
		wmb.Data = wmbd.Bytes()
		wb, err := wm.Marshal(nil)
		if err != nil {
			thisPinger.logger.Log(labelinglog.FlgError, "(id "+strconv.FormatInt(myID, 10)+") "+err.Error())
			return
		}

		thisPinger.reqList[request.icmpTargetID].Lock()
		thisPinger.reqList[request.icmpTargetID].req[request.seq&(responseListNum-1)].isReceived = false
		thisPinger.reqList[request.icmpTargetID].Unlock()

		if _, err := conn.WriteTo(wb, thisPinger.targets[request.icmpTargetID].netIPAddr); err != nil {
			thisPinger.logger.Log(labelinglog.FlgError, "(id "+strconv.FormatInt(myID, 10)+") "+err.Error())
			return
		}

		wgChild.Add(1)
		thisPinger.setTimeouter(ctx, &wgChild, icmpRequest{
			icmpTargetID:    request.icmpTargetID,
			seq:             request.seq,
			sendTimeNanosec: nowNanosec,
		})

		select {
		case <-ctx.Done():
			thisPinger.logger.Log(labelinglog.FlgDebug, "(id "+strconv.FormatInt(myID, 10)+")"+" stop request received")
			return
		case <-ticker.C:
		}
	}
}

func (thisPinger *Pinger) setTimeouter(ctx context.Context, wgFinish *sync.WaitGroup, request icmpRequest) {
	childCtx, childCtxCancel := context.WithCancel(ctx)
	if _, ok := thisPinger.timeouterList[request.icmpTargetID]; ok {
		thisPinger.timeouterList[request.icmpTargetID].Lock()
		thisPinger.timeouterList[request.icmpTargetID].cancelFunc[request.seq&(responseListNum-1)]()
		thisPinger.timeouterList[request.icmpTargetID].cancelFunc[request.seq&(responseListNum-1)] = childCtxCancel
		thisPinger.timeouterList[request.icmpTargetID].Unlock()
	}

	go (func() {
		defer wgFinish.Done()

		atomic.AddInt64(&thisPinger.status.timeouterCounter, 1)
		defer atomic.AddInt64(&thisPinger.status.timeouterCounter, -1)
		defer childCtxCancel()

		select {
		case <-ctx.Done():
			atomic.AddInt64(&thisPinger.status.timeouterCounter, 1)
		case <-childCtx.Done():
		case <-time.After(time.Duration(thisPinger.config.TimeoutMillisec) * time.Millisecond):
			select {
			case <-ctx.Done():
				atomic.AddInt64(&thisPinger.status.timeouterCounter, 1)
			case <-childCtx.Done():
			case thisPinger.chIcmpResponseTimeout <- icmpResponseTimeout{
				icmpTargetID:       request.icmpTargetID,
				seq:                request.seq,
				sendTimeNanosec:    request.sendTimeNanosec,
				receiveTimeNanosec: time.Now().UnixNano(),
			}:
			}
		}
	})()

	if atomic.LoadInt64(&thisPinger.status.timeouterCounter) >= limitTimeouter {
		thisPinger.addInterval()
		if thisPinger.config.StartSendIcmpSmoothing {
			wait := rand.Int63n(500)
			thisPinger.logger.Log(labelinglog.FlgDebug, "busy timeouter, wait "+strconv.FormatInt(wait, 10)+"ms")
			time.Sleep(time.Duration(wait) * time.Millisecond)
		}
	}
}
