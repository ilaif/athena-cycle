import time
import schedule
from loguru import logger
from jobs import sync_github_data
from database import init_db
from logger import init_logger


def main() -> None:
    init_logger()

    logger.info("Starting crawler")

    init_db()

    schedule.every(5).minutes.do(sync_github_data)

    logger.info("Starting first sync")
    sync_github_data()

    logger.info("Starting scheduler")
    while True:
        schedule.run_pending()
        time.sleep(1)


if __name__ == "__main__":
    main()
