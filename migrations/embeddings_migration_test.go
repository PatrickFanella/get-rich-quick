package migrations_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEmbeddingsUpMigrationDefinesExpectedSchema(t *testing.T) {
	upSQL := normalizeSQL(t, readMigrationFile(t, "000030_embeddings.up.sql"))

	for _, fragment := range []string{
		"create extension if not exists vector",
		"alter table news_feed add column embedding vector(768)",
		"alter table social_sentiment add column embedding vector(768)",
		"alter table social_sentiment add column post_summaries jsonb",
		"create index idx_news_feed_embedding",
		"on news_feed using hnsw (embedding vector_cosine_ops)",
		"create index idx_social_sentiment_embedding",
		"on social_sentiment using hnsw (embedding vector_cosine_ops)",
	} {
		if !strings.Contains(upSQL, fragment) {
			t.Fatalf("expected up migration to contain %q, got:\n%s", fragment, upSQL)
		}
	}
}

func TestEmbeddingsDownMigrationDropsColumnsAndExtension(t *testing.T) {
	downSQL := normalizeSQL(t, readMigrationFile(t, "000030_embeddings.down.sql"))

	for _, fragment := range []string{
		"drop index if exists idx_social_sentiment_embedding",
		"drop index if exists idx_news_feed_embedding",
		"alter table social_sentiment drop column if exists post_summaries",
		"alter table social_sentiment drop column if exists embedding",
		"alter table news_feed drop column if exists embedding",
		"drop extension if exists vector",
	} {
		if !strings.Contains(downSQL, fragment) {
			t.Fatalf("expected down migration to contain %q, got:\n%s", fragment, downSQL)
		}
	}
}

func TestEmbeddingsMigrationAppliesAgainstExistingSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping migration integration test in short mode")
	}

	databaseURL := os.Getenv("DB_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("skipping migration integration test: DB_URL or DATABASE_URL is not set")
	}

	ctx := context.Background()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to create admin pool: %v", err)
	}
	t.Cleanup(adminPool.Close)

	if _, err := adminPool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		t.Fatalf("failed to ensure pgcrypto extension: %v", err)
	}

	schemaName := "migr_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	sanitizedSchemaName := pgx.Identifier{schemaName}.Sanitize()
	if _, err := adminPool.Exec(ctx, `CREATE SCHEMA `+sanitizedSchemaName); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := adminPool.Exec(ctx, `DROP SCHEMA IF EXISTS `+sanitizedSchemaName+` CASCADE`); err != nil {
			t.Errorf("failed to drop schema %q: %v", schemaName, err)
		}
	})

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("failed to parse database config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName + ",public"
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("failed to create schema-scoped pool: %v", err)
	}
	t.Cleanup(pool.Close)

	// Apply all migrations through 000030.
	for _, filename := range sortedUpMigrationsThrough(t, "000030_embeddings.up.sql") {
		if _, err := pool.Exec(ctx, readMigrationFile(t, filename)); err != nil {
			t.Fatalf("failed to apply %s: %v", filename, err)
		}
	}

	// Verify news_feed.embedding column exists with type vector.
	var newsEmbeddingType string
	err = pool.QueryRow(ctx,
		`SELECT udt_name FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = 'news_feed' AND column_name = 'embedding'`,
		schemaName).Scan(&newsEmbeddingType)
	if err != nil {
		t.Fatalf("news_feed.embedding column not found: %v", err)
	}
	if newsEmbeddingType != "vector" {
		t.Fatalf("expected news_feed.embedding type 'vector', got %q", newsEmbeddingType)
	}

	// Verify social_sentiment.embedding column exists.
	var socialEmbeddingType string
	err = pool.QueryRow(ctx,
		`SELECT udt_name FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = 'social_sentiment' AND column_name = 'embedding'`,
		schemaName).Scan(&socialEmbeddingType)
	if err != nil {
		t.Fatalf("social_sentiment.embedding column not found: %v", err)
	}

	// Verify social_sentiment.post_summaries column exists.
	var postSummariesType string
	err = pool.QueryRow(ctx,
		`SELECT udt_name FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = 'social_sentiment' AND column_name = 'post_summaries'`,
		schemaName).Scan(&postSummariesType)
	if err != nil {
		t.Fatalf("social_sentiment.post_summaries column not found: %v", err)
	}
	if postSummariesType != "jsonb" {
		t.Fatalf("expected post_summaries type 'jsonb', got %q", postSummariesType)
	}

	// Verify down migration works: apply it and confirm columns are gone.
	downSQL := readMigrationFile(t, "000030_embeddings.down.sql")
	if _, err := pool.Exec(ctx, downSQL); err != nil {
		t.Fatalf("failed to apply down migration: %v", err)
	}

	var colCount int
	err = pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = 'news_feed' AND column_name = 'embedding'`,
		schemaName).Scan(&colCount)
	if err != nil {
		t.Fatalf("query after down migration: %v", err)
	}
	if colCount != 0 {
		t.Fatalf("expected news_feed.embedding to be dropped, but still found %d", colCount)
	}
}
