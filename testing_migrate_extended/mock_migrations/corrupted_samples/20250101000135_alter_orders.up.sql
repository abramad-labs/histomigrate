ALTER TABLE orders ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT 'test_corrupted';
