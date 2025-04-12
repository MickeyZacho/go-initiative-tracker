# Go Initiative Tracker

A web-based initiative tracker for tabletop role-playing games, built with Go.

## Features
- Add, edit, and delete characters.
- Sort characters by initiative.
- Track active characters in encounters.
- Database-backed persistence using PostgreSQL.

## Getting Started

### Prerequisites
- Go 1.21 or later
- PostgreSQL database
- `air` for hot reloading (optional)

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/go-initiative-tracker.git
   cd go-initiative-tracker
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Set up the database:
   - Create a PostgreSQL database using the credentials in `.env`.
   - Run the SQL schema to initialize the database.

4. Start the server:
   ```bash
   go run main.go
   ```

5. Open your browser and navigate to `http://localhost:8080`.

### Running Tests
Run the following command to execute the test suite:
```bash
go test ./...
```

### Configuration
Edit the `.env` file to configure database credentials:
```env
USER=postgres
PASSWORD=yourpassword
DBNAME=initiative_tracker
SSLMODE=disable
```

## Project Structure
- `main.go`: Entry point and HTTP handlers.
- `character_dao.go`: Data access layer for characters and encounters.
- `templates/`: HTML templates for the web interface.
- `static/`: Static assets (CSS, JavaScript).
- `main_test.go`: Unit tests for handlers and database interactions.

## Contributing
Contributions are welcome! Please open an issue or submit a pull request.

## License
This project is licensed under the MIT License.
