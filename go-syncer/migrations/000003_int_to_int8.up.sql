-- 000003_int_to_int8.up.sql
ALTER TABLE pull_requests
ALTER COLUMN pr_id TYPE INT8 USING pr_id::INT8,
ALTER COLUMN repo_id TYPE INT8 USING repo_id::INT8;

ALTER TABLE pull_request_reviews
ALTER COLUMN review_id TYPE INT8 USING review_id::INT8,
ALTER COLUMN pr_id TYPE INT8 USING pr_id::INT8;
