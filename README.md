# PHARMA GO BACKEND

## Project Overview
Pharma Go Backend is a project designed to automate various processes for pharmacies, enabling efficient management of pharmacy-related operations. The system streamlines tasks such as inventory management, order processing, and more.

## Environment Setup
To configure the environment for the project, create a `.env` file based on the provided `demo.env` file. Ensure that all necessary environment variables are set up properly.

## Database Migrations

To manage the database schema, you can run the following migration commands:

### 1. Apply Migrations (Create Tables):
Run the command below to apply the migrations and create the necessary tables in the database:

```bash
make migrate-up
```

### 2. Revert Migrations (Drop Tables):
If you need to undo the migrations and drop the created tables, use the following command:

```bash
make migrate-down
```

## Running the Project
To start the project, you can run it using one of the following commands:

Using make:

```bash
make run
```

Or directly with Go:

```bash
go run cmd/app/main.go
```

