FROM python:3.11-alpine

ARG POETRY_VERSION=1.7.1

RUN apk add --no-cache \
    curl \
    gcc \
    libressl-dev \
    musl-dev \
    libffi-dev && \
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --profile=minimal && \
    source $HOME/.cargo/env && \
    pip install --no-cache-dir poetry==${POETRY_VERSION}

RUN pip install poetry

WORKDIR /code
COPY poetry.lock pyproject.toml /code/

RUN poetry config virtualenvs.create false
RUN poetry install --no-interaction --no-ansi

COPY . /code

CMD [ "python", "-m", "syncer.main" ]
