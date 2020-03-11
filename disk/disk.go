package main

import (
	"flag"
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

func write(out *os.File, off, len int64, useRandomData bool) int64 {

	// fill a buffer with random data (or leave empty)
	buf := make([]byte, len, len)
	if useRandomData {
		rand.Read(buf)
	}

	// write to the file position
	// NOTE: WriteAt is safe for parallelism https://golang.org/pkg/io/#WriterAt
	_, err := out.WriteAt(buf, off)
	if err != nil {
		log.Fatalln("Error", err)
	}
	log.Printf("wrote at %d.\n", off)

	return len
}

func main() {

	// read args
	var out string
	flag.StringVar(&out, "out", "", "specify the name of the local file to write to")
	var concurrency int
	flag.IntVar(&concurrency, "concurrency", 384, "specify how many reads to run at a time")
	var blocksize int64
	flag.Int64Var(&blocksize, "block-size", 12800000, "specify the number of bytes to fetch in each block") // 12.8 MiB
	var numblocks int64
	flag.Int64Var(&numblocks, "num-blocks", 13400, "number of blocks to write")
	var useRandomData bool
	flag.BoolVar(&useRandomData, "use-random-data", true, "if true (which is default) then the buffer will fill with random data")
	flag.Parse()
	if out == "" {
		log.Fatalln("You must specify 'out' as command line parameters.")
	}

	// open the output file
	outfile, err := os.Create(out)
	if err != nil {
		log.Fatalln("Create: ", err)
	}
	defer outfile.Close()

	// start the timer
	start := time.Now()

	// loading, enforcing concurrency
	sem := make(chan bool, concurrency)
	var i int64 = 0
	for i = 0; i < numblocks; i++ {
		sem <- true
		go func(i int64) {
			defer func() { <-sem }()
			off := i * blocksize
			write(outfile, off, blocksize, useRandomData)
			complete := int(math.Round(float64(i) / float64(numblocks) * float64(100)))
			log.Printf("%d percent complete...\n", complete)
		}(i)
	}
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	// log the completion
	log.Println("100 percent complete.")
	elapsed := int64(math.Round(time.Since(start).Seconds()))
	if elapsed > 0 {
		log.Printf("completed a write of %d bytes after %d seconds for %d MiB/s.\n", blocksize*numblocks, elapsed, blocksize*numblocks/1000/1000/elapsed)
	} else {
		log.Printf("completed a write of %d bytes after %d seconds.\n", blocksize*numblocks, elapsed)
	}

}
