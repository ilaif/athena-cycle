import sys

from loguru import logger


def init():
    logger.remove()
    logger.add(sys.stdout, format=format, colorize=True)


def format(r) -> str:
    extra = " ".join([f"{k}={v}" for k, v in r["extra"].items()])
    if extra != "":
        extra = f"[{extra}]"

    return (
        f"<green>{r['time']:YYYY-MM-DD HH:mm:ss.SSS}</green> | "
        f"<level>{r['level']: <8}</level> | "
        f"<cyan>{r['name']}</cyan>:<cyan>{r['function']}</cyan>:<cyan>{r['line']}</cyan> | "
        f"<level>{r['message']}</level> "
        f"{extra}\n"
    )
