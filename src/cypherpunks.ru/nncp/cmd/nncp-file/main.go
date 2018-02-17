/*
NNCP -- Node to Node copy, utilities for store-and-forward data exchange
Copyright (C) 2016-2018 Sergey Matveev <stargrave@stargrave.org>

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

// Send file via NNCP
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"cypherpunks.ru/nncp"
)

func usage() {
	fmt.Fprintf(os.Stderr, nncp.UsageHeader())
	fmt.Fprintf(os.Stderr, "nncp-file -- send file\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s [options] SRC NODE:[DST]\nOptions:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprint(os.Stderr, `
If SRC equals to -, then read data from stdin to temporary file.
`)
}

func main() {
	var (
		cfgPath     = flag.String("cfg", nncp.DefaultCfgPath, "Path to configuration file")
		niceRaw     = flag.Int("nice", nncp.DefaultNiceFile, "Outbound packet niceness")
		minSize     = flag.Uint64("minsize", 0, "Minimal required resulting packet size, in KiB")
		chunkSize   = flag.Uint64("chunked", 0, "Split file on specified size chunks, in KiB")
		viaOverride = flag.String("via", "", "Override Via path to destination node")
		spoolPath   = flag.String("spool", "", "Override path to spool")
		logPath     = flag.String("log", "", "Override path to logfile")
		quiet       = flag.Bool("quiet", false, "Print only errors")
		debug       = flag.Bool("debug", false, "Print debug messages")
		version     = flag.Bool("version", false, "Print version information")
		warranty    = flag.Bool("warranty", false, "Print warranty information")
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
	if flag.NArg() != 2 {
		usage()
		os.Exit(1)
	}
	if *niceRaw < 1 || *niceRaw > 255 {
		log.Fatalln("-nice must be between 1 and 255")
	}
	nice := uint8(*niceRaw)

	ctx, err := nncp.CtxFromCmdline(*cfgPath, *spoolPath, *logPath, *quiet, *debug)
	if err != nil {
		log.Fatalln("Error during initialization:", err)
	}
	if ctx.Self == nil {
		log.Fatalln("Config lacks private keys")
	}

	splitted := strings.SplitN(flag.Arg(1), ":", 2)
	if len(splitted) != 2 {
		usage()
		os.Exit(1)
	}
	node, err := ctx.FindNode(splitted[0])
	if err != nil {
		log.Fatalln("Invalid NODE specified:", err)
	}

	if *viaOverride != "" {
		vias := make([]*nncp.NodeId, 0, strings.Count(*viaOverride, ",")+1)
		for _, via := range strings.Split(*viaOverride, ",") {
			foundNodeId, err := ctx.FindNode(via)
			if err != nil {
				log.Fatalln("Invalid Via node specified:", err)
			}
			vias = append(vias, foundNodeId.Id)
		}
		node.Via = vias
	}

	if *chunkSize == 0 {
		err = ctx.TxFile(
			node,
			nice,
			flag.Arg(0),
			splitted[1],
			int64(*minSize)*1024,
		)
	} else {
		err = ctx.TxFileChunked(
			node,
			nice,
			flag.Arg(0),
			splitted[1],
			int64(*minSize)*1024,
			int64(*chunkSize)*1024,
		)
	}
	if err != nil {
		log.Fatalln(err)
	}
}
