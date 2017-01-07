/*
NNCP -- Node-to-Node CoPy
Copyright (C) 2016-2017 Sergey Matveev <stargrave@stargrave.org>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

// NNCP TCP daemon
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"

	"cypherpunks.ru/nncp"
	"golang.org/x/net/netutil"
)

func usage() {
	fmt.Fprintf(os.Stderr, nncp.UsageHeader())
	fmt.Fprintln(os.Stderr, "nncp-daemon -- TCP daemon\n")
	fmt.Fprintln(os.Stderr, "Usage: %s [options]\nOptions:", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var (
		cfgPath  = flag.String("cfg", nncp.DefaultCfgPath, "Path to configuration file")
		niceRaw  = flag.Int("nice", 255, "Minimal required niceness")
		bind     = flag.String("bind", ":5400", "Address to bind to")
		maxConn  = flag.Int("maxconn", 128, "Maximal number of simultaneous connections")
		debug    = flag.Bool("debug", false, "Enable debugging information")
		version  = flag.Bool("version", false, "Print version information")
		warranty = flag.Bool("warranty", false, "Print warranty information")
	)
	flag.Usage = usage
	flag.Parse()
	if *warranty {
		fmt.Println(nncp.Warranty)
		return
	}
	if *version {
		fmt.Println(nncp.VersionGet())
		return
	}
	if *niceRaw < 1 || *niceRaw > 255 {
		log.Fatalln("-nice must be between 1 and 255")
	}
	nice := uint8(*niceRaw)

	cfgRaw, err := ioutil.ReadFile(*cfgPath)
	if err != nil {
		log.Fatalln("Can not read config:", err)
	}
	ctx, err := nncp.CfgParse(cfgRaw)
	if err != nil {
		log.Fatalln("Can not parse config:", err)
	}
	ctx.Debug = *debug

	ln, err := net.Listen("tcp", *bind)
	if err != nil {
		log.Fatalln("Can not listen:", err)
	}
	ln = netutil.LimitListener(ln, *maxConn)
	ctx.LogI("daemon", nncp.SDS{"bind": *bind}, "listening")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalln("Can not accept connection:", err)
		}
		ctx.LogD("daemon", nncp.SDS{"addr": conn.RemoteAddr()}, "accepted")
		go func(conn net.Conn) {
			state, err := ctx.StartR(conn, nice, nil)
			if err == nil {
				ctx.LogI("call-start", nncp.SDS{"node": state.NodeId}, "connected")
				state.Wait()
				ctx.LogI("call-finish", nncp.SDS{
					"node":     state.NodeId,
					"duration": strconv.FormatInt(int64(state.Duration.Seconds()), 10),
					"rxbytes":  strconv.FormatInt(state.RxBytes, 10),
					"txbytes":  strconv.FormatInt(state.TxBytes, 10),
					"rxspeed":  strconv.FormatInt(state.RxSpeed, 10),
					"txspeed":  strconv.FormatInt(state.TxSpeed, 10),
				}, "")
			} else {
				ctx.LogE("call-start", nncp.SDS{"node": state.NodeId, "err": err}, "")
			}
			conn.Close()
		}(conn)
	}
}