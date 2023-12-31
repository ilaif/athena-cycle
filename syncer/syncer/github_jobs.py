from datetime import datetime

from loguru import logger
from sqlalchemy import func
from sqlalchemy.orm import Session
from schedule import repeat, every

from syncer.adapters import database, gh
from syncer.model.pull_request import PullRequest
from syncer.config import settings
from syncer.model.review import Review
from syncer.utils import chunked_iterable


@repeat(every(5).minutes)
def sync():
    logger.info("Syncing GitHub data")

    logger.warning(f"GitHub token: {settings.GITHUB_TOKEN}")
    if settings.GITHUB_TOKEN is None:
        logger.warning("GitHub token not set, skipping sync")
        return

    if len(settings.GITHUB_REPOSITORIES) == 0:
        logger.warning("No GitHub repositories set, skipping sync")
        return

    with Session(database.engine) as session:
        for repo_name in settings.GITHUB_REPOSITORIES:
            sync_repo(repo_name, session)

    logger.info("Finished syncing GitHub data")


def sync_repo(repo_name: str, session: Session) -> None:
    repo_logger = logger.bind(repo=repo_name)
    repo_logger.info("Syncing repository")

    synced_pr_ids = sync_pull_requests(repo_name, session)

    repo_logger.info("Finished syncing repository", synced_prs=len(synced_pr_ids))


def sync_pull_requests(repo_name: str, session: Session) -> list[PullRequest]:
    repo_logger = logger.bind(repo=repo_name)
    repo_logger.info("Syncing pull requests")

    start_updated_at = get_update_at_start_for_sync(PullRequest, repo_name, session)

    chunked_prs = chunked_iterable(gh.client.get_repo(repo_name).get_pulls(
        state="all", sort="updated_at", direction="desc",
    ), size=50)
    synced_pr_ids = []
    for chunk in chunked_prs:
        repo_logger.debug("Syncing chunk of pull requests", chunk_size=len(chunk))

        prs = []
        should_stop = False
        for pr in chunk:
            if start_updated_at and pr.updated_at < start_updated_at:
                # Stop if we've reached the last updated PR
                should_stop = True
                break
            prs.append(pr)

        database.upsert_by_id_col(PullRequest, [gh_pr_to_dict(pr) for pr in prs], session)

        for pr in prs:
            reviews = [gh_review_to_dict(review, pr) for review in pr.get_reviews()]
            repo_logger.debug("Syncing reviews for pull request",
                              pr=pr.number, reviews=len(reviews))
            database.upsert_by_id_col(Review, reviews, session)

        synced_pr_ids += [pr.id for pr in prs]

        if should_stop:
            break

    return synced_pr_ids


def gh_pr_to_dict(pr) -> dict:
    return {
        "id": pr.id,
        "repo": pr.base.repo.full_name,
        "number": pr.number,
        "title": pr.title,
        "state": pr.state,
        "created_at": pr.created_at,
        "updated_at": pr.updated_at,
        "closed_at": pr.closed_at,
        "merged_at": pr.merged_at,
        "requested_reviewers": [r.login for r in pr.requested_reviewers],
        "requested_teams": [t.name for t in pr.requested_teams],
        "labels": [label.name for label in pr.labels],
        "draft": pr.draft,
        "base": pr.base.ref,
        "username": pr.user.login,
        "merged": pr.merged,
        "head_ref": pr.head.ref,
    }


def gh_review_to_dict(review, pr) -> dict:
    return {
        "id": review.id,
        "repo": pr.base.repo.full_name,
        "pull_request_id": pr.id,
        "username": review.user.login,
        "state": review.state,
        "submitted_at": review.submitted_at,
        "commit_id": review.commit_id,
        "body": review.body,
    }


def get_update_at_start_for_sync(model, repo_name: str, session: Session):
    start_updated_at_result = session.query(func.max(model.updated_at)) \
        .filter(model.repo == repo_name) \
        .first()

    start_updated_at = start_updated_at_result[0]
    if start_updated_at is None:
        start_updated_at = datetime.fromtimestamp(0)

    if settings.GITHUB_SYNC_FROM is not None:
        start_updated_at = min(settings.GITHUB_SYNC_FROM, start_updated_at)

    return start_updated_at
