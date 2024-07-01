-- 000003_int_to_int8.down.sql
ALTER TABLE pull_requests
ALTER COLUMN pr_id TYPE INT USING pr_id::INT,
ALTER COLUMN repo_id TYPE INT USING repo_id::INT;

ALTER TABLE pull_request_reviews
ALTER COLUMN review_id TYPE INT USING review_id::INT,
ALTER COLUMN pr_id TYPE INT USING pr_id::INT;
