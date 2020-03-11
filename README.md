# Download-Test

This simple golang application was written to see how fast large seismic files could be transferred from Azure Blob Storage to an Azure VM in the same region without using azcopy. The primary focus of this solution was to test disk performance and show how a large number of threads could write data at the same time.

**IMPORTANT**: You should use azcopy whenever possible. It is much more sophisticated and optimized.

## Performance

It is certainly possible that you could tweak better performance, but for the now, good performance seems to be obtained with:

-   block-size 12800000 (12.8 MB)
-   concurrency 384

## Config A Results

-   Azure D64s_v3 VM (disk: 80k IOPS, 800 MB/s throughput)
-   8x P30s striped (Storage Spaces, Simple, 8 Columns)
-   Accelerated Networking

**Best Time: 665 MiB/s**

```
block-size 12800000, concurrency 384
completed a write of 171507845844 bytes after 262 seconds.
completed a write of 171507845844 bytes after 272 seconds.
completed a write of 171507845844 bytes after 258 seconds.

block-size 12800000, concurrency 256
completed a write of 171507845844 bytes after 277 seconds.
completed a write of 171507845844 bytes after 279 seconds.
completed a write of 171507845844 bytes after 290 seconds.

block-size 12800000, concurrency 320
completed a write of 171507845844 bytes after 284 seconds.

block-size 12800000, concurrency 512
completed a write of 171507845844 bytes after 294 seconds.

block-size 14400000, concurrency 256
completed a write of 171507845844 bytes after 304 seconds.

block-size 11200000, concurrency 256
completed a write of 171507845844 bytes after 306 seconds.

block-size 9600000, concurrency 256
completed a write of 171507845844 bytes after 312 seconds.

block-size 6400000, concurrency 256
completed a write of 171507845844 bytes after 317 seconds

block-size 12800000, concurrency 32
completed a write of 171507845844 bytes after 438 seconds.

block-size 3200000, concurrency 256
completed a write of 171507845844 bytes after 456 seconds.

block-size 19200000, concurrency 256
completed a write of 171507845844 bytes after 1248 seconds.
```

## Config B Results

-   Azure N48s_v3 VM (disk: 80k IOPS, 800 MB/s throughput)
-   8x P30s striped (Storage Spaces, Simple, 8 Columns)

**Best Time: 660 MiB/s**

```
block-size 12800000, concurrency 384
completed a write of 171507845844 bytes after 279 seconds.
completed a write of 171507845844 bytes after 260 seconds.
completed a write of 171507845844 bytes after 274 seconds.

block-size 12800000, concurrency 256
completed a write of 171507845844 bytes after 273 seconds.
completed a write of 171507845844 bytes after 266 seconds.
completed a write of 171507845844 bytes after 268 seconds.
```

## Running a Test

Create a .env file in the folder you will be running the test from. It should contain:

```bash
STORAGE_ACCOUNT=myaccountname
STORAGE_CONTAINER=mycontainername
STORAGE_KEY=e...==
```

Run either the "perf" file (macos), "perf.exe" file (win64), or compile the "perf.go" for your platform. Use the following parameters:

```
./perf --in /largefile.segy --out /Users/plasne/Downloads/largefile.segy --block-size 12800000 --concurrency 256
```

```
perf.exe --in /largefile.segy --out f:\largefile.segy --block-size 12800000 --concurrency 256
```

## Concurrency

The concurrency setting determines the number of goroutines that are executed at the same time. Go abstracts the concept of threads so it could vary as to how many threads are used versus async IO on the existing threads.

## Block-Size

The block-size setting determines the size of the result from the Azure Blob Storage REST API for each download request. In other words, if you have a concurrency of 32 and block-size of 1000000, then you will have 32 requests at a time, each fetching 1 MB of data.

## WriteAt

This sample uses file.WriteAt to put an array of bytes into the sparse file at a specific position. Per the golang documentation, this method is safe for multiple parallel writers: https://golang.org/pkg/io/#WriterAt.

## Performance Improvements

I suspect the largest area for improvement is related to how the response body is handled. A new buffer is created, filled, and then the bytes in the buffer are written to the file position. There may be some better way to pipe this directly to the file location.

```go
// fill a buffer
buffer := bytes.NewBuffer(make([]byte, 0, res.ContentLength))
n, err := buffer.ReadFrom(res.Body)
if err != nil {
    log.Fatalln("ReadFrom: ", err)
}
log.Printf("completed read from %d to %d.\n", off, off+len-1)

_, err = out.WriteAt(buffer.Bytes(), off)
```
