import datetime
from typing import Optional

from sqlalchemy.orm import Mapped, mapped_column
from sqlalchemy import ForeignKey, Index, String

from syncer.model.base import Base


class Review(Base):
    __tablename__ = "reviews"

    id: Mapped[int] = mapped_column(primary_key=True)
    repo: Mapped[str] = mapped_column(String, index=True)
    pull_request_id: Mapped[int] = mapped_column(ForeignKey("pull_requests.id"))
    username: Mapped[str]
    state: Mapped[str]
    submitted_at: Mapped[Optional[datetime.datetime]]
    commit_id: Mapped[str]
    body: Mapped[str]


Index("idx_review_repo", Review.repo)
Index("idx_review_state", Review.repo)
