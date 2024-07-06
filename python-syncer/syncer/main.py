import time

import schedule
from loguru import logger

from syncer import github_jobs, jira_jobs, logger as my_logger
from syncer.adapters import database


def main() -> None:
    my_logger.init()
    logger.info("Starting syncer")

    database.init()

    logger.info("Starting first sync")
    jira_jobs.sync()
    github_jobs.sync()

    logger.info("Starting scheduler")
    while True:
        schedule.run_pending()
        time.sleep(1)


if __name__ == "__main__":
    main()
