from datetime import datetime
from typing import List

from loguru import logger
from sqlalchemy import func
from sqlalchemy.orm import Session
from sqlalchemy.dialects.postgresql import insert
from schedule import repeat, every

from syncer import database, gh
from syncer.model.pull_request import PullRequest
from syncer.config import settings


@repeat(every(5).minutes)
def sync_github_data():
    logger.info("Syncing GitHub data")

    with Session(database.engine) as session:
        for repo_name in settings.GITHUB_REPOSITORIES:
            sync_repo(repo_name, session)

    logger.info("Finished syncing GitHub data")


def sync_repo(repo_name: str, session: Session):
    repo_logger = logger.bind(repo=repo_name)
    repo_logger.info("Syncing repository")

    start_updated_at_str = session.query(func.max(PullRequest.updated_at)) \
        .filter(PullRequest.repo == repo_name) \
        .first()
    start_updated_at = datetime.fromtimestamp(0) if start_updated_at_str[0] is None \
        else datetime.strptime(start_updated_at_str[0], '%Y-%m-%d %H:%M:%S')
    if settings.GITHUB_SYNC_FROM is not None:
        start_updated_at = max(settings.GITHUB_SYNC_FROM, start_updated_at)

    values = []
    synced_pulls = 0
    for pull in gh.client.get_repo(repo_name).get_pulls(
        state="all",
        sort="updated_at",
        direction="desc",
    ):
        if pull.state == "closed" and not pull.merged:
            # We don't need to sync closed PRs that are closed and  were not merged
            continue
        if start_updated_at and pull.updated_at < start_updated_at:
            # Stop if we've reached the last updated PR
            break

        values.append({
            "id": pull.id,
            "repo": pull.base.repo.full_name,
            "number": pull.number,
            "title": pull.title,
            "state": pull.state,
            "created_at": pull.created_at,
            "updated_at": pull.updated_at,
            "closed_at": pull.closed_at,
            "merged_at": pull.merged_at,
            "requested_reviewers": [r.login for r in pull.requested_reviewers],
            "requested_teams": [t.name for t in pull.requested_teams],
            "labels": [label.name for label in pull.labels],
            "draft": pull.draft,
            "base": pull.base.ref,
            "user": pull.user.login,
            "merged": pull.merged,
        })

        if len(values) == 50:
            upsert(values, session)
            synced_pulls += len(values)
            values = []

    if len(values) > 0:
        upsert(values, session)
        synced_pulls += len(values)
        values = []

    repo_logger.info("Finished syncing repository", synced_pulls=synced_pulls)


def upsert(values: List[dict], session: Session):
    if len(values) == 0:
        return

    stmt = insert(PullRequest).values(values)
    id_col = "id"
    stmt = stmt.on_conflict_do_update(
        index_elements=[id_col],
        set_={key: stmt.excluded[key] for key in values[0].keys() if key != id_col}
    )
    session.execute(stmt)
    session.commit()
