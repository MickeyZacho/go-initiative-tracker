# Go Initiative Tracker

A web-based initiative tracker for tabletop role-playing games, built with Go (backend) and React (frontend).

## Features

-   Add, edit, and delete characters
-   Sort characters by initiative
-   Track active characters in encounters
-   Modern React UI with Material UI
-   Database-backed persistence using PostgreSQL

## Monorepo Structure

```
go-initiative-tracker/
  backend/      # Go backend (API, templates, static assets)
  front/        # React/Vite frontend
  README.md
  .gitignore
```

### Backend Setup

**Prerequisites:**

-   Go 1.21 or later
-   PostgreSQL database
-   `air` for hot reloading (optional)

**Install dependencies:**

```bash
cd backend
go mod tidy
```

**Set up the database:**

-   Create a PostgreSQL database using the credentials in `.env`.
-   Run the SQL schema to initialize the database.

**Start the server:**

```bash
go run main.go
```

**Run backend tests:**

```bash
go test ./...
```

### Frontend Setup

**Prerequisites:**

-   Node.js 18+

**Install dependencies:**

```bash
cd front/tracker
npm install
```

**Start the frontend dev server:**

```bash
npm run dev
```

**Open your browser:**

-   Backend: `http://localhost:8080`
-   Frontend: `http://localhost:5173`

## Configuration

Edit the `.env` file in `backend/` to configure database credentials:

```env
USER=postgres
PASSWORD=yourpassword
DBNAME=initiative_tracker
SSLMODE=disable
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This project is licensed under the MIT License.
