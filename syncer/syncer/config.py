from datetime import datetime
from typing import List, Optional

from pydantic import BaseSettings, validator


class Settings(BaseSettings):
    DB_USER: str
    DB_PASS: str
    DB_HOST: str
    DB_NAME: str
    DATABASE_ECHO: Optional[bool] = False

    GITHUB_TOKEN: str
    GITHUB_REPOSITORIES: List[str]
    GITHUB_SYNC_FROM: Optional[datetime] = None

    JIRA_SITE_URL: str
    JIRA_USERNAME: str
    JIRA_API_TOKEN: str
    JIRA_PROJECTS: List[str]
    JIRA_SYNC_FROM: Optional[datetime] = None
    JIRA_FORCE_RESYNC_FROM: Optional[datetime] = None

    class Config:
        case_sensitive = True
        env_file = ".env"

        @validator('GITHUB_SYNC_FROM', 'JIRA_SYNC_FROM', 'JIRA_FORCE_RESYNC_FROM', pre=True)
        def time_validate(cls, v):
            return datetime.strptime(v, '%Y-%m-%d')


settings = Settings()
