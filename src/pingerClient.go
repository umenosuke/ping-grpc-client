package main

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"umenosuke.net/labelinglog"
	pb "umenosuke.net/ping-grpc/proto/go/pingGrpc"
	"umenosuke.net/pinger4"
)

type tClientWrap struct {
	client      pb.PingerClient
	chStdinText <-chan string
	chCancel    <-chan struct{}
	chCLIStr    chan<- string
}

func (thisClient *tClientWrap) start(ctx context.Context) {
	thisClient.chCLIStr <- "Description? "
	var descStr string
	select {
	case <-ctx.Done():
		return
	case <-thisClient.chCancel:
		return
	case descStr = <-thisClient.chStdinText:
	}

	targetList := make([]*pb.StartRequest_IcmpTarget, 0)
	reg := regexp.MustCompile(`^([^# \t]*)[# \t]*(.*)$`)
	for {
		thisClient.chCLIStr <- "target [IP Comment]? "
		var targetStr string
		select {
		case <-ctx.Done():
			return
		case <-thisClient.chCancel:
			return
		case targetStr = <-thisClient.chStdinText:
		}
		if targetStr == "" {
			break
		}

		result := reg.FindStringSubmatch(targetStr)
		if result != nil {
			targetIP := result[1]
			targetComment := result[2]

			targetList = append(targetList, &pb.StartRequest_IcmpTarget{
				TargetIP: targetIP,
				Comment:  targetComment,
			})
		}
	}

	req := &pb.StartRequest{
		Description:           descStr,
		Targets:               targetList,
		IntervalMillisec:      1000,
		TimeoutMillisec:       1000,
		StopPingerSec:         3600 * 4,
		StatisticsCountsNum:   10,
		StatisticsIntervalSec: 1,
	}

	res, err := thisClient.client.Start(ctx, req)
	if res != nil {
		thisClient.chCLIStr <- "start ID: " + strconv.FormatUint(uint64(res.GetPingerID()), 10) + "\n"
	}
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
	}

	info, err := thisClient.client.GetPingerInfo(ctx, &pb.PingerID{PingerID: uint32(res.GetPingerID())})
	if info != nil {
		thisClient.printInfo(info)
	}
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
	}
}

func (thisClient *tClientWrap) stop(ctx context.Context) {
	thisClient.chCLIStr <- "PingerID? "
	var pingerID string
	select {
	case <-ctx.Done():
		return
	case <-thisClient.chCancel:
		return
	case pingerID = <-thisClient.chStdinText:
	}

	id, err := strconv.Atoi(pingerID)
	if err != nil {
		logger.Log(labelinglog.FlgError, "error : \""+pingerID+"\"")
		return
	}

	_, err = thisClient.client.Stop(ctx, &pb.PingerID{PingerID: uint32(id)})
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
	}
}

func (thisClient *tClientWrap) list(ctx context.Context) {
	list, err := thisClient.client.GetPingerList(ctx, &pb.Null{})
	if list != nil {
		thisClient.printList(list)
	}
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
	}
}

func (thisClient *tClientWrap) info(ctx context.Context) {
	thisClient.chCLIStr <- "PingerID? "
	var pingerID string
	select {
	case <-ctx.Done():
		return
	case <-thisClient.chCancel:
		return
	case pingerID = <-thisClient.chStdinText:
	}

	id, err := strconv.Atoi(pingerID)
	if err != nil {
		logger.Log(labelinglog.FlgError, "error : \""+pingerID+"\"")
		return
	}

	info, err := thisClient.client.GetPingerInfo(ctx, &pb.PingerID{PingerID: uint32(id)})
	if info != nil {
		thisClient.printInfo(info)
	}
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
	}
}

func (thisClient *tClientWrap) result(ctx context.Context) {
	thisClient.chCLIStr <- "PingerID? "
	var pingerID string
	select {
	case <-ctx.Done():
		return
	case <-thisClient.chCancel:
		return
	case pingerID = <-thisClient.chStdinText:
	}

	id, err := strconv.Atoi(pingerID)
	if err != nil {
		logger.Log(labelinglog.FlgError, "error : \""+pingerID+"\"")
		return
	}

	info, err := thisClient.client.GetPingerInfo(ctx, &pb.PingerID{PingerID: uint32(id)})
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
		return
	}
	thisClient.printInfo(info)

	targets := make(map[uint32]struct {
		IPAddress string
		Comment   string
	})
	for _, t := range info.GetTargets() {
		comment := t.GetComment()
		if t.GetTargetIP() != t.GetTargetBinIP() {
			comment += " (FQDN: " + t.GetTargetIP() + ")"
		}

		targets[t.GetTargetID()] = struct {
			IPAddress string
			Comment   string
		}{
			IPAddress: t.GetTargetBinIP(),
			Comment:   comment,
		}
	}

	childCtx, childCtxCancel := context.WithCancel(ctx)
	defer childCtxCancel()
	go (func() {
		defer childCtxCancel()
		select {
		case <-ctx.Done():
		case <-childCtx.Done():
		case <-thisClient.chCancel:
		}
	})()

	stream, err := thisClient.client.GetsIcmpResult(childCtx, &pb.PingerID{PingerID: uint32(id)})
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
		return
	}
	for {
		result, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return
			}
			if status.Code(err) == codes.Canceled {
				return
			}
			logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
			return
		}

		if result != nil {
			switch result.GetType() {
			case pb.IcmpResult_IcmpResultTypeReceive:
				thisClient.chCLIStr <- fmt.Sprintf("R O %s - %15s - %05d - %7.2fms - %s\n",
					time.Unix(0, result.GetReceiveTimeUnixNanosec()).Format("2006/01/02 15:04:05.000"),
					targets[result.GetTargetID()].IPAddress,
					result.GetSequence(),
					float64(result.GetReceiveTimeUnixNanosec()-result.GetSendTimeUnixNanosec())/1000/1000,
					targets[result.GetTargetID()].Comment,
				)
			case pb.IcmpResult_IcmpResultTypeReceiveAfterTimeout:
				thisClient.chCLIStr <- fmt.Sprintf("R ? %s - %15s - %05d - %7.2fms after Timeout - %s\n",
					time.Unix(0, result.GetReceiveTimeUnixNanosec()).Format("2006/01/02 15:04:05.000"),
					targets[result.GetTargetID()].IPAddress,
					result.GetSequence(),
					float64(result.GetReceiveTimeUnixNanosec()-result.GetSendTimeUnixNanosec())/1000/1000,
					targets[result.GetTargetID()].Comment,
				)
			case pb.IcmpResult_IcmpResultTypeTTLExceeded:
				thisClient.chCLIStr <- fmt.Sprintf("R X %s - %15s - %05d - TTL Exceeded from %s - %s\n",
					time.Unix(0, result.GetReceiveTimeUnixNanosec()).Format("2006/01/02 15:04:05.000"),
					targets[result.GetTargetID()].IPAddress,
					result.GetSequence(),
					pinger4.BinIPv4Address2String(pinger4.BinIPv4Address(result.GetBinPeerIP())),
					targets[result.GetTargetID()].Comment,
				)
			case pb.IcmpResult_IcmpResultTypeTimeout:
				thisClient.chCLIStr <- fmt.Sprintf("R X %s - %15s - %05d - Timeout!! - %s\n",
					time.Unix(0, result.GetReceiveTimeUnixNanosec()).Format("2006/01/02 15:04:05.000"),
					targets[result.GetTargetID()].IPAddress,
					result.GetSequence(),
					targets[result.GetTargetID()].Comment,
				)
			}
		}
	}
}

func (thisClient *tClientWrap) count(ctx context.Context, rateThreshold int64) {
	thisClient.chCLIStr <- "PingerID? "
	var pingerID string
	select {
	case <-ctx.Done():
		return
	case <-thisClient.chCancel:
		return
	case pingerID = <-thisClient.chStdinText:
	}

	id, err := strconv.Atoi(pingerID)
	if err != nil {
		logger.Log(labelinglog.FlgError, "error : \""+pingerID+"\"")
		return
	}

	info, err := thisClient.client.GetPingerInfo(ctx, &pb.PingerID{PingerID: uint32(id)})
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
		return
	}
	thisClient.printInfo(info)

	targets := make(map[uint32]struct {
		IPAddress string
		Comment   string
	})
	for _, t := range info.GetTargets() {
		comment := t.GetComment()
		if t.GetTargetIP() != t.GetTargetBinIP() {
			comment += " (FQDN: " + t.GetTargetIP() + ")"
		}

		targets[t.GetTargetID()] = struct {
			IPAddress string
			Comment   string
		}{
			IPAddress: t.GetTargetBinIP(),
			Comment:   comment,
		}
	}
	resultListNum := int64(info.GetStatisticsCountsNum())

	childCtx, childCtxCancel := context.WithCancel(ctx)
	defer childCtxCancel()
	go (func() {
		defer childCtxCancel()
		select {
		case <-ctx.Done():
		case <-childCtx.Done():
		case <-thisClient.chCancel:
		}
	})()

	stream, err := thisClient.client.GetsStatistics(childCtx, &pb.PingerID{PingerID: uint32(id)})
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
		return
	}
	for {
		res, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return
			}
			if status.Code(err) == codes.Canceled {
				return
			}
			logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
			return
		}

		if res != nil {
			counts := res.GetTargets()
			str := ""
			timeNowStr := time.Now().Format("2006/01/02 15:04:05.000")
			for _, c := range counts {
				rate := c.GetCount() * 100 / resultListNum
				var ox string
				if rate < rateThreshold {
					ox = "X"
				} else {
					ox = "O"
				}
				targetID := c.GetTargetID()
				str += fmt.Sprintf("S %s - %s - %15s - %03d%% in last %d - %s\n",
					ox,
					timeNowStr,
					targets[targetID].IPAddress,
					rate,
					resultListNum,
					targets[targetID].Comment,
				)
			}
			thisClient.chCLIStr <- str
		}
	}
}

func (thisClient *tClientWrap) printInfo(info *pb.PingerInfo) {
	str := ""

	str += "================================================================\n"
	str += "Description           : " + info.GetDescription() + "\n"
	str += "Targets               : \n"
	for _, t := range info.GetTargets() {
		str += "                        " + "IP     : " + t.GetTargetIP() + "\n"
		str += "                        " + "BinIP  : " + t.GetTargetBinIP() + "\n"
		str += "                        " + "Comment: " + t.GetComment() + "\n"
		str += "                        ----------------------------------------\n"
	}
	str += "IntervalMillisec      : " + strconv.FormatUint(info.GetIntervalMillisec(), 10) + "\n"
	str += "TimeoutMillisec       : " + strconv.FormatUint(info.GetTimeoutMillisec(), 10) + "\n"
	str += "StatisticsCountsNum   : " + strconv.FormatUint(info.GetStatisticsCountsNum(), 10) + "\n"
	str += "StatisticsIntervalSec : " + strconv.FormatUint(info.GetStatisticsIntervalSec(), 10) + "\n"
	str += "StartUnixNanosec      : " + time.Unix(0, int64(info.GetStartUnixNanosec())).Format("2006/01/02 15:04:05.000") + "\n"
	str += "ExpireUnixNanosec     : " + time.Unix(0, int64(info.GetExpireUnixNanosec())).Format("2006/01/02 15:04:05.000") + "\n"
	str += "================================================================\n"

	thisClient.chCLIStr <- str
}

func (thisClient *tClientWrap) printList(list *pb.PingerList) {
	str := ""

	str += "================================================================\n"
	for _, p := range list.GetPingers() {
		str += "PingerID          : " + strconv.FormatUint(uint64(p.GetPingerID()), 10) + "\n"
		str += "Description       : " + p.GetDescription() + "\n"
		str += "StartUnixNanosec  : " + time.Unix(0, int64(p.GetStartUnixNanosec())).Format("2006/01/02 15:04:05.000") + "\n"
		str += "ExpireUnixNanosec : " + time.Unix(0, int64(p.GetExpireUnixNanosec())).Format("2006/01/02 15:04:05.000") + "\n"
		str += "================================================================\n"
	}

	thisClient.chCLIStr <- str
}