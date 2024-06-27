-- 000001_create_tables.up.sql
CREATE TABLE
  pull_requests (
    id SERIAL PRIMARY KEY,
    pr_id INT UNIQUE,
    repo TEXT,
    repo_id INT,
    username TEXT,
    title TEXT,
    body TEXT,
    state TEXT,
    draft BOOLEAN,
    merged_at TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    last_ready_for_review_at TIMESTAMP,
    data JSON
  );

CREATE TABLE
  pull_request_reviews (
    id SERIAL PRIMARY KEY,
    review_id INT UNIQUE,
    pr_id INT,
    repo TEXT,
    username TEXT,
    state TEXT,
    submitted_at TIMESTAMP,
    commit_id TEXT,
    data JSON
  );

CREATE TABLE
  sync_status (repo TEXT PRIMARY KEY, last_synced TIMESTAMP);
