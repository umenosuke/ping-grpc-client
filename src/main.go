package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"github.com/umenosuke/labelinglog"
	pb "github.com/umenosuke/ping-grpc-client/proto/go/pingGrpc"
)

type tCliColor int

const (
	cliColorDefault = tCliColor(iota)
	cliColorRed
	cliColorGreen
	cliColorBlue
	cliColorYellow
)

type tCliMsg struct {
	text    string
	color   tCliColor
	noBreak bool
}

const terminateTimeOutSec = 15

var exitCode = 0

var logger = labelinglog.New("pinger-client", os.Stderr)

var (
	metaVersion  = "unknown"
	metaRevision = "unknown"
)

var (
	argDebugFlag             = flag.Bool("debug", false, "print debug log")
	argServerAddress         = flag.String("S", "127.0.0.1:5555", "server address:port")
	argNoUseTLS              = flag.Bool("noUseTLS", false, "disable tls")
	argClientCertificatePath = flag.String("cCert", "./client_pinger.crt", "client certificate file path")
	argClientPrivateKeyPath  = flag.String("cKey", "./client_pinger.pem", "client private key file path")
	argConfigPath            = flag.String("configPath", "", "config file path")
	argShowConfigFlg         = flag.Bool("printConfig", false, "show default config")
	argShowVersionFlag       = flag.Bool("version", false, "show version")
)

func init() {
	flag.Parse()

	if *argDebugFlag {
		logger.SetEnableLevel(labelinglog.FlgsetAll)
	} else {
		logger.SetEnableLevel(labelinglog.FlgsetCommon)
	}
}

func main() {
	subMain()
	os.Exit(exitCode)
}

func subMain() {
	if *argShowVersionFlag {
		fmt.Fprint(os.Stdout, "Version "+metaVersion+"\n"+"Revision "+metaRevision+"\n")
		return
	}

	if *argShowConfigFlg {
		fmt.Fprint(os.Stdout, configStringify(DefaultConfig())+"\n")
		return
	}

	config, err := configLoad(*argConfigPath)
	if err != nil {
		logger.Log(labelinglog.FlgFatal, err.Error())
		exitCode = 1
		return
	}
	if *argDebugFlag {
		logger.Log(labelinglog.FlgDebug, "now config")
		logger.LogMultiLines(labelinglog.FlgDebug, configStringify(config))
	}

	grpcDialOptions, err := getGrpcDialOptions()
	if err != nil {
		logger.Log(labelinglog.FlgFatal, err.Error())
		exitCode = 1
		return
	}

	conn, err := grpc.Dial(*argServerAddress, grpcDialOptions...)
	if err != nil {
		logger.Log(labelinglog.FlgFatal, err.Error())
		exitCode = 1
		return
	}
	defer conn.Close()

	ctx := context.Background()
	childCtx, childCtxCancel := context.WithCancel(ctx)
	defer childCtxCancel()
	wgFinish := sync.WaitGroup{}

	chCancel := make(chan struct{}, 5)
	wgFinish.Add(1)
	go (func() {
		defer wgFinish.Done()
		defer childCtxCancel()
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, os.Interrupt)
		for {
			select {
			case <-childCtx.Done():
				return
			case sig := <-c:
				switch sig {
				case syscall.SIGINT:
					logger.Log(labelinglog.FlgDebug, "request stop, SIGINT")
					chCancel <- struct{}{}
				default:
					logger.Log(labelinglog.FlgWarn, fmt.Sprintf("unknown syscall [%v]", sig))
				}
			}
		}
	})()

	chCLIStr := make(chan tCliMsg, 200)
	wgFinish.Add(1)
	go (func() {
		defer wgFinish.Done()

		isWindows := runtime.GOOS == "windows"
		for {
			select {
			case <-time.After(time.Second):
			case msg := <-chCLIStr:
				str := ""

				if !isWindows {
					switch msg.color {
					case cliColorRed:
						str += "\x1b[41m\x1b[37m"
					case cliColorGreen:
						str += "\x1b[42m\x1b[37m"
					case cliColorBlue:
						str += "\x1b[44m\x1b[37m"
					case cliColorYellow:
						str += "\x1b[43m\x1b[37m"
					case cliColorDefault:
						str += "\x1b[49m\x1b[39m"
					}
				}

				str += msg.text

				if !isWindows {
					str += "\x1b[49m\x1b[39m"
				}

				if msg.noBreak {
					fmt.Print(str)
				} else {
					fmt.Println(str)
				}
				continue
			}
			select {
			case <-childCtx.Done():
				return
			default:
			}
		}
	})()

	wgFinish.Add(1)
	go (func() {
		defer wgFinish.Done()
		defer childCtxCancel()
		chStdinText := make(chan string, 5)

		client := tClientWrap{
			client:      pb.NewPingerClient(conn),
			chStdinText: chStdinText,
			chCancel:    chCancel,
			chCLIStr:    chCLIStr,
		}

		go (func() {
			defer childCtxCancel()

			scanner := bufio.NewScanner(os.Stdin)
			logger.Log(labelinglog.FlgDebug, "start scanner")
			defer logger.Log(labelinglog.FlgDebug, "finish scanner")
			for {
				select {
				case <-childCtx.Done():
					logger.Log(labelinglog.FlgDebug, "stop scanner")
					return
				default:
				}

				if scanner.Scan() {
					text := scanner.Text()
					chStdinText <- strings.Trim(text, " \t")
				} else {
					if err := scanner.Err(); err != nil {
						logger.Log(labelinglog.FlgError, "scanner: "+err.Error())
						return
					}

					logger.Log(labelinglog.FlgDebug, "scanner: stdin closed, scanner reNew")
					scanner = bufio.NewScanner(os.Stdin)
				}
			}
		})()

		logger.Log(labelinglog.FlgDebug, "start input")
		defer logger.Log(labelinglog.FlgDebug, "finish input")
		var command string
		prompt := tCliMsg{
			text:    "\n" + *argServerAddress + "> ",
			color:   cliColorDefault,
			noBreak: true,
		}
		for {
			chCLIStr <- prompt

			select {
			case <-childCtx.Done():
				logger.Log(labelinglog.FlgDebug, "stop input, childCtx.Done")
				return
			case <-chCancel:
				logger.Log(labelinglog.FlgDebug, "stop input, chCancel")
				return
			case command = <-chStdinText:
			}

			switch command {
			case "s", "st":
				chCLIStr <- tCliMsg{
					text: "" +
						"start : start pinger\n" +
						"stop  : stop pinger",
					color:   cliColorDefault,
					noBreak: false,
				}
			case "sta", "star", "start":
				chCLIStr <- tCliMsg{
					text:    "[start]",
					color:   cliColorDefault,
					noBreak: false,
				}
				client.start(childCtx)
			case "sto", "stop":
				chCLIStr <- tCliMsg{
					text:    "[stop]",
					color:   cliColorDefault,
					noBreak: false,
				}
				client.stop(childCtx)
			case "l", "li", "lis", "list":
				chCLIStr <- tCliMsg{
					text:    "[list]",
					color:   cliColorDefault,
					noBreak: false,
				}
				client.list(childCtx)
			case "i", "in", "inf", "info":
				chCLIStr <- tCliMsg{
					text:    "[info]",
					color:   cliColorDefault,
					noBreak: false,
				}
				client.info(childCtx)
			case "r", "re", "res", "resu", "resul", "result":
				chCLIStr <- tCliMsg{
					text:    "[result]",
					color:   cliColorDefault,
					noBreak: false,
				}
				client.result(childCtx)
			case "c", "co", "cou", "coun", "count":
				chCLIStr <- tCliMsg{
					text:    "[count]",
					color:   cliColorDefault,
					noBreak: false,
				}
				client.count(childCtx, 80)
			case "q", "qu", "qui", "quit":
				chCLIStr <- tCliMsg{
					text:    "[quit]",
					color:   cliColorDefault,
					noBreak: false,
				}
				return
			case "e", "ex", "exi", "exit":
				chCLIStr <- tCliMsg{
					text:    "[exit]",
					color:   cliColorDefault,
					noBreak: false,
				}
				return
			case "?", "h", "he", "hel", "help":
				chCLIStr <- tCliMsg{
					text: "" +
						"start  : start pinger\n" +
						"stop   : stop pinger\n" +
						"\n" +
						"list   : show pinger list\n" +
						"info   : show pinger info\n" +
						"result : show ping result\n" +
						"count  : show ping statistics\n" +
						"\n" +
						"quit   : exit client\n" +
						"exit   : exit client\n" +
						"\n" +
						"help   : (this) show help",
					color:   cliColorDefault,
					noBreak: false,
				}
			case "":
				logger.Log(labelinglog.FlgDebug, "input empty")
			default:
				chCLIStr <- tCliMsg{
					text: "" +
						"unknown command \"" + command + "\"\n" +
						"? : show commands",
					color:   cliColorDefault,
					noBreak: false,
				}
			}
		}
	})()

	{
		logger.Log(labelinglog.FlgDebug, "wait childCtx.Done")
		<-childCtx.Done()
		logger.Log(labelinglog.FlgDebug, "detect childCtx.Done")

		c := make(chan struct{})
		go (func() {
			wgFinish.Wait()
			close(c)
		})()

		logger.Log(labelinglog.FlgNotice, "waiting for termination ("+strconv.FormatInt(terminateTimeOutSec, 10)+"sec)")
		select {
		case <-c:
			logger.Log(labelinglog.FlgNotice, "terminated successfully")
		case <-time.After(time.Duration(terminateTimeOutSec) * time.Second):
			logger.Log(labelinglog.FlgWarn, "forced termination")
		}
	}

	if runtime.GOOS != "windows" {
		fmt.Print("\x1b[49m\x1b[39m\x1b[0m")
	}
	fmt.Println("bye")
}

func getGrpcDialOptions() ([]grpc.DialOption, error) {
	grpcDialOptions := make([]grpc.DialOption, 0)

	{
		grpcDialOptions = append(grpcDialOptions, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                1 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}))
	}

	if !*argNoUseTLS {
		clientCert, err :=
			tls.LoadX509KeyPair(
				*argClientCertificatePath,
				*argClientPrivateKeyPath)
		if err != nil {
			return nil, err
		}

		caCert, err := ioutil.ReadFile("ca.crt")
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		creds := credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{clientCert},
			RootCAs:      caCertPool,
		})

		grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(creds))
	} else {
		grpcDialOptions = append(grpcDialOptions, grpc.WithInsecure())
	}

	return grpcDialOptions, nil
}
