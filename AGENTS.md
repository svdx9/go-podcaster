This repository does not use database migrations.
Schema creation is handled at startup in internal/db via createSchema when the database is not initialized.
