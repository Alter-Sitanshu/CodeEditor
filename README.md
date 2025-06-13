# 🧠 Collaborative Code Editor Backend (Go)

A robust and extensible backend service for a real-time collaborative code editor and execution platform, built with **Go**, **PostgreSQL**, and **WebSockets**. This project supports user authentication, email verification, room-based collaboration with role-based access, and live code execution in multiple languages.

---

## 🚀 Features

- 🔐 **JWT-based User Authentication**
- 📧 **Email Verification** for secure signup flow
- 🧑‍🤝‍🧑 **Room Creation and Membership** management
- 💬 **Real-Time Collaboration** with WebSocket (Gorilla)
- 🧪 **Online Code Execution** using [Judge0 API](https://judge0.com)
- 🧰 **CRUD Operations** for Users and Rooms
- 🌐 Support for popular languages:
  - Go
  - Python
  - Java
  - JavaScript

---

## 🛠️ Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/Alter-Sitanshu/CodeEditor.git
cd CodeEditor
```
### 2. Set Up .env
- PORT=:8080

- DB_HOST=localhost
- DB_USER=your_user
- DB_PASS=your_password
- DB_ADDR=your_postgres_url
- DB_NAME=your_db_name

- APP_SECRET=your_jwt_secret
- APP_ISS=your_app_name
- APP_AUD=your_app_audience

- JUDGE0_KEY=your_judge0_api_key

### 3. Start Server
```bash
go run cmd/api/*.go
```
