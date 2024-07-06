"""add last_convert_to_draft_event column to pr

Revision ID: d48bcfaea4a8
Revises:
Create Date: 2024-01-14 00:15:40.178960

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = "d48bcfaea4a8"
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column(
        "pull_requests", sa.Column("last_converted_to_draft_at", sa.DateTime(), nullable=True)
    )


def downgrade() -> None:
    op.drop_column("pull_requests", "last_converted_to_draft_at")
