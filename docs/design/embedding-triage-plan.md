# Embedding-Based Triage: Background Ingestion Plan

## TL;DR

Move all LLM triage (RSS, news, reddit sentiment) out of strategy runs and into background
automation jobs. Store results with vector embeddings (nomic-embed-text + pgvector) for
semantic retrieval. Strategy runs become pure DB reads — no ollama contention during the
analysis phase.

## Context: Why This Is Needed

Observed symptoms from production monitoring (April 2026):

| Symptom                                           | Evidence                                               |
| ------------------------------------------------- | ------------------------------------------------------ |
| 64 ollama timeouts in 12h                         | `ollama: complete request: context deadline exceeded`  |
| `social_media_analyst` never produces real output | 0/10 recent decisions had real sentiment               |
| `analysis_ms` always hits 300s ceiling            | phase_timings: `300022`, `300025`, `300029` ms         |
| RSS triage job perpetually overlapping            | `automation: skipping overlapping run` for `news_scan` |
| 2/5 recent runs failed outright                   | `context deadline exceeded` in error_message           |

**Root cause**: Multiple consumers fight one local ollama instance simultaneously — reddit
sentiment batches, RSS triage, analysis agents, signal evaluator, and signal hub all call
ollama with no coordination. Every strategy run triggers an LLM spike that starves itself.

## Decisions

- **Embedding model**: `nomic-embed-text` via local ollama (137M params, already downloaded, fast)
- **Storage**: FTS (GIN) for fast keyword filter + pgvector for semantic cosine ranking
- **Triage model**: `qwen3.5:latest` (9.7B) — lighter than current `qwen3:14b`, already downloaded
- **Reddit scoring**: Move to background `social_scan` job, not inline during strategy run

## Target Architecture

```text
BACKGROUND INGESTION (automation jobs, no time pressure)
├── news_scan (every 5min, market hours)
│   └── RSS fetch → LLM triage (qwen3.5) → embed (nomic-embed-text) → news_feed table
├── social_scan (every 15min, market hours)
│   └── Reddit fetch → LLM score (qwen3.5) → embed (nomic-embed-text) → social_sentiment table
└── signal hub → unchanged, benefits passively from reduced ollama load

STRATEGY RUN (pure DB reads, zero ollama calls for data)
└── loadInitialState
    ├── GetNews     → SELECT FROM news_feed (GIN filter + pgvector ranking)
    ├── GetSocial   → SELECT FROM social_sentiment WHERE ticker = X (pre-scored)
    └── OHLCV, fundamentals → unchanged
```

## Implementation Plan

### Phase 1 — pgvector + Embedding Infrastructure

#### Step 1 — Migration `000030_embeddings`

File: `migrations/000030_embeddings.up.sql`

- `CREATE EXTENSION IF NOT EXISTS vector`
- `ALTER TABLE news_feed ADD COLUMN embedding vector(768)` — nomic-embed-text outputs 768 dims
- `ALTER TABLE social_sentiment ADD COLUMN embedding vector(768)`
- `ALTER TABLE social_sentiment ADD COLUMN post_summaries JSONB` — store per-post scored detail
- `CREATE INDEX idx_news_feed_embedding ON news_feed USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)`
- `CREATE INDEX idx_social_sentiment_embedding ON social_sentiment USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)`

Down migration drops indexes, columns, and extension.

> Note: ivfflat requires at least `lists` rows to build; hnsw is an alternative with better
> recall at the cost of slower index build. Prefer ivfflat initially, migrate to hnsw if
> recall is poor.

#### Step 2 — New `internal/llm/embedding/` package

Files: `internal/llm/embedding/provider.go`, `internal/llm/embedding/ollama.go`

Interface:

```go
type Provider interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}
```

`OllamaProvider` implementation:

- POSTs to ollama `/api/embed` endpoint
- Config: `Model` (default `nomic-embed-text`), `BaseURL`, `BatchSize`, `Timeout`
- Returns `[]float32` of length 768
- Errors wrapped with `fmt.Errorf("embedding: %w", err)`

#### Step 3 — Wire into app bootstrap

`cmd/tradingagent/main.go` (or wherever llm providers are constructed):

- Construct `embedding.OllamaProvider` alongside existing `llm.Provider`
- Pass through `automation.Orchestrator` deps and `data.Service` deps

---

### Phase 2 — Background Triage with Lighter Model

#### Step 4 — Update `newsScan` in `internal/automation/jobs_news.go`

- Switch triage LLM model to `qwen3.5:latest` (pass via config, not hardcoded)
- After `rss.Triage()` returns results, call `embedding.EmbedBatch()` on `title + " " + summary` for each article
- Call `newsRepo.UpdateEmbedding(ctx, guid, vec)` to persist vector
- Gate embed call: skip articles where `embedding IS NOT NULL` (idempotent backfill support)

#### Step 5 — Update `socialScan` in `internal/automation/jobs_news.go`

Add reddit sentiment scoring (currently only runs inline during strategy):

- After StockTwits processing, iterate active strategy tickers (already done for StockTwits)
- For each ticker: call `reddit.ScorePosts(ctx, provider, model, ticker, posts, logger)` using `qwen3.5`
- Build summary string from result: `"ticker BULL:X BEAR:Y NEUTRAL:Z from N mentions"`
- Call `embedding.Embed(ctx, summaryText)` → store vector in `social_sentiment` row

#### Step 6 — Update `internal/data/reddit/provider.go`

`GetSocialSentiment`:

- Change to read from `social_sentiment` table via `socialRepo.GetLatestByTicker()`
- Remove the inline `ScorePosts` + `llmProvider` call
- Keep `ScorePosts` function — it's now called only from `socialScan` job

---

### Phase 3 — Semantic Retrieval for Strategy Runs

#### Step 7 — Add `GetRelevantByEmbedding` to `internal/repository/postgres/news_feed.go`

```sql
SELECT id, guid, source, title, description, link, published_at,
       tickers, category, sentiment, relevance, summary
FROM news_feed
WHERE published_at >= $1
  AND (tickers @> ARRAY[$2]::text[] OR embedding <=> $3 < 0.35)
ORDER BY embedding <=> $3
LIMIT $4
```

Signature: `GetRelevantByEmbedding(ctx, queryVec []float32, ticker string, from time.Time, limit int) ([]domain.NewsArticle, error)`

The `0.35` cosine distance threshold is a starting value; tune based on recall quality.

#### Step 8 — Add `GetLatestByTicker` to social_sentiment repository

File: `internal/repository/postgres/social_sentiment.go` (create if absent, else add method)

```sql
SELECT ticker, source, sentiment_score, bullish, bearish, post_count, measured_at
FROM social_sentiment
WHERE ticker = $1 AND measured_at >= $2
ORDER BY measured_at DESC
LIMIT 1
```

#### Step 9 — Update `GetNews` in `internal/data/factory.go`

Replace provider-chain fallback with:

1. Generate query embedding: `embedding.Embed(ctx, ticker + " stock trading signals news")`
2. Call `newsRepo.GetRelevantByEmbedding(ctx, vec, ticker, from, limit)` (limit from config, default 20)
3. Return articles directly — no market_data_cache write needed (DB is source of truth)
4. Retain cache read as fast path: if cache hit exists and is < 5min old, return it; otherwise query DB

#### Step 10 — Update `GetSocialSentiment` in `internal/data/factory.go`

- Read from `social_sentiment` table via `GetLatestByTicker`
- Remove inline reddit provider fallback (reddit is now background-only)
- Remove StockTwits fallback from factory (social_scan handles both sources)

#### Step 11 — Simplify `loadInitialState` in `cmd/tradingagent/prod_strategy_runner.go`

- Remove the `reddit.Provider.GetSocialSentiment()` inline call (now reads from DB)
- `GetNews` and `GetSocialSentiment` both return pre-computed data — no LLM, no timeout risk
- `social_media_analyst` receives real sentiment data on every run

---

### Phase 4 — Config + Cleanup

#### Step 12 — Add config fields

Location: wherever global pipeline config lives (check `internal/config/` or `internal/agent/resolve_config.go`):

```text
triage_model        string  // default: "qwen3.5:latest"
embedding_model     string  // default: "nomic-embed-text"
embedding_dim       int     // default: 768
news_semantic_limit int     // default: 20
```

#### Step 13 — Remove dead code paths

- Delete `ScorePosts` invocation from `prod_strategy_runner.go` / `loadInitialState`
- Delete market_data_cache write/read for `social_sentiment` data type (now direct DB reads)
- Delete old social provider chain fallback logic from `data/factory.go`

#### Step 14 — Align social_scan schedule

Ensure `social_scan` runs offset from strategy schedules so fresh data is ready:

- Strategies run `"0 */2 * * 1-5"` (top of every even hour)
- `social_scan` runs `"*/15 * * * 1-5"` — already runs at `:45` before each even hour ✓
- Verify `news_scan` at `"*/5 * * * 1-5"` is within 5min of strategy trigger — already ✓

---

## Relevant Files

| File                                               | Change                                                                |
| -------------------------------------------------- | --------------------------------------------------------------------- |
| `migrations/000030_embeddings.up.sql`              | New — pgvector ext + embedding columns + indexes                      |
| `migrations/000030_embeddings.down.sql`            | New — reversal                                                        |
| `internal/llm/embedding/provider.go`               | New — Provider interface                                              |
| `internal/llm/embedding/ollama.go`                 | New — OllamaProvider impl                                             |
| `internal/automation/jobs_news.go`                 | Modify `newsScan` + `socialScan`                                      |
| `internal/data/reddit/provider.go`                 | Read from DB instead of inline LLM                                    |
| `internal/data/reddit/sentiment.go`                | No change — stays, called by socialScan                               |
| `internal/data/factory.go`                         | `GetNews` → semantic search; `GetSocialSentiment` → direct read       |
| `internal/repository/postgres/news_feed.go`        | Add `GetRelevantByEmbedding`, `UpdateEmbedding`                       |
| `internal/repository/postgres/social_sentiment.go` | Add `GetLatestByTicker`                                               |
| `cmd/tradingagent/prod_strategy_runner.go`         | Simplify `loadInitialState`                                           |
| `internal/config/`                                 | Add triage_model, embedding_model, embedding_dim, news_semantic_limit |

---

## Verification Checklist

1. **Migration applied**

   ```sh
   docker exec augr-postgres-1 psql -U postgres -d tradingagent \
     -c "SELECT extname FROM pg_extension WHERE extname = 'vector'"
   ```

2. **Embedding provider unit tests** — test `OllamaProvider.Embed` against local ollama
   `nomic-embed-text`; assert output length = 768

3. **news_scan embeds articles**

   ```sh
   docker exec augr-postgres-1 psql -U postgres -d tradingagent \
     -c "SELECT COUNT(*) FROM news_feed WHERE embedding IS NOT NULL"
   ```

4. **social_scan scores reddit + embeds**

   ```sh
   docker exec augr-postgres-1 psql -U postgres -d tradingagent \
     -c "SELECT ticker, source, measured_at FROM social_sentiment WHERE source = 'reddit' ORDER BY measured_at DESC LIMIT 5"
   ```

5. **Strategy run — `analysis_ms` no longer hits 300s ceiling**
   - Trigger DAL strategy manually
   - Check `phase_timings.analysis_ms` < 60000 (under 60s)

6. **`social_media_analyst` produces real output**

   ```sh
   docker exec augr-postgres-1 psql -U postgres -d tradingagent \
     -c "SELECT output_text FROM agent_decisions WHERE agent_role = 'social_media_analyst' ORDER BY created_at DESC LIMIT 3"
   ```

   Should not contain "Analysis skipped to conserve resources."

7. **Semantic search quality** — query `news_feed` with embedding for "DAL airline earnings",
   verify top results are DAL-relevant articles (not unrelated noise)

8. **No regressions**

   ```sh
   go test -short -race -count=1 ./...
   ```

---

## Scope Boundaries

**Included:**

- pgvector setup + nomic-embed-text embedding infrastructure
- Background triage for RSS/news with qwen3.5
- Reddit sentiment scoring moved to background social_scan
- Semantic retrieval for strategy runs (GetNews, GetSocialSentiment)
- Config for model selection

**Excluded:**

- Signal hub evaluator (benefits passively from reduced ollama contention; no code change)
- Agent memory migration to pgvector (stays FTS per ADR-003)
- Polymarket data flow changes
- Duplicate strategy cleanup (DAL×2, MIMI×2 — separate issue)
- n8n webhook / notification fixes
- Backfilling embeddings for existing `news_feed` rows (can be done as a one-off script later)
