import datetime
from typing import List, Optional

from sqlalchemy.orm import Mapped, mapped_column
from sqlalchemy import ARRAY, Index, String

from syncer.model.base import Base


class JiraIssue(Base):
    __tablename__ = "jira_issues"

    id: Mapped[int] = mapped_column(primary_key=True)
    key: Mapped[str] = mapped_column(String, index=True)
    issue_type: Mapped[str]
    project_key: Mapped[str]
    status: Mapped[str]
    resolution: Mapped[Optional[str]]
    resolution_date: Mapped[Optional[datetime.datetime]]
    summary: Mapped[str]
    created: Mapped[datetime.datetime]
    updated: Mapped[datetime.datetime]
    priority: Mapped[str]
    labels: Mapped[List[str]] = mapped_column(ARRAY(String))
    assignee_email: Mapped[Optional[str]]
    reporter_email: Mapped[str]
    sprint_name: Mapped[Optional[str]]


Index("idx_jira_issues_key", JiraIssue.key)
