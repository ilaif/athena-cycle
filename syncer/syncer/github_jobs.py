from datetime import datetime
from multiprocessing.pool import ThreadPool

from loguru import logger, _Logger
from sqlalchemy import func
from sqlalchemy.orm import Session
from schedule import repeat, every
from github import PullRequest as GitHubPullRequest, PullRequestReview as GitHubPullRequestReview

from syncer.adapters import database, gh
from syncer.model.pull_request import PullRequest
from syncer.config import settings
from syncer.model.review import Review
from syncer.utils import chunked_iterable


@repeat(every(5).minutes)
def sync():
    logger.info("Syncing GitHub data")

    if settings.GITHUB_TOKEN is None:
        logger.warning("GitHub token not set, skipping sync")
        return

    if len(settings.GITHUB_REPOSITORIES) == 0:
        logger.warning("No GitHub repositories set, skipping sync")
        return

    logger.info("Github repositories to sync", repositories=settings.GITHUB_REPOSITORIES)

    for repo_name in settings.GITHUB_REPOSITORIES:
        sync_repo(repo_name)

    logger.info("Finished syncing GitHub data")


def sync_repo(repo_name: str) -> None:
    repo_logger = logger.bind(repo=repo_name)
    repo_logger.info("Syncing repository")

    synced_pr_ids = sync_pull_requests(repo_name)

    repo_logger.info("Finished syncing repository", synced_prs=len(synced_pr_ids))


def sync_pull_requests(repo_name: str) -> list[PullRequest]:
    repo_logger = logger.bind(repo=repo_name)
    start_updated_at = get_update_at_start_for_sync(PullRequest, repo_name)
    repo_logger.info("Syncing pull requests from", start_updated_at=start_updated_at)
    repo = gh.client.get_repo(repo_name)
    chunked_prs = chunked_iterable(
        repo.get_pulls(
            state="all",
            sort="updated_at",
            direction="desc",
        ),
        size=100,
    )

    def sync_prs(pr):
        if start_updated_at and pr.updated_at < start_updated_at:
            # Stop if we've reached the last updated PR
            return True, None
        pr_logger = repo_logger.bind(pr=pr.number)
        pr_dict = gh_pr_to_dict(pr)
        process_pr_events(pr_logger, pr, pr_dict)
        review_dicts = [gh_review_to_dict(review, pr) for review in pr.get_reviews()]
        repo_logger.info(
            "Syncing pull request", created_at=pr.created_at, reviews=len(review_dicts)
        )
        if len(review_dicts) > 0:
            pr_dict["first_reviewed_at"] = review_dicts[0]["submitted_at"]
        with Session(database.engine) as session:
            database.upsert_by_id_col(PullRequest, [pr_dict], session)
            database.upsert_by_id_col(Review, review_dicts, session)
        return False, pr_dict["id"]

    synced_pr_ids = []
    with ThreadPool(5) as p:
        should_stop = False
        for chunk in chunked_prs:
            repo_logger.debug("Syncing chunk of pull requests", chunk_size=len(chunk))
            results = p.map(sync_prs, chunk)
            for should_stop, pr_id in results:
                if should_stop:
                    should_stop = True
                synced_pr_ids.append(pr_id)
            if should_stop:
                break

    return synced_pr_ids


def gh_pr_to_dict(pr: GitHubPullRequest.PullRequest) -> dict:
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
        "additions": pr.additions,
        "deletions": pr.deletions,
        "changed_files": pr.changed_files,
    }


def gh_review_to_dict(review: GitHubPullRequestReview, pr: GitHubPullRequest) -> dict:
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


def get_update_at_start_for_sync(model, repo_name: str):
    with Session(database.engine) as session:
        start_updated_at_result = (
            session.query(func.max(model.updated_at)).filter(model.repo == repo_name).first()
        )

        start_updated_at = start_updated_at_result[0]
        if start_updated_at is None:
            start_updated_at = datetime.fromtimestamp(0)

        if settings.GITHUB_SYNC_FROM is not None:
            start_updated_at = min(settings.GITHUB_SYNC_FROM, start_updated_at)

        return start_updated_at


def process_pr_events(pr_logger: _Logger, pr: GitHubPullRequest, pr_dict: dict):
    last_ready_for_review_event = None
    all_pr_events = [event for event in pr.get_issue_events().reversed]
    pr_logger.info("Fetched events for pull request", pr=pr.number, events=len(all_pr_events))
    for event in all_pr_events:
        if event.event == "ready_for_review":
            last_ready_for_review_event = event
            break
    if last_ready_for_review_event:
        pr_logger.debug(
            "Found last ready for review event",
            pr=pr.number,
            event=last_ready_for_review_event,
        )
        pr_dict["last_ready_for_review_at"] = last_ready_for_review_event.created_at
    else:
        pr_logger.debug("No ready for review event found, using pr's created_at")
        pr_dict["last_ready_for_review_at"] = pr.created_at
