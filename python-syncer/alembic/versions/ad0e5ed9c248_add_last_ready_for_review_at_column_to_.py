"""add last_ready_for_review_at column to pr

Revision ID: ad0e5ed9c248
Revises: d48bcfaea4a8
Create Date: 2024-01-14 00:34:41.571638

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = "ad0e5ed9c248"
down_revision = "d48bcfaea4a8"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column("pull_requests", sa.Column("last_ready_for_review_at", sa.DateTime(), nullable=True))
    op.drop_column("pull_requests", "last_converted_to_draft_at")


def downgrade() -> None:
    op.add_column(
        "pull_requests", sa.Column("last_converted_to_draft_at", sa.DateTime(), nullable=True)
    )
    op.drop_column("pull_requests", "last_ready_for_review_at")
