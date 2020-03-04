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
	argNoUseTLS              = flag.Bool("noUseTLS", false, "enable tls")
	argClientCertificatePath = flag.String("cCert", "./client_pinger.crt", "client certificate file path")
	argClientPrivateKeyPath  = flag.String("cKey", "./client_pinger.pem", "client private key file path")
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
			logger.Log(labelinglog.FlgFatal, err.Error())
			exitCode = 1
			return
		}

		caCert, err := ioutil.ReadFile("ca.crt")
		if err != nil {
			logger.Log(labelinglog.FlgFatal, err.Error())
			exitCode = 1
			return
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

	chCLIStr := make(chan string, 200)
	wgFinish.Add(1)
	go (func() {
		defer wgFinish.Done()

		for {
			select {
			case <-time.After(time.Second):
			case str := <-chCLIStr:
				fmt.Print(str)
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
		for {
			chCLIStr <- "\n" + *argServerAddress + "> "

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
				str := ""
				str += "start : start pinger\n"
				str += "stop  : stop pinger\n"
				chCLIStr <- str
			case "sta", "star", "start":
				chCLIStr <- "[start]\n"
				client.start(childCtx)
			case "sto", "stop":
				chCLIStr <- "[stop]\n"
				client.stop(childCtx)
			case "l", "li", "lis", "list":
				chCLIStr <- "[list]\n"
				client.list(childCtx)
			case "i", "in", "inf", "info":
				chCLIStr <- "[info]\n"
				client.info(childCtx)
			case "r", "re", "res", "resu", "resul", "result":
				chCLIStr <- "[result]\n"
				client.result(childCtx)
			case "c", "co", "cou", "coun", "count":
				chCLIStr <- "[count]\n"
				client.count(childCtx, 80)
			case "q", "qu", "qui", "quit":
				chCLIStr <- "[quit]\n"
				return
			case "e", "ex", "exi", "exit":
				chCLIStr <- "[exit]\n"
				return
			case "?", "h", "he", "hel", "help":
				str := ""
				str += "start  : start pinger\n"
				str += "stop   : stop pinger\n"
				str += "\n"
				str += "list   : show pinger list\n"
				str += "info   : show pinger info\n"
				str += "result : show ping result\n"
				str += "count  : show ping statistics\n"
				str += "\n"
				str += "quit   : exit client\n"
				str += "exit   : exit client\n"
				str += "\n"
				str += "help   : (this) show help\n"
				chCLIStr <- str
			case "":
				logger.Log(labelinglog.FlgDebug, "input empty")
			default:
				str := ""
				str += "unknown command \"" + command + "\"\n"
				str += "? : show commands\n"
				chCLIStr <- str
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
	fmt.Println("\x1b[49m\x1b[39m\x1b[0m" + "bye")
}
