from atlassian import Jira

from syncer.config import settings

client = Jira(
    url=settings.JIRA_SITE_URL,
    username=settings.JIRA_USERNAME,
    password=settings.JIRA_API_TOKEN,
    cloud=True,
)
