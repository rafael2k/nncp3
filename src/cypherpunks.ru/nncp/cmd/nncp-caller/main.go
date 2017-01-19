/*
NNCP -- Node to Node copy, utilities for store-and-forward data exchange
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

// Croned NNCP TCP daemon caller
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"cypherpunks.ru/nncp"
)

func usage() {
	fmt.Fprintf(os.Stderr, nncp.UsageHeader())
	fmt.Fprintln(os.Stderr, "nncp-caller -- croned NNCP TCP daemon caller\n")
	fmt.Fprintf(os.Stderr, "Usage: %s [options] [NODE ...]\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
}

func main() {
	var (
		cfgPath  = flag.String("cfg", nncp.DefaultCfgPath, "Path to configuration file")
		quiet    = flag.Bool("quiet", false, "Print only errors")
		debug    = flag.Bool("debug", false, "Print debug messages")
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

	cfgRaw, err := ioutil.ReadFile(nncp.CfgPathFromEnv(cfgPath))
	if err != nil {
		log.Fatalln("Can not read config:", err)
	}
	ctx, err := nncp.CfgParse(cfgRaw)
	if err != nil {
		log.Fatalln("Can not parse config:", err)
	}
	if ctx.Self == nil {
		log.Fatalln("Config lacks private keys")
	}
	ctx.Quiet = *quiet
	ctx.Debug = *debug

	var nodes []*nncp.Node
	if flag.NArg() > 0 {
		for _, nodeId := range flag.Args() {
			node, err := ctx.FindNode(nodeId)
			if err != nil {
				log.Fatalln("Invalid NODE specified:", err)
			}
			if len(node.Calls) == 0 {
				ctx.LogD("caller", nncp.SDS{"node": node.Id}, "has no calls, skipping")
				continue
			}
			nodes = append(nodes, node)
		}
	} else {
		for _, node := range ctx.Neigh {
			if len(node.Calls) == 0 {
				ctx.LogD("caller", nncp.SDS{"node": node.Id}, "has no calls, skipping")
				continue
			}
			nodes = append(nodes, node)
		}
	}

	var wg sync.WaitGroup
	for _, node := range nodes {
		for i, call := range node.Calls {
			wg.Add(1)
			go func(node *nncp.Node, i int, call *nncp.Call) {
				defer wg.Done()
				var addrs []string
				if call.Addr == nil {
					for _, addr := range node.Addrs {
						addrs = append(addrs, addr)
					}
				} else {
					addrs = append(addrs, *call.Addr)
				}
				sds := nncp.SDS{"node": node.Id, "callindex": strconv.Itoa(i)}
				for {
					n := time.Now()
					t := call.Cron.Next(n)
					ctx.LogD("caller", sds, t.String())
					if t.IsZero() {
						ctx.LogE("caller", sds, "got zero time")
						return
					}
					time.Sleep(t.Sub(n))
					node.Lock()
					if node.Busy {
						node.Unlock()
						ctx.LogD("caller", sds, "busy")
						continue
					} else {
						node.Busy = true
						node.Unlock()
						ctx.CallNode(
							node,
							addrs,
							call.Nice,
							call.Xx,
							call.OnlineDeadline,
							call.MaxOnlineTime,
						)
						node.Lock()
						node.Busy = false
						node.Unlock()
					}
				}
			}(node, i, call)
		}
	}
	wg.Wait()
}