// Comboserver serves multiple files combined in a single request.
//
//   Usage: comboserver <directory>
//     -addr="localhost:8080": address to serve content from
//     -maxfiles=50: maximum files to concatenate in a single request
//     -root="/": root URL
//     -sep="&": file list separator
// 
// Example:
//
//  $ comboserver -addr=localhost:8081 -root=/combo /var/www
//
// This will launch a web server serving files from /var/www. Suppose it
// contains the following files:
//
//     base.css
//     pure/grids.css
//     pure/buttons.css
//     
// Then these files, concatenated, are available with a single request:
//
//     http://localhost:8081/combo?base.css&pure/grids.css&pure/buttons.css
//
// Files can be combined in any way:
//
//     http://localhost:8081/combo?pure/buttons.css&base.css
//
// The program preserves the order of request, doesn't allow repeated
// filenames, and limits the number of files to 50 by default. Content-type
// header of response is set to the type of the first file in list.
//
// Comboserver doesn't do any caching, as it's a job of whatever nginx you put
// in front of it.
//
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dchest/comboserver/combo"
)

var (
	fAddr     = flag.String("addr", "localhost:8080", "address to serve content from")
	fRoot     = flag.String("root", "/", "root URL")
	fSep      = flag.String("sep", "&", "file list separator")
	fMaxFiles = flag.Int("maxfiles", 50, "maximum files to concatenate in a single request")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <directory>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	http.Handle(*fRoot, &combo.Handler{
		Root:      http.Dir(flag.Arg(0)),
		URLPath:   *fRoot,
		Separator: *fSep,
		MaxFiles:  *fMaxFiles,
	})

	fmt.Printf("serving combo files from %s at %s\n", flag.Arg(0), *fAddr)
	log.Fatal(http.ListenAndServe(*fAddr, nil))
}
