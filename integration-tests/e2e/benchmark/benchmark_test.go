//go:build benchmark

package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/minimove/integration-tests/e2e/cluster"
)

const (
	clusterReadyTimeout = 120 * time.Second
	mempoolDrainTimeout = 180 * time.Second
	mempoolPollInterval = 500 * time.Millisecond
	warmupSettleTime    = 5 * time.Second
)

func resultsDir(t *testing.T) string {
	t.Helper()
	if d := os.Getenv("BENCHMARK_RESULTS_DIR"); d != "" {
		return d
	}
	return filepath.Join("results")
}

func setupCluster(t *testing.T, ctx context.Context, cfg BenchConfig) *cluster.Cluster {
	t.Helper()

	cl, err := cluster.NewCluster(ctx, t, cluster.ClusterOptions{
		NodeCount:      cfg.NodeCount,
		AccountCount:   cfg.AccountCount,
		ChainID:        "bench-minimove",
		BinaryPath:     os.Getenv("E2E_MINITIAD_BIN"),
		MemIAVL:        cfg.MemIAVL,
		ValidatorCount: cfg.ValidatorCount,
		MaxBlockGas:    cfg.MaxBlockGas,
		NoAllowQueued:  cfg.NoAllowQueued,
	})
	require.NoError(t, err)

	require.NoError(t, cl.Start(ctx))
	t.Cleanup(cl.Close)
	require.NoError(t, cl.WaitForReady(ctx, clusterReadyTimeout))

	return cl
}

func runBenchmarkWithCluster(t *testing.T, ctx context.Context, cl *cluster.Cluster, cfg BenchConfig, loadFn func(ctx context.Context, cl *cluster.Cluster, cfg BenchConfig, metas map[string]cluster.AccountMeta) LoadResult) BenchResult {
	t.Helper()

	metas, err := CollectInitialMetas(ctx, cl)
	require.NoError(t, err)

	Warmup(ctx, cl, metas)
	require.NoError(t, cl.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(warmupSettleTime)

	metas, err = CollectInitialMetas(ctx, cl)
	require.NoError(t, err)

	startHeight, err := cl.LatestHeight(ctx, 0)
	require.NoError(t, err)

	poller := NewMempoolPoller(ctx, cl, mempoolPollInterval)

	t.Logf("Starting load: %d accounts x %d txs = %d total", cfg.AccountCount, cfg.TxPerAccount, cfg.TotalTx())
	loadResult := loadFn(ctx, cl, cfg, metas)
	t.Logf("Load complete: %d submitted, %d errors, duration=%.1fs",
		len(loadResult.Submissions), len(loadResult.Errors),
		loadResult.EndTime.Sub(loadResult.StartTime).Seconds())

	drainTimeout := mempoolDrainTimeout + time.Duration(cfg.TotalTx()/20)*time.Second
	endHeight, err := WaitForLoadToSettle(ctx, cl, drainTimeout, cfg.NoAllowQueued)
	require.NoError(t, err)

	peakMempool := poller.Stop()

	result, err := CollectResults(ctx, cl, cfg, loadResult, startHeight, endHeight, peakMempool)
	require.NoError(t, err)

	t.Logf("Results: TPS=%.1f, P50=%.0fms, P95=%.0fms, P99=%.0fms, included=%d/%d, peak_mempool=%d",
		result.TxPerSecond, result.P50LatencyMs, result.P95LatencyMs, result.P99LatencyMs,
		result.TotalIncluded, result.TotalSubmitted, result.PeakMempoolSize)

	require.NoError(t, WriteResult(t, result, resultsDir(t)))

	return result
}

func runBenchmark(t *testing.T, cfg BenchConfig, loadFn func(ctx context.Context, cl *cluster.Cluster, cfg BenchConfig, metas map[string]cluster.AccountMeta) LoadResult) BenchResult {
	t.Helper()
	ctx := context.Background()

	cl := setupCluster(t, ctx, cfg)
	defer cl.Close()

	return runBenchmarkWithCluster(t, ctx, cl, cfg, loadFn)
}

// ---------------------------------------------------------------------------
// Pre-signed HTTP broadcast benchmarks
// ---------------------------------------------------------------------------

func runPreSignedBenchmark(
	t *testing.T, ctx context.Context, cl *cluster.Cluster, cfg BenchConfig,
	preSignFn func(metas map[string]cluster.AccountMeta) []cluster.SignedTx,
	loadFnFactory func([]cluster.SignedTx) func(ctx context.Context, cl *cluster.Cluster, cfg BenchConfig, metas map[string]cluster.AccountMeta) LoadResult,
) BenchResult {
	t.Helper()

	metas, err := CollectInitialMetas(ctx, cl)
	require.NoError(t, err)

	Warmup(ctx, cl, metas)
	require.NoError(t, cl.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(warmupSettleTime)

	metas, err = CollectInitialMetas(ctx, cl)
	require.NoError(t, err)

	signedTxs := preSignFn(metas)

	startHeight, err := cl.LatestHeight(ctx, 0)
	require.NoError(t, err)

	poller := NewMempoolPoller(ctx, cl, mempoolPollInterval)

	t.Logf("Starting load: %d accounts x %d txs = %d total (pre-signed HTTP)", cfg.AccountCount, cfg.TxPerAccount, cfg.TotalTx())
	loadFn := loadFnFactory(signedTxs)
	loadResult := loadFn(ctx, cl, cfg, metas)
	t.Logf("Load complete: %d submitted, %d errors, duration=%.1fs",
		len(loadResult.Submissions), len(loadResult.Errors),
		loadResult.EndTime.Sub(loadResult.StartTime).Seconds())

	drainTimeout := mempoolDrainTimeout + time.Duration(cfg.TotalTx()/20)*time.Second
	endHeight, err := WaitForLoadToSettle(ctx, cl, drainTimeout, cfg.NoAllowQueued)
	if err != nil {
		t.Logf("Warning: mempool drain incomplete: %v (collecting partial results)", err)
		endHeight, _ = cl.LatestHeight(ctx, 0)
	}

	peakMempool := poller.Stop()

	result, err := CollectResults(ctx, cl, cfg, loadResult, startHeight, endHeight, peakMempool)
	require.NoError(t, err)

	t.Logf("Results: TPS=%.1f, P50=%.0fms, P95=%.0fms, P99=%.0fms, included=%d/%d, peak_mempool=%d",
		result.TxPerSecond, result.P50LatencyMs, result.P95LatencyMs, result.P99LatencyMs,
		result.TotalIncluded, result.TotalSubmitted, result.PeakMempoolSize)

	require.NoError(t, WriteResult(t, result, resultsDir(t)))

	return result
}

// ---------------------------------------------------------------------------
// Move exec setup
// ---------------------------------------------------------------------------

type moveExecLoadMode int

const (
	moveExecBurst moveExecLoadMode = iota
	moveExecSequential
)

// setupMoveExecLoad deploys the BenchHeavyState module and estimates gas once.
// Returns a LoadFn closure that uses burst or sequential Move exec depending on the mode.
func setupMoveExecLoad(t *testing.T, ctx context.Context, cl *cluster.Cluster, mode ...moveExecLoadMode) func(ctx context.Context, cl *cluster.Cluster, cfg BenchConfig, metas map[string]cluster.AccountMeta) LoadResult {
	t.Helper()

	const (
		sharedWrites = "5"  // contended writes to global shared state per tx
		localWrites  = "25" // non-contended writes to per-account state per tx
	)

	// 1. get publisher hex address for Move named-address
	publisherName := cl.AccountNames()[0]
	publisherHex, err := cl.AccountAddressHex(ctx, publisherName)
	require.NoError(t, err)
	t.Logf("Publisher hex address: %s", publisherHex)

	// 2. build BenchHeavyState module with Publisher = acc1's address
	packagePath := cl.RepoPath("integration-tests", "e2e", "benchmark", "move-bench")
	modulePath, err := cl.BuildMoveModule(ctx,
		packagePath, "BenchHeavyState",
		map[string]string{"Publisher": publisherHex})
	require.NoError(t, err)
	t.Logf("Built BenchHeavyState module: %s", modulePath)

	// 3. publish via acc1
	res := cl.MovePublish(ctx, publisherName, []string{modulePath}, 0)
	require.NoError(t, res.Err)
	require.Equal(t, int64(0), res.Code, "publish failed: %s", res.RawLog)

	// 4. wait for inclusion
	require.NoError(t, cl.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(3 * time.Second)

	// 5. estimate gas once for write_mixed(shared_count, local_count)
	args := []string{sharedWrites, localWrites}
	publisherAddr, err := cl.AccountAddress(publisherName)
	require.NoError(t, err)
	meta, err := cl.QueryAccountMeta(ctx, 0, publisherAddr)
	require.NoError(t, err)

	estimatedGas, err := cl.MoveEstimateExecuteJSONGasWithSequence(
		ctx,
		publisherName,
		publisherHex,
		"BenchHeavyState",
		"write_mixed",
		nil, args,
		meta.AccountNumber, meta.Sequence,
		0,
	)
	require.NoError(t, err)
	t.Logf("Estimated gas for BenchHeavyState::write_mixed(%s shared, %s local): %d", sharedWrites, localWrites, estimatedGas)

	// 6. return load function with captured parameters
	m := moveExecBurst
	if len(mode) > 0 {
		m = mode[0]
	}
	if m == moveExecSequential {
		return MoveExecSequentialLoad(publisherHex, "BenchHeavyState", "write_mixed", nil, args, estimatedGas)
	}

	return MoveExecBurstLoad(publisherHex, "BenchHeavyState", "write_mixed", nil, args, estimatedGas)
}

// setupMoveExecCluster deploys BenchHeavyState and estimates gas. Returns the args needed
// to pre-sign Move exec transactions.
func setupMoveExecCluster(t *testing.T, ctx context.Context, cl *cluster.Cluster, writeArgs ...string) (publisherHex string, args []string, estimatedGas uint64) {
	t.Helper()

	sharedWrites, localWrites := "5", "25"
	if len(writeArgs) == 2 {
		sharedWrites, localWrites = writeArgs[0], writeArgs[1]
	}

	publisherName := cl.AccountNames()[0]
	var err error
	publisherHex, err = cl.AccountAddressHex(ctx, publisherName)
	require.NoError(t, err)

	packagePath := cl.RepoPath("integration-tests", "e2e", "benchmark", "move-bench")
	modulePath, err := cl.BuildMoveModule(ctx, packagePath, "BenchHeavyState",
		map[string]string{"Publisher": publisherHex})
	require.NoError(t, err)

	pubRes := cl.MovePublish(ctx, publisherName, []string{modulePath}, 0)
	require.NoError(t, pubRes.Err)
	require.Equal(t, int64(0), pubRes.Code, "publish failed: %s", pubRes.RawLog)
	require.NoError(t, cl.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(3 * time.Second)

	args = []string{sharedWrites, localWrites}
	publisherAddr, err := cl.AccountAddress(publisherName)
	require.NoError(t, err)
	meta, err := cl.QueryAccountMeta(ctx, 0, publisherAddr)
	require.NoError(t, err)
	estimatedGas, err = cl.MoveEstimateExecuteJSONGasWithSequence(
		ctx, publisherName, publisherHex, "BenchHeavyState", "write_mixed",
		nil, args, meta.AccountNumber, meta.Sequence, 0)
	require.NoError(t, err)

	estimatedGas = estimatedGas * 3 / 2
	t.Logf("Estimated gas for write_mixed(%s shared, %s local): %d (with 1.5x adjustment)", sharedWrites, localWrites, estimatedGas)

	return publisherHex, args, estimatedGas
}

// ---------------------------------------------------------------------------
// Mempool comparison: CList vs. Proxy+Priority (bank send)
// ---------------------------------------------------------------------------

func TestBenchmarkBaselineSeq(t *testing.T) {
	cfg := BaselineConfig()
	cfg.Label = "clist/iavl/seq"
	runBenchmark(t, cfg, SequentialLoad)
}

func TestBenchmarkBaselineBurst(t *testing.T) {
	cfg := BaselineConfig()
	cfg.Label = "clist/iavl/burst"
	runBenchmark(t, cfg, BurstLoad)
}

func TestBenchmarkSeqComparison(t *testing.T) {
	var results []BenchResult

	baselines := LoadBaselineResultsByLabel(resultsDir(t), "clist/iavl/seq")
	if len(baselines) > 0 {
		t.Logf("Loaded baseline result: %s", baselines[0].Config.Label)
		results = append(results, baselines[0])
	} else {
		t.Log("No baseline results found. Run TestBenchmarkBaselineSeq with pre-proxy binary for full comparison.")
	}

	t.Run("MempoolOnly", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.Label = "proxy+priority/iavl/seq"
		result := runBenchmark(t, cfg, SequentialLoad)
		results = append(results, result)
	})

	t.Run("Combined", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.Label = "proxy+priority/memiavl/seq"
		result := runBenchmark(t, cfg, SequentialLoad)
		results = append(results, result)
	})

	if len(results) >= 2 {
		PrintComparisonTable(t, results)
		PrintImprovementTable(t, results)
	}
}

func TestBenchmarkBurstComparison(t *testing.T) {
	var results []BenchResult

	baselines := LoadBaselineResultsByLabel(resultsDir(t), "clist/iavl/burst")
	if len(baselines) > 0 {
		t.Logf("Loaded baseline result: %s", baselines[0].Config.Label)
		results = append(results, baselines[0])
	} else {
		t.Log("No baseline results found. Run TestBenchmarkBaselineBurst with pre-proxy binary for full comparison.")
	}

	t.Run("MempoolOnly", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.Label = "proxy+priority/iavl/burst"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	t.Run("Combined", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.Label = "proxy+priority/memiavl/burst"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	if len(results) >= 2 {
		PrintComparisonTable(t, results)
		PrintImprovementTable(t, results)
	}
}

// ---------------------------------------------------------------------------
// Move exec tests
// ---------------------------------------------------------------------------

// TestBenchmarkBaselineSeqMoveExec records CList baseline with sequential Move exec load.
// Build the pre-proxy binary (v1.1.10) and pass via E2E_MINITIAD_BIN.
func TestBenchmarkBaselineSeqMoveExec(t *testing.T) {
	cfg := BaselineConfig()
	cfg.AccountCount = 100
	cfg.TxPerAccount = 50
	cfg.Label = "clist/iavl/seq-move-exec"

	ctx := context.Background()
	cl := setupCluster(t, ctx, cfg)
	defer cl.Close()

	moveLoadFn := setupMoveExecLoad(t, ctx, cl, moveExecSequential)
	runBenchmarkWithCluster(t, ctx, cl, cfg, moveLoadFn)
}

func TestBenchmarkSeqComparisonMoveExec(t *testing.T) {
	var results []BenchResult

	baselines := LoadBaselineResultsByLabel(resultsDir(t), "clist/iavl/seq-move-exec")
	if len(baselines) > 0 {
		t.Logf("Loaded baseline result: %s", baselines[0].Config.Label)
		results = append(results, baselines[0])
	} else {
		t.Log("No baseline results found. Run TestBenchmarkBaselineSeqMoveExec with pre-proxy binary for full comparison.")
	}

	t.Run("MempoolOnly", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
		cfg.Label = "proxy+priority/iavl/seq-move-exec"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cl, moveExecSequential)
		result := runBenchmarkWithCluster(t, ctx, cl, cfg, moveLoadFn)
		results = append(results, result)
	})

	t.Run("Combined", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
		cfg.Label = "proxy+priority/memiavl/seq-move-exec"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cl, moveExecSequential)
		result := runBenchmarkWithCluster(t, ctx, cl, cfg, moveLoadFn)
		results = append(results, result)
	})

	if len(results) >= 2 {
		PrintComparisonTable(t, results)
		PrintImprovementTable(t, results)
	}
}

func TestBenchmarkBurstComparisonMoveExec(t *testing.T) {
	var results []BenchResult

	t.Run("MempoolOnly", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
		cfg.Label = "proxy+priority/iavl/burst-move-exec"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cl)
		result := runBenchmarkWithCluster(t, ctx, cl, cfg, moveLoadFn)
		results = append(results, result)
	})

	t.Run("Combined", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
		cfg.Label = "proxy+priority/memiavl/burst-move-exec"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cl)
		result := runBenchmarkWithCluster(t, ctx, cl, cfg, moveLoadFn)
		results = append(results, result)
	})

	if len(results) >= 2 {
		PrintComparisonTable(t, results)
	}
}

// ---------------------------------------------------------------------------
// Pre-signed HTTP broadcast benchmarks (bank send)
// ---------------------------------------------------------------------------

func TestBenchmarkPreSignedSeqComparison(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/iavl/seq"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		result := runPreSignedBenchmark(t, ctx, cl, cfg,
			func(metas map[string]cluster.AccountMeta) []cluster.SignedTx {
				return PreSignBankTxs(ctx, t, cl, cfg, metas)
			}, PreSignedSequentialLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/memiavl/seq"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		result := runPreSignedBenchmark(t, ctx, cl, cfg,
			func(metas map[string]cluster.AccountMeta) []cluster.SignedTx {
				return PreSignBankTxs(ctx, t, cl, cfg, metas)
			}, PreSignedSequentialLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

func TestBenchmarkPreSignedBurstComparison(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/iavl/burst"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		result := runPreSignedBenchmark(t, ctx, cl, cfg,
			func(metas map[string]cluster.AccountMeta) []cluster.SignedTx {
				return PreSignBankTxs(ctx, t, cl, cfg, metas)
			}, PreSignedBurstLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/memiavl/burst"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		result := runPreSignedBenchmark(t, ctx, cl, cfg,
			func(metas map[string]cluster.AccountMeta) []cluster.SignedTx {
				return PreSignBankTxs(ctx, t, cl, cfg, metas)
			}, PreSignedBurstLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// ---------------------------------------------------------------------------
// Pre-signed Move exec benchmarks
// ---------------------------------------------------------------------------

func TestBenchmarkPreSignedSeqMoveExec(t *testing.T) {
	var results []BenchResult

	const (
		sharedWrites = "10"
		localWrites  = "50"
	)

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/iavl/seq-move-exec"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		pubHex, args, gas := setupMoveExecCluster(t, ctx, cl, sharedWrites, localWrites)

		result := runPreSignedBenchmark(t, ctx, cl, cfg,
			func(metas map[string]cluster.AccountMeta) []cluster.SignedTx {
				return PreSignMoveExecTxs(ctx, t, cl, cfg, metas,
					pubHex, "BenchHeavyState", "write_mixed", nil, args, gas)
			}, PreSignedSequentialLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 100
		cfg.Label = "presigned/memiavl/seq-move-exec"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		pubHex, args, gas := setupMoveExecCluster(t, ctx, cl, sharedWrites, localWrites)

		result := runPreSignedBenchmark(t, ctx, cl, cfg,
			func(metas map[string]cluster.AccountMeta) []cluster.SignedTx {
				return PreSignMoveExecTxs(ctx, t, cl, cfg, metas,
					pubHex, "BenchHeavyState", "write_mixed", nil, args, gas)
			}, PreSignedSequentialLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

func TestBenchmarkPreSignedSeqMoveExecStress(t *testing.T) {
	var results []BenchResult

	const (
		sharedWrites = "20"
		localWrites  = "80"
	)

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 200
		cfg.Label = "presigned-stress/iavl/seq-move-exec"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		pubHex, args, gas := setupMoveExecCluster(t, ctx, cl, sharedWrites, localWrites)

		result := runPreSignedBenchmark(t, ctx, cl, cfg,
			func(metas map[string]cluster.AccountMeta) []cluster.SignedTx {
				return PreSignMoveExecTxs(ctx, t, cl, cfg, metas,
					pubHex, "BenchHeavyState", "write_mixed", nil, args, gas)
			}, PreSignedSequentialLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 20
		cfg.TxPerAccount = 200
		cfg.Label = "presigned-stress/memiavl/seq-move-exec"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		pubHex, args, gas := setupMoveExecCluster(t, ctx, cl, sharedWrites, localWrites)

		result := runPreSignedBenchmark(t, ctx, cl, cfg,
			func(metas map[string]cluster.AccountMeta) []cluster.SignedTx {
				return PreSignMoveExecTxs(ctx, t, cl, cfg, metas,
					pubHex, "BenchHeavyState", "write_mixed", nil, args, gas)
			}, PreSignedSequentialLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// ---------------------------------------------------------------------------
// State DB comparison: IAVL vs MemIAVL (bank send)
// ---------------------------------------------------------------------------

func TestBenchmarkMemIAVLBankSend(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 200
		cfg.Label = "memiavl-compare/iavl/bank-send"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 200
		cfg.Label = "memiavl-compare/memiavl/bank-send"
		result := runBenchmark(t, cfg, BurstLoad)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// TestBenchmarkMemIAVLMoveExec compares IAVL vs. MemIAVL with heavy state Move exec workload.
func TestBenchmarkMemIAVLMoveExec(t *testing.T) {
	var results []BenchResult

	t.Run("IAVL", func(t *testing.T) {
		cfg := MempoolOnlyConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
		cfg.Label = "memiavl-compare/iavl/move-exec"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cl)
		result := runBenchmarkWithCluster(t, ctx, cl, cfg, moveLoadFn)
		results = append(results, result)
	})

	t.Run("MemIAVL", func(t *testing.T) {
		cfg := CombinedConfig()
		cfg.AccountCount = 100
		cfg.TxPerAccount = 50
		cfg.Label = "memiavl-compare/memiavl/move-exec"

		ctx := context.Background()
		cl := setupCluster(t, ctx, cfg)
		defer cl.Close()

		moveLoadFn := setupMoveExecLoad(t, ctx, cl)
		result := runBenchmarkWithCluster(t, ctx, cl, cfg, moveLoadFn)
		results = append(results, result)
	})

	if len(results) == 2 {
		PrintComparisonTable(t, results)
	}
}

// ---------------------------------------------------------------------------
// Capability demos
// ---------------------------------------------------------------------------

func TestBenchmarkQueuePromotion(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.TxPerAccount = 50
	cfg.Label = "queue-promotion/mempool-only"
	result := runBenchmark(t, cfg, OutOfOrderLoad)

	require.Equal(t, result.TotalSubmitted, result.TotalIncluded,
		"not all out-of-order transactions were included: submitted=%d included=%d",
		result.TotalSubmitted, result.TotalIncluded)
}

func TestBenchmarkQueuedFlood(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.TxPerAccount = 50
	cfg.Label = "queued-flood/mempool-only"
	result := runBenchmark(t, cfg, QueuedFloodLoad)

	require.Equal(t, result.TotalSubmitted, result.TotalIncluded,
		"not all queued-flood transactions were included: submitted=%d included=%d",
		result.TotalSubmitted, result.TotalIncluded)
}

func TestBenchmarkQueuedGapEviction(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.TxPerAccount = 50
	cfg.Label = "queued-gap-eviction/mempool-only"

	ctx := context.Background()
	cl := setupCluster(t, ctx, cfg)
	defer cl.Close()

	metas, err := CollectInitialMetas(ctx, cl)
	require.NoError(t, err)

	Warmup(ctx, cl, metas)
	require.NoError(t, cl.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(warmupSettleTime)

	metas, err = CollectInitialMetas(ctx, cl)
	require.NoError(t, err)

	// start mempool poller before load to capture peak queued size
	poller := NewMempoolPoller(ctx, cl, mempoolPollInterval)

	loadResult := QueuedGapLoad(ctx, cl, cfg, metas)
	t.Logf("Submitted %d future-nonce txs (no gap fill), %d errors",
		len(loadResult.Submissions), len(loadResult.Errors))

	t.Log("Waiting for gap TTL eviction (60s + 30s buffer)...")
	time.Sleep(90 * time.Second)

	err = cl.WaitForMempoolEmpty(ctx, 30*time.Second)
	peakMempool := poller.Stop()

	t.Logf("Gap eviction test: peak_mempool=%d, mempool_drained=%v",
		peakMempool, err == nil)

	require.NoError(t, err, "mempool should be empty after gap TTL eviction")
	require.Greater(t, peakMempool, 0, "should have observed queued txs in mempool")
}

func TestBenchmarkGossipPropagation(t *testing.T) {
	cfg := MempoolOnlyConfig()
	cfg.AccountCount = 5
	cfg.TxPerAccount = 50
	cfg.Label = "gossip/mempool-only"

	ctx := context.Background()
	cl := setupCluster(t, ctx, cfg)
	defer cl.Close()

	metas, err := CollectInitialMetas(ctx, cl)
	require.NoError(t, err)

	Warmup(ctx, cl, metas)
	require.NoError(t, cl.WaitForMempoolEmpty(ctx, 30*time.Second))
	time.Sleep(warmupSettleTime)

	metas, err = CollectInitialMetas(ctx, cl)
	require.NoError(t, err)

	startHeight, err := cl.LatestHeight(ctx, 0)
	require.NoError(t, err)

	poller := NewMempoolPoller(ctx, cl, mempoolPollInterval)

	loadResult := SingleNodeLoad(ctx, cl, cfg, metas, 0)
	t.Logf("Submitted %d txs to node 0", len(loadResult.Submissions))

	endHeight, err := WaitForLoadToSettle(ctx, cl, mempoolDrainTimeout, false)
	require.NoError(t, err)

	peakMempool := poller.Stop()
	t.Logf("Cluster peak mempool size: %d", peakMempool)

	result, err := CollectResults(ctx, cl, cfg, loadResult, startHeight, endHeight, peakMempool)
	require.NoError(t, err)

	t.Logf("Gossip test: TPS=%.1f, included=%d/%d",
		result.TxPerSecond, result.TotalIncluded, result.TotalSubmitted)
	require.NoError(t, WriteResult(t, result, resultsDir(t)))
}
