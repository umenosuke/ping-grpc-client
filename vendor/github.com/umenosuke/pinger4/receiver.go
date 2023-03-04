package pinger4

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"github.com/umenosuke/labelinglog"
)

func (thisPinger *Pinger) listener(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer thisPinger.logger.Log(labelinglog.FlgDebug, "finish")

	wgChild := sync.WaitGroup{}
	defer wgChild.Wait()
	defer thisPinger.cancelFunc()

	childCtx, childCtxCancel := context.WithCancel(context.Background())
	defer childCtxCancel()

	thisPinger.logger.Log(labelinglog.FlgDebug, "start responseParser")
	wgChild.Add(1)
	go thisPinger.responseParser(childCtx, &wgChild)

	conn, err := icmp.ListenPacket("ip4:icmp", thisPinger.config.SourceIPAddress)
	if err != nil {
		thisPinger.logger.Log(labelinglog.FlgError, err.Error())
		return
	}
	defer conn.Close()

	rb := make([]byte, responseMTU)
	for {
		select {
		case <-ctx.Done():
			thisPinger.logger.Log(labelinglog.FlgDebug, "stop request received")
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, peer, err := conn.ReadFrom(rb)
		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Timeout() {
					continue
				} else {
					thisPinger.logger.Log(labelinglog.FlgError, err.Error())
					return
				}
			} else {
				thisPinger.logger.Log(labelinglog.FlgError, err.Error())
				return
			}
		}
		nowNanosec := time.Now().UnixNano()

		icmpMessage, err := icmp.ParseMessage(1, rb[:n])
		if err != nil {
			continue
		}
		if body, ok := icmpMessage.Body.(*icmp.Echo); ok {
			id := body.ID
			if thisPinger.icmpID != id {
				continue
			}
		} else if body, ok := icmpMessage.Body.(*icmp.TimeExceeded); ok {
			data := body.Data
			id := int(binary.BigEndian.Uint16(data[24:26]))
			if thisPinger.icmpID != id {
				continue
			}
		} else {
			continue
		}

		select {
		case thisPinger.chIcmpResponse <- icmpResponse{
			peer:               peer,
			message:            icmpMessage,
			receiveTimeNanosec: nowNanosec,
		}:
		default:
			thisPinger.logger.Log(labelinglog.FlgWarn, "busy responseParser")
			thisPinger.addInterval()
		}
	}
}

func (thisPinger *Pinger) responseParser(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer thisPinger.logger.Log(labelinglog.FlgDebug, "finish")

	wgChild := sync.WaitGroup{}
	defer wgChild.Wait()
	defer thisPinger.cancelFunc()

	recvRes := func(res icmpResponse) {
		wgChild.Add(1)
		go (func() {
			defer wgChild.Done()

			switch res.message.Type {
			case ipv4.ICMPTypeEchoReply:
				if body, ok := res.message.Body.(*icmp.Echo); ok {
					data := &tICMPData{}
					binary.Read(bytes.NewReader(body.Data), binary.LittleEndian, data)

					targetID := BinIPv4Address(data.IcmpTargetID)
					seq := body.Seq
					if res.receiveTimeNanosec-data.SendTimeNanosec <= thisPinger.config.TimeoutMillisec*1000*1000 {
						if _, ok := thisPinger.reqList[targetID]; ok {
							thisPinger.reqList[targetID].Lock()
							thisPinger.reqList[targetID].req[seq&(responseListNum-1)].isReceived = true
							thisPinger.reqList[targetID].Unlock()

							if _, ok := thisPinger.timeouterList[targetID]; ok {
								thisPinger.timeouterList[targetID].Lock()
								thisPinger.timeouterList[targetID].cancelFunc[seq&(responseListNum-1)]()
								thisPinger.timeouterList[targetID].Unlock()
							}

							select {
							case thisPinger.chIcmpResult <- IcmpResult{
								ResultType:             IcmpResultTypeReceive,
								IcmpTargetID:           targetID,
								Seq:                    seq,
								SendTimeUnixNanosec:    data.SendTimeNanosec,
								ReceiveTimeUnixNanosec: res.receiveTimeNanosec,
							}:
							default:
								thisPinger.logger.Log(labelinglog.FlgWarn, "busy statistics")
								atomic.AddInt64(&thisPinger.status.resultDropCounter, 1)
							}
						}
					} else {
						select {
						case thisPinger.chIcmpResult <- IcmpResult{
							ResultType:             IcmpResultTypeReceiveAfterTimeout,
							IcmpTargetID:           targetID,
							Seq:                    seq,
							SendTimeUnixNanosec:    data.SendTimeNanosec,
							ReceiveTimeUnixNanosec: res.receiveTimeNanosec,
						}:
						default:
							thisPinger.logger.Log(labelinglog.FlgWarn, "busy statistics")
							atomic.AddInt64(&thisPinger.status.resultDropCounter, 1)
						}
					}
				}
			case ipv4.ICMPTypeTimeExceeded:
				if body, ok := res.message.Body.(*icmp.TimeExceeded); ok {
					data := body.Data

					targetID := BinIPv4Address(binary.BigEndian.Uint32(data[16:20]))
					seq := int(binary.BigEndian.Uint16(data[26:]))
					if _, ok := thisPinger.reqList[targetID]; ok {
						thisPinger.reqList[targetID].Lock()
						isReceived := thisPinger.reqList[targetID].req[seq&(responseListNum-1)].isReceived
						thisPinger.reqList[targetID].req[seq&(responseListNum-1)].isReceived = true
						thisPinger.reqList[targetID].Unlock()

						if !isReceived {
							if _, ok := thisPinger.timeouterList[targetID]; ok {
								thisPinger.timeouterList[targetID].Lock()
								thisPinger.timeouterList[targetID].cancelFunc[seq&(responseListNum-1)]()
								thisPinger.timeouterList[targetID].Unlock()
							}

							select {
							case thisPinger.chIcmpResult <- IcmpResult{
								ResultType:             IcmpResultTypeTTLExceeded,
								IcmpTargetID:           targetID,
								BinPeerIP:              NetIP2BinIPv4Address(net.ParseIP(res.peer.String())),
								Seq:                    seq,
								ReceiveTimeUnixNanosec: res.receiveTimeNanosec,
							}:
							default:
								thisPinger.logger.Log(labelinglog.FlgWarn, "busy statistics")
								atomic.AddInt64(&thisPinger.status.resultDropCounter, 1)
							}
						}
					}
				}
			default:
			}
		})()
	}

	recvTimeout := func(timeoutReq icmpResponseTimeout) {
		for false {
			select {
			case res := <-thisPinger.chIcmpResponse:
				recvRes(res)
				continue
			default:
			}
		}

		wgChild.Add(1)
		go (func() {
			defer wgChild.Done()

			targetID := timeoutReq.icmpTargetID
			seq := timeoutReq.seq
			if _, ok := thisPinger.reqList[targetID]; ok {
				thisPinger.reqList[targetID].Lock()
				isReceived := thisPinger.reqList[targetID].req[seq&(responseListNum-1)].isReceived
				thisPinger.reqList[targetID].req[seq&(responseListNum-1)].isReceived = true
				thisPinger.reqList[targetID].Unlock()

				if !isReceived {
					select {
					case thisPinger.chIcmpResult <- IcmpResult{
						ResultType:             IcmpResultTypeTimeout,
						IcmpTargetID:           targetID,
						Seq:                    seq,
						SendTimeUnixNanosec:    timeoutReq.sendTimeNanosec,
						ReceiveTimeUnixNanosec: timeoutReq.receiveTimeNanosec,
					}:
					default:
						thisPinger.logger.Log(labelinglog.FlgWarn, "busy statistics")
						atomic.AddInt64(&thisPinger.status.resultDropCounter, 1)
					}
				}
			}
		})()
	}

	for {
		select {
		case <-ctx.Done():
			thisPinger.logger.Log(labelinglog.FlgDebug, "stop request received")
			return
		case res := <-thisPinger.chIcmpResponse:
			recvRes(res)
		case timeoutReq := <-thisPinger.chIcmpResponseTimeout:
			recvTimeout(timeoutReq)
		}
	}
}
