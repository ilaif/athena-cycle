"""add stats columns to pr

Revision ID: 6f645089b860
Revises: 0665a26859b5
Create Date: 2024-01-16 01:38:07.347896

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = "6f645089b860"
down_revision = "0665a26859b5"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column("pull_requests", sa.Column("additions", sa.Integer(), nullable=True))
    op.add_column("pull_requests", sa.Column("deletions", sa.Integer(), nullable=True))
    op.add_column("pull_requests", sa.Column("changed_files", sa.Integer(), nullable=True))


def downgrade() -> None:
    op.drop_column("pull_requests", "changed_files")
    op.drop_column("pull_requests", "deletions")
    op.drop_column("pull_requests", "additions")
