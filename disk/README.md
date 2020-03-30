I was asked to run a disk-only test as well to see how concurrency/parallelism helps with only the disk portion.

## Test Configuration

-   Azure N48s_v3 VM (disk: 80k IOPS, 800 MB/s throughput)
-   8x P30s striped (Storage Spaces, Simple, 8 Columns)

```
Get-VirtualDisk data | Format-List

Access                            : Read/Write
AllocatedSize                     : 8787503087616
AllocationUnitSize                : 1073741824
ColumnIsolation                   : PhysicalDisk
DetachedReason                    : None
FaultDomainAwareness              : PhysicalDisk
FootprintOnPool                   : 8787503087616
FriendlyName                      : data
HealthStatus                      : Healthy
Interleave                        : 262144
IsDeduplicationEnabled            : False
IsEnclosureAware                  : False
IsManualAttach                    : False
IsSnapshot                        : False
IsTiered                          : False
LogicalSectorSize                 : 512
MediaType                         : Unspecified
NumberOfColumns                   : 8
NumberOfDataCopies                : 1
NumberOfGroups                    : 1
OperationalStatus                 : OK
PhysicalDiskRedundancy            : 0
PhysicalSectorSize                : 4096
ProvisioningType                  : Fixed
ReadCacheSize                     : 0
RequestNoSinglePointOfFailure     : False
ResiliencySettingName             : Simple
Size                              : 8787503087616
UniqueIdFormat                    : Vendor Specific
Usage                             : Other
WriteCacheSize                    : 0
```

Notice that the Interleave is 256K and the NumberOfColumns is 8, which means a full stripe is 2 MB.

## 12.8 MiB blocks x 13,400 blocks

It seems for disk-only writing, a concurrency of 2 (which could be async IO or threads) is sufficient to saturate the disk.

```
concurrency 1
completed a write of 171520000000 bytes after 305 seconds for 562 MiB/s.
completed a write of 171520000000 bytes after 301 seconds for 569 MiB/s.

concurrency 2
completed a write of 171520000000 bytes after 195 seconds for 879 MiB/s.
completed a write of 171520000000 bytes after 194 seconds for 884 MiB/s.
completed a write of 171520000000 bytes after 185 seconds for 927 MiB/s.

concurrency 8
completed a write of 171520000000 bytes after 183 seconds for 937 MiB/s.

concurrency 32
completed a write of 171520000000 bytes after 190 seconds for 902 MiB/s.
completed a write of 171520000000 bytes after 197 seconds for 870 MiB/s.
completed a write of 171520000000 bytes after 192 seconds for 893 MiB/s.

concurrency 256
completed a write of 171520000000 bytes after 204 seconds for 840 MiB/s.
```

## 1 MB blocks x 163,574 blocks

Even with a smaller block size, a concurrency of 2 (which could be async IO or threads) is sufficient to saturate the disk. I am a little surprised that the performance isn't a bit worse because a 1 MB block is less than the 2 MB stripe size, however, there is probably enough disk queue length to compensate.

```
concurrency 1
completed a write of 171519770624 bytes after 339 seconds for 505 MiB/s.

concurrency 2
completed a write of 171519770624 bytes after 208 seconds for 824 MiB/s.

concurrency 8
completed a write of 171519770624 bytes after 218 seconds for 786 MiB/s.

concurrency 32
completed a write of 171519770624 bytes after 196 seconds for 875 MiB/s.
```

## 256 KB blocks x 654,296 blocks

Smaller block sizes do not perform as well, but a concurrency of 8 did help.

```
concurrency 2
completed a write of 171519770624 bytes after 305 seconds for 562 MiB/s.

concurrency 4
completed a write of 171519770624 bytes after 277 seconds for 619 MiB/s.

concurrency 8
completed a write of 171519770624 bytes after 244 seconds for 702 MiB/s.

concurrency 32
completed a write of 171519770624 bytes after 356 seconds for 481 MiB/s.
```

## Empty Buffer - 12.8 MiB blocks x 13,400 blocks

I did run tests without using random data. They are significantly faster than the actual underlying disk can support. Since changing concurrency does not change this behavior, I think this just serves as proof that this is not a valid way to test - there is some caching, compression, or something else going on.

```
concurrency 1
completed a write of 171520000000 bytes after 135 seconds for 1270 MiB/s.

concurrency 2
completed a write of 171520000000 bytes after 118 seconds for 1453 MiB/s.

concurrency 32
completed a write of 171520000000 bytes after 113 seconds for 1517 MiB/s.
completed a write of 171520000000 bytes after 126 seconds for 1361 MiB/s.

concurrency 256
completed a write of 171520000000 bytes after 124 seconds for 1383 MiB/s.
```
