from loguru import logger
from sqlalchemy import create_engine
from sqlalchemy.orm import Session

from syncer.model.base import Base
from syncer.config import settings

connect_args = {}
db_uri = f"postgresql://{settings.DB_USER}:{settings.DB_PASS}@{settings.DB_HOST}:5432/{settings.DB_NAME}"
engine = create_engine(db_uri, echo=settings.DATABASE_ECHO, connect_args=connect_args)


def init():
    logger.info("Initializing database")

    Base.metadata.create_all(engine)


def get_session() -> Session:
    with Session(engine) as session:
        yield session
