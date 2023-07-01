from loguru import logger
from sqlalchemy import create_engine
from sqlalchemy.orm import Session

from syncer.model.base import Base
from syncer.config import settings

connect_args = {}
engine = create_engine(settings.DATABASE_URI, echo=settings.DATABASE_ECHO, connect_args=connect_args)


def init():
    logger.info("Initializing database")

    Base.metadata.create_all(engine)


def get_session() -> Session:
    with Session(engine) as session:
        yield session
