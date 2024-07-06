import datetime
from typing import Optional, List

from sqlalchemy.orm import Mapped, mapped_column
from sqlalchemy import Index, String
from sqlalchemy.dialects.postgresql import ARRAY

from syncer.model.base import Base


class PullRequest(Base):
    __tablename__ = "pull_requests"

    id: Mapped[int] = mapped_column(primary_key=True)
    repo: Mapped[str] = mapped_column(String, index=True)
    number: Mapped[int]
    title: Mapped[str]
    state: Mapped[str]
    created_at: Mapped[Optional[datetime.datetime]]
    updated_at: Mapped[Optional[datetime.datetime]]
    closed_at: Mapped[Optional[datetime.datetime]]
    merged_at: Mapped[Optional[datetime.datetime]]
    requested_reviewers: Mapped[List[str]] = mapped_column(ARRAY(String))
    requested_teams: Mapped[List[str]] = mapped_column(ARRAY(String))
    labels: Mapped[List[str]] = mapped_column(ARRAY(String))
    draft: Mapped[bool]
    base: Mapped[str]
    username: Mapped[Optional[str]]
    merged: Mapped[bool]
    head_ref: Mapped[str]
    last_ready_for_review_at: Mapped[Optional[datetime.datetime]]
    first_reviewed_at: Mapped[Optional[datetime.datetime]]
    additions: Mapped[int]
    deletions: Mapped[int]
    changed_files: Mapped[int]


Index("idx_pull_request_updated_at", PullRequest.updated_at)
Index("idx_pull_request_repo", PullRequest.repo)
