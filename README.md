# MySQL Database Backup Tool

This Go application backs up all tables in a MySQL database, excluding specified tables, and compresses the backup files into a zip archive. The application uses `mysqldump` for dumping the tables and creates a zip file containing the backups.

## Features

- Backs up all tables in a MySQL database, excluding specified tables.
- Creates individual SQL dump files for each table.
- Uses a worker pool for parallel backups.
- Compresses the backup files into a zip archive.

## Prerequisites

- Go (version 1.18 or later)
- MySQL database
- `mysqldump` command-line tool
- A `.env` file with your MySQL database credentials:

```env
DB_USERNAME=your_db_username
DB_PASSWORD=your_db_password
DB_HOST=your_db_host
DB_PORT=your_db_port
DB_NAME=your_db_name
