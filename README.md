# <img src="frontend/src/assets/logo.png" alt="UniBase Logo" width="48" valign="middle" /> UniBase

UniBase provides a unified, web-based interface to connect to and interact with multiple database systems, including PostgreSQL, Microsoft SQL Server, MySQL, MongoDB, and Redis.

## 🚀 Key Features

- **Multi-Database Support**: Connect to and manage multiple different database systems from a single application.
- **Query Editor**: Write and execute queries using a fully-featured integrated text editor.
- **Interactive Metadata Explorer**: Deep-dive into your database structure with a dynamic tree view for tables, collections, and schemas.
- **Connection Management**: Easily add, test, save, and manage multiple database connections.

---

## 🏁 Getting Started

### Prerequisites

Ensure you have the following installed on your machine:

- **Node.js**: v18.x or higher
- **Go**: v1.21.x or higher
- **Angular CLI**: v20.3.x (`npm install -g @angular/cli@20.3.10`)

### Installation & Running

1. **Clone the Repository**

   ```bash
   git clone https://github.com/your-username/unibase.git
   cd UniBase
   ```

2. **Launch the Application**

   The quickest way to start UniBase is to use the provided launch scripts. These scripts will automatically build the frontend into static files and then serve the entire application from the Go backend.

   **On Windows:**
   Open a PowerShell terminal and run:

   ```powershell
   ./run.ps1
   ```

   **On macOS/Linux:**
   Open a terminal and run:

   ```bash
   ./run.sh
   ```

   Once the application is running, open your web browser and navigate to `http://localhost:5000` to start adding database connections.

---

## ℹ️ Important Details

- **How it works**: The launch script builds the Angular frontend placing the artifacts in a structure that the Go backend seamlessly serves while securing database connections.

---

## 📄 License

This project is licensed under the MIT License.
