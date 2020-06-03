package main

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/umenosuke/labelinglog"
	pb "github.com/umenosuke/ping-grpc-client/proto/pingGrpc"
	"github.com/umenosuke/pinger4"
)

type tClientWrap struct {
	client      pb.PingerClient
	chStdinText <-chan string
	chCancel    <-chan struct{}
	chCLIStr    chan<- tCliMsg
	config      Config
}

func (thisClient *tClientWrap) start(ctx context.Context) {
	thisClient.chCLIStr <- tCliMsg{
		text:    "Description? ",
		color:   cliColorDefault,
		noBreak: true,
	}
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
		thisClient.chCLIStr <- tCliMsg{
			text:    "target [IP Comment]? ",
			color:   cliColorDefault,
			noBreak: true,
		}
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
		StopPingerSec:         thisClient.config.StopPingerSec,
		IntervalMillisec:      thisClient.config.IntervalMillisec,
		TimeoutMillisec:       thisClient.config.TimeoutMillisec,
		StatisticsCountsNum:   thisClient.config.StatisticsCountsNum,
		StatisticsIntervalSec: thisClient.config.StatisticsIntervalSec,
	}

	res, err := thisClient.client.Start(ctx, req)
	if res != nil {
		thisClient.chCLIStr <- tCliMsg{
			text:    "start ID: " + strconv.FormatUint(uint64(res.GetPingerID()), 10),
			color:   cliColorDefault,
			noBreak: false,
		}
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
	thisClient.printListSummary(ctx)

	thisClient.chCLIStr <- tCliMsg{
		text:    "PingerID? ",
		color:   cliColorDefault,
		noBreak: true,
	}
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
	thisClient.printList(ctx)
}

func (thisClient *tClientWrap) info(ctx context.Context) {
	thisClient.printListSummary(ctx)

	thisClient.chCLIStr <- tCliMsg{
		text:    "PingerID? ",
		color:   cliColorDefault,
		noBreak: true,
	}
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
	thisClient.printListSummary(ctx)

	thisClient.chCLIStr <- tCliMsg{
		text:    "PingerID? ",
		color:   cliColorDefault,
		noBreak: true,
	}
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
				thisClient.chCLIStr <- tCliMsg{
					text: fmt.Sprintf("R O - %s - %15s - %05d - %7.2fms - %s",
						time.Unix(0, result.GetReceiveTimeUnixNanosec()).Format("2006/01/02 15:04:05.000"),
						targets[result.GetTargetID()].IPAddress,
						result.GetSequence(),
						float64(result.GetReceiveTimeUnixNanosec()-result.GetSendTimeUnixNanosec())/1000/1000,
						targets[result.GetTargetID()].Comment,
					),
					color:   cliColorGreen,
					noBreak: false,
				}
			case pb.IcmpResult_IcmpResultTypeReceiveAfterTimeout:
				thisClient.chCLIStr <- tCliMsg{
					text: fmt.Sprintf("R ? - %s - %15s - %05d - %7.2fms after Timeout - %s",
						time.Unix(0, result.GetReceiveTimeUnixNanosec()).Format("2006/01/02 15:04:05.000"),
						targets[result.GetTargetID()].IPAddress,
						result.GetSequence(),
						float64(result.GetReceiveTimeUnixNanosec()-result.GetSendTimeUnixNanosec())/1000/1000,
						targets[result.GetTargetID()].Comment,
					),
					color:   cliColorYellow,
					noBreak: false,
				}
			case pb.IcmpResult_IcmpResultTypeTTLExceeded:
				thisClient.chCLIStr <- tCliMsg{
					text: fmt.Sprintf("R X - %s - %15s - %05d - TTL Exceeded from %s - %s",
						time.Unix(0, result.GetReceiveTimeUnixNanosec()).Format("2006/01/02 15:04:05.000"),
						targets[result.GetTargetID()].IPAddress,
						result.GetSequence(),
						pinger4.BinIPv4Address2String(pinger4.BinIPv4Address(result.GetBinPeerIP())),
						targets[result.GetTargetID()].Comment,
					),
					color:   cliColorRed,
					noBreak: false,
				}
			case pb.IcmpResult_IcmpResultTypeTimeout:
				thisClient.chCLIStr <- tCliMsg{
					text: fmt.Sprintf("R X - %s - %15s - %05d - Timeout!! - %s",
						time.Unix(0, result.GetReceiveTimeUnixNanosec()).Format("2006/01/02 15:04:05.000"),
						targets[result.GetTargetID()].IPAddress,
						result.GetSequence(),
						targets[result.GetTargetID()].Comment,
					),
					color:   cliColorRed,
					noBreak: false,
				}
			}
		}
	}
}

func (thisClient *tClientWrap) count(ctx context.Context) {
	thisClient.printListSummary(ctx)

	thisClient.chCLIStr <- tCliMsg{
		text:    "PingerID? ",
		color:   cliColorDefault,
		noBreak: true,
	}
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
			thisClient.chCLIStr <- tCliMsg{
				text:    "",
				color:   cliColorDefault,
				noBreak: false,
			}

			counts := res.GetTargets()
			timeNowStr := time.Now().Format("2006/01/02 15:04:05.000")
			for _, c := range counts {
				str := ""
				strColor := cliColorDefault

				rate := c.GetCount() * 100 / resultListNum
				var ox string
				if rate < thisClient.config.CountRateThreshold {
					ox = "X"
					strColor = cliColorRed
				} else {
					ox = "O"
					strColor = cliColorGreen
				}
				targetID := c.GetTargetID()
				str += fmt.Sprintf("S %s - %s - %15s - %03d%% in last %d - %s",
					ox,
					timeNowStr,
					targets[targetID].IPAddress,
					rate,
					resultListNum,
					targets[targetID].Comment,
				)

				thisClient.chCLIStr <- tCliMsg{
					text:    str,
					color:   strColor,
					noBreak: false,
				}
			}
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
	str += "================================================================"

	thisClient.chCLIStr <- tCliMsg{
		text:    str,
		color:   cliColorDefault,
		noBreak: false,
	}
}

func (thisClient *tClientWrap) printList(ctx context.Context) {
	list, err := thisClient.client.GetPingerList(ctx, &pb.Null{})
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
	}

	if list != nil {
		pingers := list.GetPingers()
		sort.Slice(pingers, func(i, j int) bool { return pingers[i].GetStartUnixNanosec() < pingers[j].GetStartUnixNanosec() })

		str := ""

		str += "================================================================\n"
		for _, p := range pingers {
			str += "PingerID          : " + strconv.FormatUint(uint64(p.GetPingerID()), 10) + "\n"
			str += "Description       : " + p.GetDescription() + "\n"
			str += "StartUnixNanosec  : " + time.Unix(0, int64(p.GetStartUnixNanosec())).Format("2006/01/02 15:04:05.000") + "\n"
			str += "ExpireUnixNanosec : " + time.Unix(0, int64(p.GetExpireUnixNanosec())).Format("2006/01/02 15:04:05.000") + "\n"
			str += "================================================================\n"
		}

		thisClient.chCLIStr <- tCliMsg{
			text:    str,
			color:   cliColorDefault,
			noBreak: true,
		}
	}
}

func (thisClient *tClientWrap) printListSummary(ctx context.Context) {
	list, err := thisClient.client.GetPingerList(ctx, &pb.Null{})
	if err != nil {
		logger.Log(labelinglog.FlgError, "\""+err.Error()+"\"")
	}

	if list != nil {
		pingers := list.GetPingers()
		sort.Slice(pingers, func(i, j int) bool { return pingers[i].GetStartUnixNanosec() < pingers[j].GetStartUnixNanosec() })

		str := ""

		str += "================================================================\n"
		str += "running Pingers (start order)\n"
		str += "----------------------------------------------------------------\n"
		str += "PingerID : Description\n"
		for _, p := range pingers {
			str += strconv.FormatUint(uint64(p.GetPingerID()), 10) + " : " + p.GetDescription() + "\n"
		}
		str += "================================================================\n"

		thisClient.chCLIStr <- tCliMsg{
			text:    str,
			color:   cliColorDefault,
			noBreak: true,
		}
	}
}
