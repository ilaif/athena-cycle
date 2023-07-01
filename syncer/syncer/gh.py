from github import Github, Auth

from syncer.config import settings

auth = Auth.Token(settings.GITHUB_TOKEN)

client = Github(auth=auth)
