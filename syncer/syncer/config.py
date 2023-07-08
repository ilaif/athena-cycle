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

    class Config:
        case_sensitive = True
        env_file = ".env"

        @validator('GITHUB_SYNC_FROM', pre=True)
        def time_validate(cls, v):
            return datetime.strptime(v, '%Y-%m-%d')


settings = Settings()
