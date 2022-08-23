// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package cmd

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Notch-Technologies/dotshake/daemon"
	dd "github.com/Notch-Technologies/dotshake/daemon/dotshaker"
	"github.com/Notch-Technologies/dotshake/dotlog"
	"github.com/Notch-Technologies/dotshake/paths"
	"github.com/Notch-Technologies/dotshake/rcn"
	"github.com/Notch-Technologies/dotshake/types/flagtype"
	"github.com/peterbourgon/ff/v2/ffcli"
)

var upArgs struct {
	clientPath string
	signalHost string
	signalPort int64
	serverHost string
	serverPort int64
	logFile    string
	logLevel   string
	debug      bool
	daemon     bool
}

var upCmd = &ffcli.Command{
	Name:       "up",
	ShortUsage: "up [flags]",
	ShortHelp:  "command to start dotshaker",
	FlagSet: (func() *flag.FlagSet {
		fs := flag.NewFlagSet("up", flag.ExitOnError)
		fs.StringVar(&upArgs.clientPath, "path", paths.DefaultClientConfigFile(), "client default config file")
		fs.StringVar(&upArgs.signalHost, "signal-host", "https://signal.dotshake.com", "signaling server host url")
		fs.Int64Var(&upArgs.signalPort, "signal-port", flagtype.DefaultSignalingServerPort, "signaling server host port")
		fs.StringVar(&upArgs.serverHost, "server-host", "https://ctl.dotshake.com", "grpc server host url")
		fs.Int64Var(&upArgs.serverPort, "server-port", flagtype.DefaultServerPort, "grpc server host port")
		fs.StringVar(&upArgs.logFile, "logfile", paths.DefaultDotShakerLogFile(), "set logfile path")
		fs.StringVar(&upArgs.logLevel, "loglevel", dotlog.DebugLevelStr, "set log level")
		fs.BoolVar(&upArgs.debug, "debug", false, "is debug")
		fs.BoolVar(&upArgs.daemon, "daemon", true, "whether to run the daemon process")
		return fs
	})(),
	Exec: execUp,
}

func execUp(ctx context.Context, args []string) error {
	err := dotlog.InitDotLog(upArgs.logLevel, upArgs.logFile, upArgs.debug)
	if err != nil {
		log.Fatalf("failed to initialize logger. because %v", err)
	}

	dotlog := dotlog.NewDotLog("dotshaker up")

	clientCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	signalClient, serverClient, clientConf, mPubKey := initializeDotShakerConf(clientCtx, upArgs.clientPath, upArgs.debug, upArgs.serverHost, uint(upArgs.serverPort), upArgs.signalHost, uint(upArgs.signalPort), dotlog)

	ch := make(chan struct{})

	r := rcn.NewRcn(signalClient, serverClient, clientConf, mPubKey, ch, dotlog)

	if upArgs.daemon {
		dotlog.Logger.Debugf("launching dotshaker daemon...\n")
		d := daemon.NewDaemon(dd.BinPath, dd.ServiceName, dd.DaemonFilePath, dd.SystemConfig, dotlog)
		err = d.Install()
		if err != nil {
			dotlog.Logger.Errorf("failed to install dotshaker. %v", err)
			return err
		}
		dotlog.Logger.Debugf("start dotshaker daemon.\n")
		return nil
	}

	go r.Start()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c,
			os.Interrupt,
			syscall.SIGTERM,
			syscall.SIGINT,
		)
		select {
		case <-c:
			close(ch)
		case <-ctx.Done():
			close(ch)
		}
	}()
	<-ch

	r.Close()

	return nil
}
