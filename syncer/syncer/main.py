import time

import schedule
from loguru import logger

from syncer import jobs, database, logger as my_logger


def main() -> None:
    my_logger.init()
    logger.info("Starting syncer")

    database.init()

    logger.info("Starting first sync")
    jobs.sync_github_data()

    logger.info("Starting scheduler")
    while True:
        schedule.run_pending()
        time.sleep(1)


if __name__ == "__main__":
    main()
