// Copyright 2019 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package engine

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/cockroachdb/cockroach/pkg/base"
	"github.com/cockroachdb/cockroach/pkg/roachpb"
	"github.com/cockroachdb/cockroach/pkg/util/encoding"
	"github.com/cockroachdb/cockroach/pkg/util/hlc"
	"github.com/cockroachdb/cockroach/pkg/util/randutil"
	"github.com/cockroachdb/cockroach/pkg/util/timeutil"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

func testPebbleOptions(fs vfs.FS) *pebble.Options {
	opts := DefaultPebbleOptions()
	opts.Cache = pebble.NewCache(testCacheSize)
	opts.FS = fs
	return opts
}

func setupMVCCPebble(b testing.TB, dir string) Engine {
	peb, err := NewPebble(PebbleConfig{
		StorageConfig: base.StorageConfig{
			Dir: dir,
		},
		Opts: testPebbleOptions(vfs.Default),
	})
	if err != nil {
		b.Fatalf("could not create new pebble instance at %s: %+v", dir, err)
	}
	return peb
}

func setupMVCCInMemPebble(b testing.TB, loc string) Engine {
	peb, err := NewPebble(PebbleConfig{
		Opts: testPebbleOptions(vfs.NewMem()),
	})
	if err != nil {
		b.Fatalf("could not create new in-mem pebble instance: %+v", err)
	}
	return peb
}

func BenchmarkMVCCScan_Pebble(b *testing.B) {
	if testing.Short() {
		b.Skip("TODO: fix benchmark")
	}

	ctx := context.Background()
	for _, numRows := range []int{1, 10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("rows=%d", numRows), func(b *testing.B) {
			for _, numVersions := range []int{1, 2, 10, 100} {
				b.Run(fmt.Sprintf("versions=%d", numVersions), func(b *testing.B) {
					for _, valueSize := range []int{8, 64, 512} {
						b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
							runMVCCScan(ctx, b, setupMVCCPebble, benchScanOptions{
								benchDataOptions: benchDataOptions{
									numVersions: numVersions,
									valueBytes:  valueSize,
								},
								numRows: numRows,
								reverse: false,
							})
						})
					}
				})
			}
		})
	}
}

func BenchmarkMVCCReverseScan_Pebble(b *testing.B) {
	if testing.Short() {
		b.Skip("TODO: fix benchmark")
	}

	ctx := context.Background()
	for _, numRows := range []int{1, 10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("rows=%d", numRows), func(b *testing.B) {
			for _, numVersions := range []int{1, 2, 10, 100} {
				b.Run(fmt.Sprintf("versions=%d", numVersions), func(b *testing.B) {
					for _, valueSize := range []int{8, 64, 512} {
						b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
							runMVCCScan(ctx, b, setupMVCCPebble, benchScanOptions{
								benchDataOptions: benchDataOptions{
									numVersions: numVersions,
									valueBytes:  valueSize,
								},
								numRows: numRows,
								reverse: true,
							})
						})
					}
				})
			}
		})
	}
}

func BenchmarkMVCCGet_Pebble(b *testing.B) {
	ctx := context.Background()
	for _, numVersions := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("versions=%d", numVersions), func(b *testing.B) {
			for _, valueSize := range []int{8} {
				b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
					runMVCCGet(ctx, b, setupMVCCPebble, benchDataOptions{
						numVersions: numVersions,
						valueBytes:  valueSize,
					})
				})
			}
		})
	}
}

func BenchmarkMVCCComputeStats_Pebble(b *testing.B) {
	if testing.Short() {
		b.Skip("short flag")
	}
	ctx := context.Background()
	for _, valueSize := range []int{8, 32, 256} {
		b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
			runMVCCComputeStats(ctx, b, setupMVCCPebble, valueSize)
		})
	}
}

func BenchmarkMVCCFindSplitKey_Pebble(b *testing.B) {
	ctx := context.Background()
	for _, valueSize := range []int{32} {
		b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
			runMVCCFindSplitKey(ctx, b, setupMVCCPebble, valueSize)
		})
	}
}

func BenchmarkMVCCPut_Pebble(b *testing.B) {
	ctx := context.Background()
	for _, valueSize := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
			runMVCCPut(ctx, b, setupMVCCInMemPebble, valueSize)
		})
	}
}

func BenchmarkMVCCBlindPut_Pebble(b *testing.B) {
	ctx := context.Background()
	for _, valueSize := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
			runMVCCBlindPut(ctx, b, setupMVCCInMemPebble, valueSize)
		})
	}
}

func BenchmarkMVCCConditionalPut_Pebble(b *testing.B) {
	ctx := context.Background()
	for _, createFirst := range []bool{false, true} {
		prefix := "Create"
		if createFirst {
			prefix = "Replace"
		}
		b.Run(prefix, func(b *testing.B) {
			for _, valueSize := range []int{10, 100, 1000, 10000} {
				b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
					runMVCCConditionalPut(ctx, b, setupMVCCInMemPebble, valueSize, createFirst)
				})
			}
		})
	}
}

func BenchmarkMVCCBlindConditionalPut_Pebble(b *testing.B) {
	ctx := context.Background()
	for _, valueSize := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
			runMVCCBlindConditionalPut(ctx, b, setupMVCCInMemPebble, valueSize)
		})
	}
}

func BenchmarkMVCCInitPut_Pebble(b *testing.B) {
	ctx := context.Background()
	for _, valueSize := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
			runMVCCInitPut(ctx, b, setupMVCCInMemPebble, valueSize)
		})
	}
}

func BenchmarkMVCCBlindInitPut_Pebble(b *testing.B) {
	ctx := context.Background()
	for _, valueSize := range []int{10, 100, 1000, 10000} {
		b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
			runMVCCBlindInitPut(ctx, b, setupMVCCInMemPebble, valueSize)
		})
	}
}

func BenchmarkMVCCPutDelete_Pebble(b *testing.B) {
	ctx := context.Background()
	db := setupMVCCInMemPebble(b, "put_delete")
	defer db.Close()

	r := rand.New(rand.NewSource(timeutil.Now().UnixNano()))
	value := roachpb.MakeValueFromBytes(randutil.RandBytes(r, 10))
	var blockNum int64

	for i := 0; i < b.N; i++ {
		blockID := r.Int63()
		blockNum++
		key := encoding.EncodeVarintAscending(nil, blockID)
		key = encoding.EncodeVarintAscending(key, blockNum)

		if err := MVCCPut(ctx, db, nil, key, hlc.Timestamp{}, value, nil /* txn */); err != nil {
			b.Fatal(err)
		}
		if err := MVCCDelete(ctx, db, nil, key, hlc.Timestamp{}, nil /* txn */); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMVCCBatchPut_Pebble(b *testing.B) {
	ctx := context.Background()
	for _, valueSize := range []int{10} {
		b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
			for _, batchSize := range []int{1, 100, 10000, 100000} {
				b.Run(fmt.Sprintf("batchSize=%d", batchSize), func(b *testing.B) {
					runMVCCBatchPut(ctx, b, setupMVCCInMemPebble, valueSize, batchSize)
				})
			}
		})
	}
}

func BenchmarkMVCCBatchTimeSeries_Pebble(b *testing.B) {
	ctx := context.Background()
	for _, batchSize := range []int{282} {
		b.Run(fmt.Sprintf("batchSize=%d", batchSize), func(b *testing.B) {
			runMVCCBatchTimeSeries(ctx, b, setupMVCCInMemPebble, batchSize)
		})
	}
}

// BenchmarkMVCCGetMergedTimeSeries computes performance of reading merged
// time series data using `MVCCGet()`. Uses an in-memory engine.
func BenchmarkMVCCGetMergedTimeSeries_Pebble(b *testing.B) {
	if testing.Short() {
		b.Skip("short flag")
	}
	ctx := context.Background()
	for _, numKeys := range []int{1, 16, 256} {
		b.Run(fmt.Sprintf("numKeys=%d", numKeys), func(b *testing.B) {
			for _, mergesPerKey := range []int{1, 16, 256} {
				b.Run(fmt.Sprintf("mergesPerKey=%d", mergesPerKey), func(b *testing.B) {
					runMVCCGetMergedValue(ctx, b, setupMVCCInMemPebble, numKeys, mergesPerKey)
				})
			}
		})
	}
}

func BenchmarkMVCCGarbageCollect_Pebble(b *testing.B) {
	if testing.Short() {
		b.Skip("short flag")
	}

	// NB: To debug #16068, test only 128-128-15000-6.
	ctx := context.Background()
	for _, keySize := range []int{128} {
		b.Run(fmt.Sprintf("keySize=%d", keySize), func(b *testing.B) {
			for _, valSize := range []int{128} {
				b.Run(fmt.Sprintf("valSize=%d", valSize), func(b *testing.B) {
					for _, numKeys := range []int{1, 1024} {
						b.Run(fmt.Sprintf("numKeys=%d", numKeys), func(b *testing.B) {
							for _, numVersions := range []int{2, 1024} {
								b.Run(fmt.Sprintf("numVersions=%d", numVersions), func(b *testing.B) {
									runMVCCGarbageCollect(ctx, b, setupMVCCInMemPebble, benchGarbageCollectOptions{
										benchDataOptions: benchDataOptions{
											numKeys:     numKeys,
											numVersions: numVersions,
											valueBytes:  valSize,
										},
										keyBytes:       keySize,
										deleteVersions: numVersions - 1,
									})
								})
							}
						})
					}
				})
			}
		})
	}
}

func BenchmarkBatchApplyBatchRepr_Pebble(b *testing.B) {
	if testing.Short() {
		b.Skip("short flag")
	}
	ctx := context.Background()
	for _, indexed := range []bool{false, true} {
		b.Run(fmt.Sprintf("indexed=%t", indexed), func(b *testing.B) {
			for _, sequential := range []bool{false, true} {
				b.Run(fmt.Sprintf("seq=%t", sequential), func(b *testing.B) {
					for _, valueSize := range []int{10} {
						b.Run(fmt.Sprintf("valueSize=%d", valueSize), func(b *testing.B) {
							for _, batchSize := range []int{10000} {
								b.Run(fmt.Sprintf("batchSize=%d", batchSize), func(b *testing.B) {
									runBatchApplyBatchRepr(ctx, b, setupMVCCInMemPebble,
										indexed, sequential, valueSize, batchSize)
								})
							}
						})
					}
				})
			}
		})
	}
}
