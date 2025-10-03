# Task Manager Application

A WhatsApp-integrated task management application for order management, reminders, and financial calculations.

## Features

- **WhatsApp Integration**: All interactions through WhatsApp commands
- **User Management**: Super Admin, Admin, and User roles
- **Task Management**: Daily, monthly, and custom tasks
- **Order Management**: Complete order lifecycle with automatic financial calculations
- **Financial Calculations**: Automatic tax, marketing, and rental cost calculations
- **Redis Caching**: Session management and temporary data storage
- **Real-time Notifications**: WhatsApp reminders and updates

## Tech Stack

- **Backend**: Go (Golang)
- **Database**: PostgreSQL
- **Cache**: Redis
- **WhatsApp Integration**: Custom WhatsApp API (whatsapp-go.sebagja.id)
- **Authentication**: JWT-based
- **Deployment**: Docker

## Quick Start

### Using Docker Compose

1. Clone the repository
2. Copy `.env.example` to `.env` and configure your environment variables
3. Run the application:

```bash
docker-compose up -d
```

### Manual Setup

1. Install dependencies:
```bash
go mod download
```

2. Set up PostgreSQL and Redis
3. Configure environment variables in `.env`
4. Run the application:
```bash
go run cmd/server/main.go
```

## Environment Variables

```env
DATABASE_URL=postgres://user:password@localhost:5432/task_manager
REDIS_URL=redis://localhost:6379
JWT_SECRET=your_jwt_secret
WHATSAPP_API_URL=https://whatsapp-go.sebagja.id
WHATSAPP_USERNAME=your_whatsapp_username
WHATSAPP_PASSWORD=your_whatsapp_password
WHATSAPP_PATH=your_whatsapp_path
SERVER_PORT=8080
SESSION_TIMEOUT=3600
CACHE_TTL=1800
```

## WhatsApp Commands

### General Commands
- `/help` - Show available commands
- `/my_tasks` - View assigned tasks
- `/my_daily_tasks` - View today's daily tasks
- `/my_monthly_tasks` - View this month's tasks
- `/update_progress [task_id] [percentage]` - Update task progress
- `/mark_complete [task_id]` - Mark task as implemented
- `/view_orders` - View related orders
- `/my_report` - View personal financial reports
- `/report_by_date [start_date] [end_date]` - Generate reports by date range

### Admin Commands
- `/add_user [username] [email] [phone] [role]` - Add new user
- `/list_users` - View all users
- `/create_order [customer_name] [total_amount]` - Create new order
- `/assign_task [user_id] [title] [description]` - Assign task to user
- `/create_daily_task [user_id] [title] [description]` - Create daily recurring task
- `/create_monthly_task [user_id] [title] [description]` - Create monthly recurring task
- `/set_tax_rate [percentage]` - Set tax percentage
- `/set_marketing_rate [percentage]` - Set marketing cost percentage
- `/set_rental_rate [percentage]` - Set rental cost percentage
- `/generate_report` - Generate financial reports
- `/daily_report` - Generate daily report
- `/monthly_report` - Generate monthly report

## API Endpoints

### WhatsApp Integration
- `POST /api/whatsapp/webhook` - Receive WhatsApp messages
- `POST /api/whatsapp/send-message` - Send WhatsApp messages
- `POST /api/whatsapp/interactive-session` - Start interactive session
- `PUT /api/whatsapp/session/{session_id}` - Update session
- `DELETE /api/whatsapp/session/{session_id}` - End session

### Cache Management
- `GET /api/cache/session/{session_id}` - Get session data
- `POST /api/cache/session` - Create session
- `PUT /api/cache/session/{session_id}` - Update session
- `DELETE /api/cache/session/{session_id}` - Delete session
- `GET /api/cache/temp-data/{key}` - Get temporary data
- `POST /api/cache/temp-data` - Store temporary data
- `DELETE /api/cache/temp-data/{key}` - Delete temporary data

## Database Schema

The application uses the following main tables:
- `users` - User management
- `tasks` - Task management
- `orders` - Order management
- `reminders` - Reminder system
- `financial_settings` - Financial configuration
- `calculation_history` - Financial calculation history
- `report_queries` - Report generation

## Development

### Project Structure
```
task_manager/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── database/
│   ├── handlers/
│   ├── models/
│   ├── repository/
│   ├── services/
│   └── redis/
├── pkg/
│   └── whatsapp/
├── docker-compose.yml
├── Dockerfile
└── go.mod
```

### Running Tests
```bash
go test ./...
```

### Building for Production
```bash
go build -o task_manager ./cmd/server
```

## License

This project is licensed under the MIT License.
