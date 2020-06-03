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
	"regexp"
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
	pb "github.com/umenosuke/ping-grpc-client/proto/pingGrpc"
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
	argDebugFlag             bool
	argServerAddress         string
	argNoUseTLS              bool
	argCACertificatePath     string
	argClientCertificatePath string
	argClientPrivateKeyPath  string
	argConfig                string
	argConfigPath            string
	argNoColor               bool
	argShowConfigFlg         bool
	argShowVersionFlag       bool
)

func init() {
	flag.BoolVar(&argDebugFlag, "debug", false, "print debug log")
	flag.StringVar(&argServerAddress, "server", "127.0.0.1:5555", "server address:port")
	flag.StringVar(&argServerAddress, "s", "127.0.0.1:5555", "server address:port (shorthand)")
	flag.StringVar(&argServerAddress, "S", "127.0.0.1:5555", "server address:port (shorthand)")
	flag.BoolVar(&argNoUseTLS, "noUseTLS", false, "disable tls")
	flag.StringVar(&argCACertificatePath, "caCert", "./ca.crt", "CA certificate file path")
	flag.StringVar(&argClientCertificatePath, "cCert", "./client_pinger.crt", "client certificate file path")
	flag.StringVar(&argClientPrivateKeyPath, "cKey", "./client_pinger.pem", "client private key file path")
	flag.StringVar(&argConfig, "config", "{}", "config json string")
	flag.StringVar(&argConfigPath, "configPath", "", "config file path")
	flag.BoolVar(&argNoColor, "noColor", false, "disable colorful output")
	flag.BoolVar(&argShowConfigFlg, "printConfig", false, "show default config")
	flag.BoolVar(&argShowVersionFlag, "version", false, "show version")
	flag.BoolVar(&argShowVersionFlag, "v", false, "show version (shorthand)")
}

func main() {
	flag.Parse()

	if argDebugFlag {
		logger.SetEnableLevel(labelinglog.FlgsetAll)
	} else {
		logger.SetEnableLevel(labelinglog.FlgsetCommon - labelinglog.FlgNotice)
	}

	subMain()
	os.Exit(exitCode)
}

func subMain() {
	if argShowVersionFlag {
		fmt.Fprint(os.Stdout, "Version "+metaVersion+"\n"+"Revision "+metaRevision+"\n")
		return
	}

	if argShowConfigFlg {
		fmt.Fprint(os.Stdout, configStringify(DefaultConfig())+"\n")
		return
	}

	config, err := configLoad(argConfigPath, argConfig)
	if err != nil {
		logger.Log(labelinglog.FlgFatal, err.Error())
		exitCode = 1
		return
	}
	if argDebugFlag {
		logger.Log(labelinglog.FlgDebug, "now config")
		logger.LogMultiLines(labelinglog.FlgDebug, configStringify(config))
	}

	grpcDialOptions, err := getGrpcDialOptions()
	if err != nil {
		logger.Log(labelinglog.FlgFatal, err.Error())
		exitCode = 1
		return
	}

	conn, err := grpc.Dial(argServerAddress, grpcDialOptions...)
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
					fmt.Println()
					logger.Log(labelinglog.FlgDebug, "request stop, SIGINT")
					chCancel <- struct{}{}
				default:
					logger.Log(labelinglog.FlgWarn, fmt.Sprintf("unknown syscall [%v]", sig))
				}
			}
		}
	})()

	enableColor := !argNoColor && runtime.GOOS != "windows"
	chCLIStr := make(chan tCliMsg, 200)
	wgFinish.Add(1)
	go (func() {
		defer wgFinish.Done()
		for {
			select {
			case <-time.After(time.Second):
			case msg := <-chCLIStr:
				str := ""

				if enableColor {
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

				if enableColor {
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

	var isInteractive = len(flag.Args()) < 1

	wgFinish.Add(1)
	go (func() {
		defer wgFinish.Done()
		defer childCtxCancel()

		client := tClientWrap{
			client:   pb.NewPingerClient(conn),
			chCancel: chCancel,
			chCLIStr: chCLIStr,
			config:   config,
		}

		if isInteractive {
			client.interactive(childCtx)
		} else {
			var subCommand = flag.Args()[0]
			var subCommandArgs = flag.Args()[1:]

			switch subCommand {
			case "s", "st":
				chCLIStr <- tCliMsg{
					text: "" +
						"unknown command \"" + subCommand + "\"\n" +
						"\n" +
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
				if len(subCommandArgs) >= 1 {
					var path = subCommandArgs[0]
					var descStr string
					if len(subCommandArgs) >= 2 {
						descStr = subCommandArgs[1]
					} else {
						descStr = path
					}

					if _, err := os.Stat(path); err != nil {
						logger.Log(labelinglog.FlgError, "target list file not found ["+path+"]")
						chCLIStr <- tCliMsg{
							text:    "not found [" + path + "]",
							color:   cliColorDefault,
							noBreak: false,
						}
						return
					}

					file, err := os.Open(path)
					if err != nil {
						logger.Log(labelinglog.FlgError, err.Error())
						chCLIStr <- tCliMsg{
							text:    "can not open [" + path + "]",
							color:   cliColorDefault,
							noBreak: false,
						}
						return
					}
					defer file.Close()

					targetList := make([]*pb.StartRequest_IcmpTarget, 0)
					reg := regexp.MustCompile(`^([^# \t]*)[# \t]*(.*)$`)
					scanner := bufio.NewScanner(file)
					lineNum := 0
					for scanner.Scan() {
						lineNum++
						line := strings.Trim(scanner.Text(), " \t")
						if line != "" {
							result := reg.FindStringSubmatch(line)
							if result != nil {
								targetIP := result[1]
								targetComment := result[2]

								targetList = append(targetList, &pb.StartRequest_IcmpTarget{
									TargetIP: targetIP,
									Comment:  targetComment,
								})
							} else {
								logger.Log(labelinglog.FlgInfo, fmt.Sprintf("[%s] line %3d skip, comment or format error \"%s\"", path, lineNum, line))
							}
						} else {
							logger.Log(labelinglog.FlgInfo, fmt.Sprintf("[%s] line %3d skip, empty \"%s\"", path, lineNum, line))
						}
					}
					if err := scanner.Err(); err != nil {
						logger.Log(labelinglog.FlgError, err.Error())
						return
					}

					client.start(childCtx, descStr, targetList)
				} else {
					client.chCLIStr <- tCliMsg{
						text:    "Please enter \"target list path\"",
						color:   cliColorDefault,
						noBreak: false,
					}
					return
				}
			case "sto", "stop":
				chCLIStr <- tCliMsg{
					text:    "[stop]",
					color:   cliColorDefault,
					noBreak: false,
				}
				if len(subCommandArgs) >= 1 {
					client.stop(childCtx, subCommandArgs[0])
				} else {
					client.chCLIStr <- tCliMsg{
						text:    "Please enter \"pingerID\"",
						color:   cliColorDefault,
						noBreak: false,
					}
					return
				}
			case "l", "li", "lis", "list":
				if len(subCommandArgs) < 1 {
					client.chCLIStr <- tCliMsg{
						text:    "[list]",
						color:   cliColorDefault,
						noBreak: false,
					}
					client.printListSummary(childCtx)
				} else {
					switch subCommandArgs[0] {
					case "l", "lo", "lon", "long":
						client.chCLIStr <- tCliMsg{
							text:    "[list long]",
							color:   cliColorDefault,
							noBreak: false,
						}
						client.printList(childCtx)
					case "s", "sh", "sho", "shor", "short":
						client.chCLIStr <- tCliMsg{
							text:    "[list short]",
							color:   cliColorDefault,
							noBreak: false,
						}
						client.printListVeryShort(childCtx)
					default:
						client.chCLIStr <- tCliMsg{
							text:    "[list]",
							color:   cliColorDefault,
							noBreak: false,
						}
						client.printListSummary(childCtx)
					}
				}
			case "i", "in", "inf", "info":
				chCLIStr <- tCliMsg{
					text:    "[info]",
					color:   cliColorDefault,
					noBreak: false,
				}
				if len(subCommandArgs) >= 1 {
					client.info(childCtx, subCommandArgs[0])
				} else {
					client.chCLIStr <- tCliMsg{
						text:    "Please enter \"pingerID\"",
						color:   cliColorDefault,
						noBreak: false,
					}
					return
				}
			case "r", "re", "res", "resu", "resul", "result":
				chCLIStr <- tCliMsg{
					text:    "[result]",
					color:   cliColorDefault,
					noBreak: false,
				}
				if len(subCommandArgs) >= 1 {
					client.result(childCtx, subCommandArgs[0])
				} else {
					client.chCLIStr <- tCliMsg{
						text:    "Please enter \"pingerID\"",
						color:   cliColorDefault,
						noBreak: false,
					}
					return
				}
			case "c", "co", "cou", "coun", "count":
				chCLIStr <- tCliMsg{
					text:    "[count]",
					color:   cliColorDefault,
					noBreak: false,
				}
				if len(subCommandArgs) >= 1 {
					client.count(childCtx, subCommandArgs[0])
				} else {
					client.chCLIStr <- tCliMsg{
						text:    "Please enter \"pingerID\"",
						color:   cliColorDefault,
						noBreak: false,
					}
					return
				}
			case "h", "he", "hel", "help":
				chCLIStr <- tCliMsg{
					text: "" +
						"[help]\n" +
						"start \"{target list path}\"                 : start pinger\n" +
						"start \"{target list path}\" \"{description}\" : start pinger\n" +
						"stop \"{pingerID}\"                          : stop pinger\n" +
						"\n" +
						"list       : show pinger list summary\n" +
						"list long  : show pinger list verbose\n" +
						"list short : show pinger id list\n" +
						"\n" +
						"info \"{pingerID}\"     : show pinger info\n" +
						"result \"{pingerID}\"   : show ping result\n" +
						"count \"{pingerID}\"    : show ping statistics\n" +
						"\n" +
						"help     : (this) show help",
					color:   cliColorDefault,
					noBreak: false,
				}
			default:
				chCLIStr <- tCliMsg{
					text: "" +
						"unknown command \"" + subCommand + "\"\n" +
						"\n" +
						"help : show commands",
					color:   cliColorDefault,
					noBreak: false,
				}
				return
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

	if enableColor {
		fmt.Print("\x1b[49m\x1b[39m\x1b[0m")
	}

	if isInteractive {
		fmt.Println("\nbye")
	}
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

	if !argNoUseTLS {
		clientCert, err :=
			tls.LoadX509KeyPair(
				argClientCertificatePath,
				argClientPrivateKeyPath)
		if err != nil {
			return nil, err
		}

		caCert, err := ioutil.ReadFile(argCACertificatePath)
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
