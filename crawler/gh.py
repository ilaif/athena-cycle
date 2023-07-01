from config import settings
from github import Github, Auth

auth = Auth.Token(settings.GITHUB_TOKEN)

gh = Github(auth=auth)
