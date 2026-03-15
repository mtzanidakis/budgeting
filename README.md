# Budgeting App

A secure, fast, and simple Progressive Web App (PWA) for managing income and expenses across multiple family members. Provides both individual budgeting views and aggregate family-level insights.

## Features

### Core Functionality
- **Secure Authentication**: Username/password authentication with bcrypt password hashing
- **Session Management**: HTTP-only cookies with secure session storage
- **Multi-user Support**: Track actions across multiple family members
- **Action Management**: Create, edit, and delete income/expense entries with ownership controls
- **Category Management**: Organize income and expenses with type-specific categories
- **User Profiles**: Update name and change password through profile page
- **Filtering**: Filter actions by user, type (income/expense), and date range
- **Pagination**: Browse through all actions with 20 items per page

### Visualization & Insights
- **Monthly Charts**: Visual breakdown of income vs expenses by month using Chart.js
- **Dashboard Overview**: Quick view of 10 most recent actions
- **Charts Page**: Annual view of monthly income and expense trends

### User Experience
- **Multi-language Support**: Full internationalization with English and Greek translations
- **Dark/Light Theme**: Theme toggle with persistent preference
- **Responsive Design**: Mobile-first design with burger menu for small screens (≤768px)
- **Progressive Web App**: Installable on Android with offline support
- **Inline Actions**: Click on your own actions to edit or delete them directly

### Administration
- **Admin CLI**: Manage users via command-line interface
- **Structured Logging**: JSON-formatted logs with request tracking
- **Docker Support**: Production and development Docker Compose configurations

## Tech Stack

### Backend
- **Go 1.26+** with Chi router
- **SQLite3** database with foreign key constraints
- **bcrypt** for password hashing
- **slog** for structured JSON logging
- Environment configuration via [caarlos0/env](https://github.com/caarlos0/env)

### Frontend
- Vanilla JavaScript (no build step required)
- Custom CSS with CSS Variables for theming
- Chart.js for data visualization
- Service Worker for PWA functionality
- Responsive mobile-first design with burger menu

## Quick Start

### Prerequisites
- Go 1.26 or later
- Make (optional, for convenience)
- Docker & Docker Compose (for containerized deployment)

### Local Development

1. **Clone the repository**
```bash
git clone <repository-url>
cd budgeting
```

2. **Set environment variables**
```bash
export SESSION_SECRET="your-secret-key-here"
export DATABASE_PATH="./data/budgeting.db"
export PORT="8080"
```

3. **Build and run**
```bash
make run
```

Or manually:
```bash
CGO_ENABLED=1 go build -o bin/server ./cmd/server
CGO_ENABLED=1 go build -o bin/cli ./cmd/cli
./bin/server
```

4. **Create your first user**
```bash
# In a separate terminal
make cli ARGS="user:add -username admin -name Admin"
```

Or manually:
```bash
./bin/cli user:add -username admin -name Admin
```

5. **Access the app**
Open your browser to `http://localhost:8080`

### Docker Deployment

Two Docker Compose configurations are provided:

#### Production (`docker-compose.yml`)
```bash
# Create a .env file
echo "SESSION_SECRET=$(openssl rand -base64 32)" > .env

# Build and run
docker compose up -d

# Create users via CLI
docker compose exec budgeting-app /app/cli user:add -username admin -name Admin

# View logs
docker compose logs -f
```

#### Development (`docker-compose.dev.yml`)
```bash
# Run development setup with exposed port 8080
docker compose -f docker-compose.dev.yml up -d
```

The production configuration keeps the port unexposed for security, while the development configuration exposes port 8080 for local access.

## CLI Commands

### User Management

#### Add User
```bash
./bin/cli user:add -username <username> -name <display-name>
```
- Prompts for password (6+ characters required)
- Leave password empty to generate a random one

#### Edit User
```bash
./bin/cli user:edit -username <username> [-name <new-name>]
```
- Prompts for new password (optional)

#### Delete User
```bash
./bin/cli user:delete -username <username>
```
- Requires confirmation

#### List Users
```bash
./bin/cli user:list
```
- Displays all users in table format

### Actions Query

```bash
./bin/cli actions:query -username <username> [-type income|expense] [-date-range YYYYMMDD-YYYYMMDD]
```

**Examples:**
```bash
# All actions for a user
./bin/cli actions:query -username john

# Only expenses
./bin/cli actions:query -username john -type expense

# Actions in date range
./bin/cli actions:query -username john -date-range 20240101-20241231
```

## API Endpoints

### Authentication
- `POST /api/login` - Login with username/password
- `POST /api/logout` - Logout and clear session
- `GET /api/me` - Get current user session info

### Actions
- `GET /api/actions` - List actions (with filters)
  - Query params: `username`, `type`, `date_from`, `date_to`, `limit`, `offset`
  - Returns paginated response when `offset` is provided
- `POST /api/actions` - Create new action
  - Body: `{"type": "income|expense", "date": "YYYY-MM-DD", "description": "...", "amount": 0.00, "category_id": 1}` (category_id is required)
- `PUT /api/actions/{id}` - Update action (requires ownership)
  - Body: `{"type": "income|expense", "date": "YYYY-MM-DD", "description": "...", "amount": 0.00, "category_id": 1}` (category_id is required)
- `DELETE /api/actions/{id}` - Delete action (requires ownership)

### Categories
- `GET /api/categories` - List all categories
  - Query params: `action_type` (optional, filter by income/expense)
- `POST /api/categories` - Create new category
  - Body: `{"description": "...", "action_type": "income|expense"}`
- `PUT /api/categories/{id}` - Update category
  - Body: `{"description": "...", "action_type": "income|expense"}`
- `DELETE /api/categories/{id}` - Delete category (sets related actions' category_id to NULL)

### Charts
- `GET /api/charts/monthly` - Get monthly income/expense summary
  - Query params: `year` (optional, defaults to current year)

### Users
- `GET /api/users` - List all users (for filter dropdown)
- `PUT /api/profile` - Update user profile (name and/or password)

### Configuration
- `GET /api/config` - Get app configuration (e.g., currency symbol)

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | Server port |
| `DATABASE_PATH` | No | `./data/budgeting.db` | SQLite database file path |
| `SESSION_SECRET` | **Yes** | - | Secret key for session encryption |
| `LOG_LEVEL` | No | `info` | Logging level |
| `CURRENCY` | No | `€` | Currency symbol to display |

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Categories Table
```sql
CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    description TEXT NOT NULL,
    action_type TEXT NOT NULL CHECK(action_type IN ('income', 'expense')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Actions Table
```sql
CREATE TABLE actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('income', 'expense')),
    date DATE NOT NULL,
    description TEXT NOT NULL,
    amount REAL NOT NULL,
    category_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
);
```

## Testing

Run all tests:
```bash
make test
```

Or manually:
```bash
go test -v ./...
```

## Project Structure

```
.
├── cmd/
│   ├── server/          # Main server application
│   │   ├── main.go
│   │   └── frontend/    # Frontend assets (embedded)
│   │       ├── css/
│   │       ├── js/
│   │       ├── index.html
│   │       ├── manifest.json
│   │       └── sw.js
│   └── cli/             # Admin CLI tool
│       └── main.go
├── internal/
│   ├── auth/            # Authentication & password hashing
│   ├── config/          # Configuration management
│   ├── database/        # Database operations
│   ├── handlers/        # HTTP request handlers
│   ├── middleware/      # HTTP middleware (auth, logging)
│   └── models/          # Data models
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Security Notes

- **Password Security**: Passwords are hashed using bcrypt before storage
- **Session Security**: Sessions use HTTP-only cookies (not accessible via JavaScript)
- **Authentication**: All API endpoints except `/api/login` and `/api/config` require authentication
- **Ownership Controls**: Users can only edit/delete their own actions (enforced at database level)
- **Data Integrity**: Foreign key constraints prevent orphaned data
- **Input Validation**: All user-submitted data is validated on both frontend and backend
- **CSRF Protection**: SameSite=Strict cookies prevent cross-site request forgery

## PWA Installation

### Android
1. Open the app in Chrome
2. Tap the menu (⋮) and select "Install app" or "Add to Home screen"
3. The app will be installed and can be launched like a native app

### Features
- Works offline (with cached UI)
- App icon on home screen
- Standalone window (no browser UI)

## Makefile Commands

```bash
make build          # Build server and CLI
make run            # Build and run server
make test           # Run tests
make clean          # Clean build artifacts
make cli            # Run CLI with arguments
make docker-build   # Build Docker image
make docker-up      # Start Docker containers
make docker-down    # Stop Docker containers
make docker-logs    # View Docker logs
```

### User Management Shortcuts
```bash
make user-add -username john -name "John Doe"
make user-list
make user-edit -username john -name "John Smith"
make user-delete -username john
make actions-query -username john
```

## Future Enhancements

- CSV/PDF exports
- Role-based permissions
- Recurring transactions
- Budget limits & alerts
- Email notifications

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
