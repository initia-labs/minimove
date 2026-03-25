# E2E Benchmark

Performance benchmark for ProxyMempool + PriorityMempool + MemIAVL on MiniMove. Measures throughput, latency, and mempool behavior
across optimization layers using a multi-node cluster with production-realistic settings.

## Cluster topology

4-node cluster: 1 sequencer + 3 fullnodes on localhost with deterministic port allocation.

- **Fullnode submission**: Benchmark load is submitted to edge (non-validator) nodes (indices 1-3), testing gossip propagation to the sequencer.
- **Block interval**: 100ms (sequencing default, `CreateEmptyBlocks = false` thus blocks only created when txs exist)
- **Gas price**: 0umin
- **Queued tx extension**: All tx submissions include `--allow-queued` flag (required for `ExtensionOptionQueuedTx`)

## Comparison matrix

### 1. Mempool comparison: CList vs Proxy+Priority

Three load patterns: sequential tests give a fair TPS comparison (both mempools handle in-order correctly), burst tests demonstrate CList's tx-drop problem.

**Baselines** (run with v1.1.10 binary):

| Test | Load | Config |
|---|---|---|
| `TestBenchmarkBaselineSeq` | Sequential, bank send | 10 accts x 200 txs |
| `TestBenchmarkBaselineBurst` | Burst, bank send | 10 accts x 200 txs |
| `TestBenchmarkBaselineSeqMoveExec` | Sequential, Move exec | 100 accts x 50 txs, 30 writes/tx |

**Comparisons** (3-way: CList vs Proxy+IAVL vs Proxy+MemIAVL):

| Test | Load | Purpose |
|---|---|---|
| `TestBenchmarkSeqComparison` | Sequential, bank send | Fair TPS comparison, lightweight workload |
| `TestBenchmarkSeqComparisonMoveExec` | Sequential, Move exec | Fair TPS comparison under heavy state pressure |
| `TestBenchmarkBurstComparison` | Burst, bank send | Inclusion rate (CList drops txs) |
| `TestBenchmarkBurstComparisonMoveExec` | Burst, Move exec | Inclusion rate under heavy state pressure |

### 2. State db comparison: IAVL vs MemIAVL

Both use Proxy+Priority mempool. Isolates state storage impact.

| Test | Workload | Config |
|---|---|---|
| `TestBenchmarkMemIAVLBankSend` | Bank sends | 100 accts x 200 txs |
| `TestBenchmarkMemIAVLMoveExec` | Move exec (write_mixed) | 100 accts x 50 txs, 30 writes/tx |

### 3. Pre-signed HTTP broadcast (saturated chain)

Bypasses CLI bottleneck. Txs are generated+signed offline, then POSTed via HTTP to `/broadcast_tx_sync`.

| Test | Load | Config |
|---|---|---|
| `TestBenchmarkPreSignedSeqComparison` | Sequential, bank send, HTTP | 20 accts x 100 txs |
| `TestBenchmarkPreSignedBurstComparison` | Burst, bank send, HTTP | 20 accts x 100 txs |
| `TestBenchmarkPreSignedSeqMoveExec` | Sequential, Move exec, HTTP | 20 accts x 100 txs, 100 writes/tx |
| `TestBenchmarkPreSignedSeqMoveExecStress` | Sequential, Move exec, HTTP (stress) | 20 accts x 200 txs, 100 writes/tx |

### 4. Capability demos

| Test | What | Config |
|---|---|---|
| `TestBenchmarkQueuePromotion` | Out-of-order nonce handling, 100% inclusion | 10 accts x 50 txs |
| `TestBenchmarkGossipPropagation` | Gossip across nodes | 5 accts x 50 txs |
| `TestBenchmarkQueuedFlood` | Future-nonce burst (nonce gaps), queued pool stress + promotion cascade | 10 accts x 50 txs |
| `TestBenchmarkQueuedGapEviction` | Gap TTL eviction under sustained load | 10 accts x 50 txs |

## Expected outcomes

1. **Sequential (fair comparison)**: CList and Proxy+Priority both handle in-order nonces correctly, so sequential submission should show similar TPS. This is the control that proves Proxy+Priority doesn't regress on the happy path.
2. **Burst (stress test)**: Proxy+Priority >> CList. Under burst, CList's `reCheckTx` and cache-based dedup cause it to silently drop txs, while Proxy+Priority's queued pool absorbs out-of-order arrivals and achieves 100% inclusion.
3. **Heavy state writes**: MemIAVL > IAVL. Lightweight workloads (bank send) won't show a difference because the state db isn't the bottleneck. Heavy Move exec with many writes per tx is needed, and the chain must be saturated (pre-signed HTTP) so the state db becomes the limiting factor.
4. **Combined (Proxy+Priority+MemIAVL)**: Best overall throughput and latency, the mempool improvement eliminates tx drops, and MemIAVL reduces state commit time under heavy writes.

## Results

### 1. Mempool comparison: CList (v1.1.10) vs Proxy+Priority

#### Sequential bank send

| Config | Variant | TPS | vs base | P50ms | vs base | P95ms | vs base | Included | Peak MP |
|---|---|---:|--------:|---:|--------:|---:|--------:|---:|---:|
| clist/iavl/seq | baseline | 55.1 |       - | 1057 |       - | 1905 |       - | 2000/2000 | 102 |
| proxy+priority/iavl/seq | mempool-only | 53.5 |   -2.9% | 262 |  -75.2% | 352 |  -81.5% | 2000/2000 | 10 |
| proxy+priority/memiavl/seq | combined | 53.4 |   -3.1% | 255 |  -75.9% | 350 |  -81.6% | 2000/2000 | 12 |

#### Burst bank send

| Config | Variant | TPS | vs base | P50ms | vs base | P95ms | vs base | Included | Peak MP |
|---|---|---:|--------:|---:|--------:|---:|--------:|---:|---:|
| clist/iavl/burst | baseline | 27.0 |       - | 295 |       - | 823 |       - | 22/2000 | 10 |
| proxy+priority/iavl/burst | mempool-only | 54.4 | +101.5% | 257 |  -12.9% | 346 |  -58.0% | 2000/2000 | 10 |
| proxy+priority/memiavl/burst | combined | 54.9 | +103.3% | 252 |  -14.6% | 349 |  -57.6% | 2000/2000 | 12 |

#### Sequential Move exec (BenchHeavyState, 30 unique-key writes/tx)

| Config | Variant | TPS | vs base | P50ms | vs base | P95ms | vs base | Included | Peak MP |
|---|---|---:|--------:|---:|--------:|---:|--------:|---:|---:|
| clist/iavl/seq-move-exec | baseline | 30.8 |       - | 2325 |       - | 3223 |       - | 3091/5000 | 1000 |
| proxy+priority/iavl/seq-move-exec | mempool-only | 49.6 |  +61.0% | 2198 |   -5.5% | 2962 |   -8.1% | 5000/5000 | 147 |
| proxy+priority/memiavl/seq-move-exec | combined | 51.7 |  +67.9% | 1972 |  -15.2% | 2649 |  -17.8% | 5000/5000 | 44 |

#### Burst Move exec (BenchHeavyState, 30 unique-key writes/tx)

| Config | Variant | TPS | P50ms | P95ms | P99ms | Included | Peak MP |
|---|---|---:|---:|---:|---:|---:|---:|
| proxy+priority/iavl/burst-move-exec | mempool-only | 49.8 | 2173 | 3034 | 3483 | 5000/5000 | 115 |
| proxy+priority/memiavl/burst-move-exec | combined | 50.4 | 2086 | 2807 | 3090 | 5000/5000 | 66 |

### 2. State db comparison: IAVL vs MemIAVL (CLI-based, Proxy+Priority)

| Config | Workload | TPS | P50ms | P95ms | P99ms | Included | Peak MP |
|---|---|---:|---:|---:|---:|---:|---:|
| memiavl-compare/iavl/bank-send | bank send | 50.3 | 2107 | 2751 | 3039 | 20000/20000 | 104 |
| memiavl-compare/memiavl/bank-send | bank send | 50.5 | 2054 | 2685 | 2945 | 20000/20000 | 57 |
| memiavl-compare/iavl/move-exec | move exec | 50.3 | 2159 | 2943 | 3294 | 5000/5000 | 112 |
| memiavl-compare/memiavl/move-exec | move exec | 51.7 | 1984 | 2647 | 2931 | 5000/5000 | 45 |

CLI-based tests are bottlenecked by CLI overhead (~50 TPS ceiling), masking IAVL vs MemIAVL throughput differences. MemIAVL shows improvement in peak mempool size (-45% for bank send, -60% for move exec). See pre-signed HTTP results below for saturated-chain comparison.

### 3. Pre-signed HTTP broadcast (saturated chain)

#### Bank send (IAVL vs MemIAVL)

| Config | TPS | P50ms | P95ms | P99ms | Included | Peak MP |
|---|---:|---:|---:|---:|---:|---:|
| presigned/iavl/seq | 1249.7 | 263 | 375 | 405 | 2000/2000 | 624 |
| presigned/memiavl/seq | 1449.3 | 332 | 418 | 591 | 2000/2000 | 729 |
| presigned/iavl/burst | 1440.9 | 713 | 1214 | 1219 | 2000/2000 | 1597 |
| presigned/memiavl/burst | 1471.6 | 676 | 1216 | 1222 | 2000/2000 | 1594 |

#### Move exec (IAVL vs MemIAVL, 100 unique-key writes/tx)

| Config | TPS | P50ms | P95ms | P99ms | Included | Peak MP |
|---|---:|---:|---:|---:|---:|---:|
| presigned/iavl/seq-move-exec | 512.8 | 992 | 2107 | 2130 | 2000/2000 | 1121 |
| presigned/memiavl/seq-move-exec | 1021.7 | 550 | 900 | 942 | 2000/2000 | 896 |

#### Move exec stress (IAVL vs MemIAVL, 100 unique-key writes/tx, 4000 txs)

| Config | TPS | P50ms | P95ms | P99ms | Included | Peak MP |
|---|---:|---:|---:|---:|---:|---:|
| presigned-stress/iavl/seq-move-exec | 237.1 | 3342 | 8601 | 9093 | 4000/4000 | 1718 |
| presigned-stress/memiavl/seq-move-exec | 806.1 | 1238 | 2239 | 2273 | 4000/4000 | 1726 |

Under saturated heavy state writes with continuously growing state tree, MemIAVL demonstrates decisive superiority.
At 2000 txs (100 writes/tx): **+99.2% TPS** (1021.7 vs 512.8) and **-44.6% P50 latency** (550 vs 992ms) with 100% inclusion for both.
At 4000 txs (100 writes/tx): **+240.0% TPS** (806.1 vs 237.1) and **-63.0% P50 latency** (1238 vs 3342ms) with 100% inclusion for both.
IAVL degrades sharply as the state tree grows while MemIAVL maintains consistent throughput.

### 4. Capability demos

| Test |   TPS | P50ms | P95ms | P99ms | Included | Peak MP | Notes |
|---|------:|------:|------:|------:|---------:|--------:|---|
| queue-promotion |  55.6 |   250 |   361 |   601 |  500/500 |      11 | Out-of-order nonces, 100% inclusion |
| gossip |     - |     - |     - |     - |  250/250 |       - | All txs to single node, gossip to validator |
| queued-flood | 574.0 |  6890 | 10823 | 11302 |  500/500 |     490 | Nonce gap burst + promotion cascade |
| queued-gap-eviction |     - |     - |     - |     - |        - |       - | Qualitative: gap TTL eviction confirmed, mempool drained |

### Understanding the metrics

#### Latency (P50 / P99)

P50 (median) is the latency half of txs beat. P99 is the 99th percentile, meaning only 1% of txs are slower. 
The gap between them measures how consistent the system is.

**Sequential bank send with Proxy+Priority** has the lowest latency (P50=255ms, P99=397ms) and the tightest gap (~150ms). 
This is the lightest happy path: in-order nonces, no state contention. A tx arrives at a fullnode, gossips to the sequencer, gets proposed in the next block (~100ms interval), and commits almost instantly. 
Nearly all txs have the same experience.

**CLI-based Move exec** has the highest latency (P50 ~2000ms+, P99 ~3000ms+) for two reasons. 
- First, each `minitiad tx move execute-json` spawns a process, signs, and broadcasts which serializes submissions per account, so txs submitted later have already waited in the CLI queue before reaching the mempool. 
- Second, 30 writes/tx means blocks take longer to execute and commit. The wide P50-P99 gap (~1000ms) reflects accounts finishing at different rates.

**Pre-signed Move exec** shows the most interesting pattern:

| Variant | P50ms | P99ms | Gap |
|---|---:|---:|---:|
| IAVL | 992 | 2130 | 1138 |
| MemIAVL | 550 | 942 | 392 |

MemIAVL is lower on both, but the P50-P99 gap shrinks dramatically (1138ms -> 392ms). 
With IAVL, slow state commits cause a backlog that txs submitted mid-run wait much longer than early txs, spreading out the latency distribution. 
MemIAVL commits faster so the backlog never builds up, keeping latency consistent across all txs. 
P99 captures the worst-case user experience, a system with low P50 but high P99 means most users are happy but some wait disproportionately longer.

#### Peak mempool size

Peak mempool measures the maximum number of txs sitting in the mempool at any point during the test. It's a queue depth indicator.

The chain processes txs in a pipeline: **mempool -> proposal -> execute -> commit state**. 
The commit phase is where IAVL vs MemIAVL matters. With IAVL, state commit takes longer (tree rebalancing under growing state), so blocks take longer to finalize. 
While the chain is stuck committing block N, incoming txs accumulate in the mempool waiting for block N+1. MemIAVL commits faster -> blocks finalize faster -> the mempool drains faster -> fewer txs pile up at any given moment.

The effect scales with workload intensity:

| Test | IAVL | MemIAVL | Reduction |
|---|---:|---:|---:|
| CLI bank send (20k txs) | 104 | 57 | -45% |
| CLI move exec (5k txs, 30 writes/tx) | 112 | 45 | -60% |
| Pre-signed move exec (2k txs, 100 writes/tx) | 1121 | 896 | -20% |

The heavier the state writes per tx, the more time IAVL spends on commit, and the more the mempool backs up. 
The pre-signed case is slightly different with both peak high because the HTTP broadcast submission rate far exceeds block throughput, but MemIAVL still drains the queue faster overall.

## Run

All commands assume `cd integration-tests` first. The full workflow has 3 phases:
baselines first, then current-branch benchmarks, then the comparison tests that
load both result sets. Capability demos / queued tests are standalone and can run
any time.

### Phase 1: Collecting baselines (CList mempool)

Build the pre-proxy binary once, then run the baseline tests.
Results are written to `e2e/benchmark/results/` as JSON keyed by label.

```bash
# Build pre-proxy binary
git checkout tags/v1.1.10
go build -o build/minitiad-baseline ./cmd/minitiad
git checkout -   # return to current branch

cd integration-tests

# Sequential bank send baseline
E2E_MINITIAD_BIN="$(pwd)/../build/minitiad-baseline" \
  go test -v -tags benchmark -run TestBenchmarkBaselineSeq -timeout 30m -count=1 ./e2e/benchmark/

# Burst bank send baseline
E2E_MINITIAD_BIN="$(pwd)/../build/minitiad-baseline" \
  go test -v -tags benchmark -run TestBenchmarkBaselineBurst -timeout 30m -count=1 ./e2e/benchmark/

# Sequential Move exec baseline
E2E_MINITIAD_BIN="$(pwd)/../build/minitiad-baseline" \
  go test -v -tags benchmark -run TestBenchmarkBaselineSeqMoveExec -timeout 60m -count=1 ./e2e/benchmark/
```

### Phase 2: Running current-branch benchmarks

These use the current binary (auto-built or via `E2E_MINITIAD_BIN`).
Each test writes its own result JSON.

```bash
# Build current binary
go build -o ./minitiad ../cmd/minitiad

# State db comparison (IAVL vs MemIAVL)
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkMemIAVLBankSend -timeout 60m -count=1 ./e2e/benchmark/
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkMemIAVLMoveExec -timeout 60m -count=1 ./e2e/benchmark/

# Capability demos (standalone, no baselines needed)
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkQueuePromotion -timeout 30m -count=1 ./e2e/benchmark/
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkGossipPropagation -timeout 30m -count=1 ./e2e/benchmark/

# Queued mempool behavior (standalone, no baselines needed)
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkQueuedFlood -timeout 30m -count=1 ./e2e/benchmark/
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkQueuedGapEviction -timeout 30m -count=1 ./e2e/benchmark/
```

### Pre-signed HTTP broadcast tests (saturated chain)

These use pre-signed txs via HTTP to saturate the chain, bypassing the CLI bottleneck.

```bash
# Sequential bank send (IAVL vs MemIAVL)
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkPreSignedSeqComparison -timeout 20m -count=1 ./e2e/benchmark/

# Burst bank send (IAVL vs MemIAVL)
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkPreSignedBurstComparison -timeout 20m -count=1 ./e2e/benchmark/

# Sequential Move exec (IAVL vs MemIAVL)
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkPreSignedSeqMoveExec$ -timeout 30m -count=1 ./e2e/benchmark/

# Sequential Move exec stress (IAVL vs MemIAVL, 4000 txs)
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkPreSignedSeqMoveExecStress -timeout 30m -count=1 ./e2e/benchmark/
```

### Phase 3: Comparison tests (baseline vs current)

These load baseline JSONs from `e2e/benchmark/results/` by label and run Proxy+IAVL
and Proxy+MemIAVL variants, then print a side-by-side comparison table with deltas.

```bash
# Sequential bank send: CList vs Proxy+IAVL vs Proxy+MemIAVL
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkSeqComparison$ -timeout 30m -count=1 ./e2e/benchmark/

# Sequential Move exec: CList vs Proxy+IAVL vs Proxy+MemIAVL
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkSeqComparisonMoveExec -timeout 60m -count=1 ./e2e/benchmark/

# Burst bank send: CList vs Proxy+IAVL vs Proxy+MemIAVL
E2E_MINITIAD_BIN=./minitiad \
  go test -v -tags benchmark -run TestBenchmarkBurstComparison -timeout 30m -count=1 ./e2e/benchmark/
```

## Configuration

### Ground Rules

1. Baseline requires a separate binary built from v1.1.10 (pre-proxy CometBFT, pre-ABCI++ changes).
2. Run baseline and current benchmarks on the same machine.
3. Warmup runs before every measured load (5 txs, metadata re-queried after).
4. TPS is derived from block timestamps, not submission wall clock.
5. Latency = `block_time - submit_time` (covers mempool wait, gossip, proposal, execution).
6. Load is submitted to edge nodes (non-validator) to test realistic gossip propagation.

### Configurable mempool limits

These can be tuned in `app.toml` under `[abcipp]` (defaults shown):

| Parameter | Default | Description |
|---|---|---|
| `max-queued-per-sender` | 64 | Max queued txs per sender |
| `max-queued-total` | 1024 | Max queued txs globally |
| `queued-gap-ttl` | 60s | TTL for stalled senders missing head nonce |

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `E2E_MINITIAD_BIN` | (auto-build) | Path to prebuilt `minitiad` binary |
| `BENCHMARK_RESULTS_DIR` | `results/` | Output directory for JSON results |

## Structure

```
benchmark/
  config.go          Variant definitions, BenchConfig, preset constructors
  load.go            Load generators
  collector.go       MempoolPoller, CollectResults, latency aggregation
  report.go          JSON output, comparison tables, delta calculations, LoadBaselineResultsByLabel
  benchmark_test.go  Test suite (build-tagged `benchmark`)
  move-bench/        Move module (BenchHeavyState, write_mixed)
  results/           JSON output directory
```

### Load generators

All load generators route transactions to fullnodes when `ValidatorCount > 0`.

- **BurstLoad**: All accounts submit concurrently with sequential nonces, round-robin across fullnodes.
- **SequentialLoad**: Accounts run concurrently, but each account sends txs one-at-a-time. Each account pinned to a single fullnode.
- **OutOfOrderLoad**: First 3 txs per account use `[seq+2, seq+0, seq+1]` to test queue promotion.
- **SingleNodeLoad**: All txs to a single node for gossip propagation measurement.
- **MoveExecBurstLoad**: Like BurstLoad but calls `SendMoveExecuteJSONWithGas` (`write_mixed`) instead of bank sends.
- **MoveExecSequentialLoad**: Like SequentialLoad but calls `SendMoveExecuteJSONWithGas`. Each account pinned to a single fullnode.
- **QueuedFloodLoad**: Sends txs with nonces `[base+1..base+N]` (skipping `base+0`), then after all are submitted, sends the gap-filling `base+0` tx to trigger promotion cascade.
- **PreSignedBurstLoad**: Broadcasts pre-signed Cosmos txs via HTTP POST to `/broadcast_tx_sync`. All accounts concurrent, round-robin across fullnodes.
- **PreSignedSequentialLoad**: Broadcasts pre-signed Cosmos txs via HTTP POST. Each account pinned to a single fullnode, txs sent sequentially per account.

### Metrics

| Metric | Source |
|---|---|
| **TPS** | `included_tx_count / block_time_span` |
| **Latency** (avg, p50, p95, p99, max) | `block_timestamp - submit_timestamp` per tx |
| **Peak mempool size** | Goroutine polling `/num_unconfirmed_txs` every 500ms |
| **Per-block tx count** | CometBFT RPC `/block?height=N` |

## Move exec workload: BenchHeavyState

The Move exec tests deploy the `BenchHeavyState` Move module at runtime (built via `minitiad move build`). Each tx calls `write_mixed(shared_count, local_count)` which performs:

- **shared writes** to the `SharedState` table (contended across all accounts).
- **local writes** to the per-account `State` table (non-contended).

Each call writes to **unique keys** using a per-sender counter, so the state tree grows continuously. This creates IAVL rebalancing pressure that MemIAVL handles more efficiently.

CLI-based tests use `write_mixed(5, 25)` = 30 writes/tx. Pre-signed HTTP tests and stress tests use `write_mixed(20, 80)` = 100 writes/tx.
