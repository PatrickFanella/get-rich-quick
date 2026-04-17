-- pgvector extension for semantic embedding storage and retrieval.
CREATE EXTENSION IF NOT EXISTS vector;

-- Embedding column on news_feed: nomic-embed-text outputs 768-dimensional vectors.
ALTER TABLE news_feed ADD COLUMN embedding vector(768);

-- Embedding column on social_sentiment.
ALTER TABLE social_sentiment ADD COLUMN embedding vector(768);

-- Per-post scored details for reddit sentiment (replaces loss of inline data).
ALTER TABLE social_sentiment ADD COLUMN post_summaries JSONB;

-- HNSW indexes for approximate cosine similarity search.
-- HNSW chosen over ivfflat: no minimum row requirement, better recall,
-- and both tables have enough rows (1500+ / 3200+) for either index type.
CREATE INDEX idx_news_feed_embedding
    ON news_feed USING hnsw (embedding vector_cosine_ops);

CREATE INDEX idx_social_sentiment_embedding
    ON social_sentiment USING hnsw (embedding vector_cosine_ops);
