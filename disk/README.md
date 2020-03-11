I was asked to run a disk-only test as well to see how concurrency/parallelism helps with only the disk portion.

## Results

It seems for disk-only writing, a concurrency of 2 (which could be async IO or threads) is sufficient to saturate the disk.

```
concurrency 1, use-random-data true
completed a write of 171520000000 bytes after 305 seconds for 562 MiB/s.
completed a write of 171520000000 bytes after 301 seconds for 569 MiB/s.

concurrency 2, use-random-data true
completed a write of 171520000000 bytes after 195 seconds for 879 MiB/s.
completed a write of 171520000000 bytes after 194 seconds for 884 MiB/s.
completed a write of 171520000000 bytes after 185 seconds for 927 MiB/s.

concurrency 8, use-random-data true
completed a write of 171520000000 bytes after 183 seconds for 937 MiB/s.

concurrency 32, use-random-data true
completed a write of 171520000000 bytes after 190 seconds for 902 MiB/s.
completed a write of 171520000000 bytes after 197 seconds for 870 MiB/s.
completed a write of 171520000000 bytes after 192 seconds for 893 MiB/s.

concurrency 256, use-random-data true
completed a write of 171520000000 bytes after 204 seconds for 840 MiB/s.
```

## Empty Buffer

I did run tests without using random data. They are significantly faster than the actual underlying disk can support. Since changing concurrency does not change this behavior, I think this just serves as proof that this is not a valid way to test - there is some caching, compression, or something else going on.

```
concurrency 1, use-random-data false
completed a write of 171520000000 bytes after 135 seconds for 1270 MiB/s.

concurrency 2, use-random-data false
completed a write of 171520000000 bytes after 118 seconds for 1453 MiB/s.

concurrency 32, use-random-data false
completed a write of 171520000000 bytes after 113 seconds for 1517 MiB/s.
completed a write of 171520000000 bytes after 126 seconds for 1361 MiB/s.

concurrency 256, use-random-data false
completed a write of 171520000000 bytes after 124 seconds for 1383 MiB/s.
```
