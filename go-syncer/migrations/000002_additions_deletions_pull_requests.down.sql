-- 000002_additions_deletions_pull_requests.down.sql
ALTER TABLE pull_requests
DROP COLUMN additions,
DROP COLUMN deletions,
DROP COLUMN changed_files;
