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
    created_at: Mapped[str]
    updated_at: Mapped[str] = mapped_column(String, index=True)
    closed_at: Mapped[Optional[datetime.datetime]]
    merged_at: Mapped[Optional[datetime.datetime]]
    requested_reviewers: Mapped[List[str]] = mapped_column(ARRAY(String))
    requested_teams: Mapped[List[str]] = mapped_column(ARRAY(String))
    labels: Mapped[List[str]] = mapped_column(ARRAY(String))
    draft: Mapped[bool]
    base: Mapped[str]
    user: Mapped[Optional[str]]
    merged: Mapped[bool]


Index("idx_updated_at", PullRequest.updated_at)
Index("idx_repo", PullRequest.repo)
