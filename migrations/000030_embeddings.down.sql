-- Reverse migration: remove pgvector embedding columns and extension.

DROP INDEX IF EXISTS idx_social_sentiment_embedding;
DROP INDEX IF EXISTS idx_news_feed_embedding;

ALTER TABLE social_sentiment DROP COLUMN IF EXISTS post_summaries;
ALTER TABLE social_sentiment DROP COLUMN IF EXISTS embedding;
ALTER TABLE news_feed DROP COLUMN IF EXISTS embedding;

DROP EXTENSION IF EXISTS vector;
