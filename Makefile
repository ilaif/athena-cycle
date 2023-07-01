setup-ide:
	curl -sSL https://install.python-poetry.org | python3 -
	brew install act

ci-pr-local:
	act --container-architecture linux/amd64 pull_request
