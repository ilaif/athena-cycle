"""add first_reviewed_at column to pr

Revision ID: 0665a26859b5
Revises: ad0e5ed9c248
Create Date: 2024-01-16 00:41:57.525515

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '0665a26859b5'
down_revision = 'ad0e5ed9c248'
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column('pull_requests', sa.Column('first_reviewed_at', sa.DateTime(), nullable=True))


def downgrade() -> None:
    op.drop_column('pull_requests', 'first_reviewed_at')
