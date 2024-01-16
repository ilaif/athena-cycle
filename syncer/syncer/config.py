from datetime import datetime
from typing import List, Optional

from pydantic import BaseSettings, validator


class Settings(BaseSettings):
    DB_USER: str
    DB_PASS: str
    DB_HOST: str
    DB_PORT: int = 5432
    DB_NAME: str
    DATABASE_ECHO: Optional[bool] = False

    GITHUB_TOKEN: Optional[str] = None
    GITHUB_REPOSITORIES: Optional[List[str]] = []
    GITHUB_SYNC_FROM: Optional[datetime] = None

    JIRA_SITE_URL: Optional[str] = None
    JIRA_USERNAME: Optional[str] = None
    JIRA_API_TOKEN: Optional[str] = None
    JIRA_PROJECTS: Optional[List[str]] = []
    JIRA_SYNC_FROM: Optional[datetime] = None
    JIRA_FORCE_RESYNC_FROM: Optional[datetime] = None

    class Config:
        case_sensitive = True
        env_file = ".env"

        @validator("GITHUB_SYNC_FROM", "JIRA_SYNC_FROM", "JIRA_FORCE_RESYNC_FROM", pre=True)
        def time_validate(cls, v):
            return datetime.strptime(v, "%Y-%m-%d")


settings = Settings()
