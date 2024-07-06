from datetime import datetime, timezone

from loguru import logger
from sqlalchemy import func
from sqlalchemy.orm import Session
from schedule import repeat, every

from syncer.adapters import database, jira
from syncer.model.jira_issue import JiraIssue
from syncer.config import settings


@repeat(every(5).minutes)
def sync():
    logger.info("Syncing jira data")

    if settings.JIRA_API_TOKEN is None:
        logger.warning("Jira token not set, skipping sync")
        return

    with Session(database.engine) as session:
        for project_key in settings.JIRA_PROJECTS:
            sync_project(project_key, session)

    logger.info("Finished syncing jira data")


def sync_project(project_key: str, session: Session) -> None:
    proj_logger = logger.bind(project=project_key)
    proj_logger.info("Syncing jira project")

    synced_issue_ids = sync_issues(project_key, session)

    proj_logger.info("Finished syncing jira project", synced_issues=len(synced_issue_ids))


def sync_issues(project_key: str, session: Session) -> list[JiraIssue]:
    proj_logger = logger.bind(project=project_key)
    proj_logger.info("Syncing jira issues")

    start_updated = get_updated_start_for_sync(JiraIssue, project_key, session)

    synced_issue_ids = []
    start_at = 0
    while True:
        result = jira.client.jql(
            jql=f"project = {project_key} ORDER BY updated DESC",
            start=start_at,
            limit=50,
        )

        if len(result["issues"]) == 0:
            break
        start_at += len(result["issues"])

        issues = []
        should_stop = False
        last_updated = None
        for issue in result["issues"]:
            jira_issue = map_jira_issue(issue)
            last_updated = jira_issue["updated"]
            if start_updated and jira_issue["updated"] < start_updated:
                # Stop if we've reached the last updated PR
                should_stop = True
                break
            issues.append(jira_issue)

        proj_logger.debug(
            "Synced chunk of jira issues", size=len(issues), last_updated=last_updated
        )

        database.upsert_by_id_col(JiraIssue, issues, session)

        synced_issue_ids += [issue["id"] for issue in issues]

        if should_stop:
            break

    return synced_issue_ids


def map_jira_issue(issue) -> dict:
    resolution_date = issue["fields"]["resolutiondate"]
    return {
        "id": issue["id"],
        "key": issue["key"],
        "issue_type": issue["fields"]["issuetype"]["name"],
        "project_key": issue["fields"]["project"]["key"],
        "resolution": (
            issue["fields"]["resolution"]["name"] if issue["fields"]["resolution"] else None
        ),
        "resolution_date": datetime.fromisoformat(resolution_date) if resolution_date else None,
        "summary": issue["fields"]["summary"],
        "created": datetime.fromisoformat(issue["fields"]["created"]),
        "updated": datetime.fromisoformat(issue["fields"]["updated"]),
        "priority": issue["fields"]["priority"]["name"],
        "labels": [label for label in issue["fields"]["labels"]],
        "assignee_email": (
            issue["fields"]["assignee"]["emailAddress"] if issue["fields"]["assignee"] else None
        ),
        "status": issue["fields"]["status"]["name"],
        "reporter_email": issue["fields"]["reporter"]["emailAddress"],
        "sprint_name": take_most_relevant_sprint(issue["fields"]["customfield_10020"]),
    }


def take_most_relevant_sprint(sprints):
    if not sprints or len(sprints) == 0:
        return None
    return sprints[-1]["name"]


def get_updated_start_for_sync(model, project_key: str, session: Session):
    start_updated_at_result = (
        session.query(func.max(model.updated)).filter(model.project_key == project_key).first()
    )

    if settings.JIRA_FORCE_RESYNC_FROM is not None:
        return settings.JIRA_FORCE_RESYNC_FROM

    start_updated_at = start_updated_at_result[0]
    if start_updated_at is None:
        start_updated_at = datetime.fromtimestamp(0)
    start_updated_at = start_updated_at.replace(tzinfo=timezone.utc)

    if settings.JIRA_SYNC_FROM is not None:
        start_updated_at = min(settings.JIRA_SYNC_FROM, start_updated_at)

    return start_updated_at
