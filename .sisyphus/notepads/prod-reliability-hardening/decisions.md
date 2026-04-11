2026-04-11: Retained retry behavior for 429/5xx/context canceled; snip only DeadlineExceeded changed.
2026-04-11: Kept 429/5xx retryable; only DeadlineExceeded moved to fail-fast so fallback can take over.
