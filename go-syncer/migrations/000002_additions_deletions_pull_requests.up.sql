-- 000002_additions_deletions_pull_requests.up.sql
ALTER TABLE pull_requests
ADD COLUMN additions INT,
ADD COLUMN deletions INT,
ADD COLUMN changed_files INT;
