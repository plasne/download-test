package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// globals
var storageAccount, storageKey, containerName string

func generateSignature(method, path string, headers map[string]string) string {
	// NOTE: this only supports methods without a body for now
	// NOTE: path must begin with / if not empty

	// pull out all x-ms- headers so they can be sorted and used in the signature
	xheaders := make([]string, 0, len(headers))
	for key, value := range headers {
		key := strings.ToLower(key)
		if strings.HasPrefix(key, "x-ms-") {
			xheaders = append(xheaders, key+":"+value)
		}
	}
	sort.Strings(xheaders)

	// get Content-Type and If-None-Match
	ct := headers["Content-Type"]
	none := headers["If-None-Match"]

	// generate the signature line
	var raw strings.Builder
	raw.WriteString(method)
	raw.WriteString("\n\n\n")
	raw.WriteString("") // len, but empty rather than 0
	raw.WriteString("\n\n")
	raw.WriteString(ct)
	raw.WriteString("\n\n\n\n")
	raw.WriteString(none)
	raw.WriteString("\n\n\n")
	if len(xheaders) > 0 {
		for _, header := range xheaders {
			raw.WriteString(header)
			raw.WriteString("\n")
		}
	} else {
		raw.WriteString("\n")
	}
	raw.WriteString("/")
	raw.WriteString(storageAccount)
	raw.WriteString("/")
	raw.WriteString(containerName)
	raw.WriteString(path)

	// sign the signature line
	key, err := base64.StdEncoding.DecodeString(storageKey)
	if err != nil {
		log.Fatalln("base64.StdEncoding.DecodeString: ", err)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(raw.String()))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return "SharedKey " + storageAccount + ":" + sig
}

func getSize(in string) int64 {
	log.Println("determine the file size...")

	// define the headers
	headers := make(map[string]string)
	headers["x-ms-version"] = "2019-07-07"
	utc := time.Now().UTC().Format(time.RFC1123)
	gmt := strings.Replace(utc, "UTC", "GMT", -1)
	headers["x-ms-date"] = gmt

	// fetch the data from blob
	client := &http.Client{}
	req, err := http.NewRequest("HEAD", "https://"+storageAccount+".blob.core.windows.net/"+containerName+in, nil)
	if err != nil {
		log.Fatalln("http.NewRequest: ", err)
	}
	for key, val := range headers {
		req.Header.Add(key, val)
	}
	sig := generateSignature("HEAD", in, headers)
	req.Header.Add("Authorization", sig)
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln("client.Do: ", err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		log.Fatalln("client.Do: ", res.StatusCode, res.Status)
	}

	// parse the header
	len, err := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		log.Fatalln("ParseInt: ", err)
	}
	log.Printf("file size determined to be %d bytes.", len)

	return len
}

func download(in string, out *os.File, off, len int64) int64 {
	log.Printf("read from %d to %d...\n", off, off+len-1)

	// define the headers
	headers := make(map[string]string)
	headers["x-ms-version"] = "2019-07-07"
	utc := time.Now().UTC().Format(time.RFC1123)
	gmt := strings.Replace(utc, "UTC", "GMT", -1)
	headers["x-ms-date"] = gmt
	headers["x-ms-range"] = "bytes=" + strconv.FormatInt(off, 10) + "-" + strconv.FormatInt(off+len-1, 10)

	// fetch the data from blob
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://"+storageAccount+".blob.core.windows.net/"+containerName+in, nil)
	if err != nil {
		log.Fatalln("http.NewRequest: ", err)
	}
	for key, val := range headers {
		req.Header.Add(key, val)
	}
	sig := generateSignature("GET", in, headers)
	req.Header.Add("Authorization", sig)
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln("client.Do: ", err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		log.Fatalln("client.Do: ", res.StatusCode, res.Status)
	}

	// fill a buffer
	buffer := bytes.NewBuffer(make([]byte, 0, res.ContentLength))
	n, err := buffer.ReadFrom(res.Body)
	if err != nil {
		log.Fatalln("ReadFrom: ", err)
	}
	log.Printf("completed read from %d to %d.\n", off, off+len-1)

	// write to the file position
	// NOTE: WriteAt is safe for parallelism https://golang.org/pkg/io/#WriterAt
	_, err = out.WriteAt(buffer.Bytes(), off)
	if err != nil {
		log.Fatalln("WriteAt: ", err)
	}
	log.Printf("wrote at %d.\n", off)

	return n
}

func main() {

	// load env
	godotenv.Load() // use of .env is optional
	storageAccount = os.Getenv("STORAGE_ACCOUNT")
	storageKey = os.Getenv("STORAGE_KEY")
	containerName = os.Getenv("STORAGE_CONTAINER")
	if storageAccount == "" || storageKey == "" || containerName == "" {
		log.Fatalln("You must specify 'STORAGE_ACCOUNT', 'STORAGE_KEY', and 'STORAGE_CONTAINER' as environment variables.")
	}

	// read args
	var in, out string
	flag.StringVar(&in, "in", "", "specify the URL to pull data from")
	flag.StringVar(&out, "out", "", "specify the name of the local file to write to")
	var concurrency int
	flag.IntVar(&concurrency, "concurrency", 384, "specify how many reads to run at a time")
	var blockSize int64
	flag.Int64Var(&blockSize, "block-size", 12800000, "specify the number of bytes to fetch in each block") // 12.8 MiB
	flag.Parse()
	if in == "" || out == "" {
		log.Fatalln("You must specify both 'in' and 'out' as command line parameters.")
	}

	// determine the size of file
	fileLen := getSize(in)

	// open the output file
	outfile, err := os.Create(out)
	if err != nil {
		log.Fatalln("Create: ", err)
	}
	defer outfile.Close()

	// start the timer
	start := time.Now()

	// downloading, enforcing concurrency
	var offset, total int64 = 0, 0
	sem := make(chan bool, concurrency)
	for offset <= fileLen {
		sem <- true
		go func(off int64) {
			defer func() { <-sem }()
			total += download(in, outfile, off, blockSize)
			complete := int(math.Round(float64(total) / float64(fileLen) * 100))
			log.Printf("%d percent complete...\n", complete)
		}(offset)
		offset += blockSize
	}
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	// log the completion
	log.Println("100 percent complete.")
	elapsed := int64(math.Round(time.Since(start).Seconds()))
	log.Printf("completed a write of %d bytes after %d seconds for %d MiB/s.\n", fileLen, elapsed, fileLen/1000/1000/elapsed)

}
