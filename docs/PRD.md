# Family Budgeting App -- PRD

## 1. Overview

### Product Name

Budgeting

### Purpose

A secure, fast, and simple **Progressive Web App (PWA)** for managing
income and expenses across multiple family members. The app provides
both **individual budgeting views** and **aggregate family-level
insights**.

User lifecycle (create/edit/delete) is handled **exclusively via Admin
CLI commands**, while day-to-day budgeting is done through the
authenticated web UI.

### Target Users

-   Families who want a centralized budgeting tool
-   Tech-savvy users comfortable with CLI-based admin tasks
-   Privacy-conscious users (local-first, SQLite, minimal dependencies)

------------------------------------------------------------------------

## 2. Goals & Non-Goals

### Goals

-   Simple, intuitive budgeting experience
-   Secure authentication-only access
-   Fast load times (PWA + server-side static assets)
-   Clear separation between admin (CLI) and users (UI)
-   Accurate per-user and family-wide financial summaries

### Non-Goals (v1)

-   No cloud sync / multi-tenant SaaS
-   No third-party auth providers
-   No bank integrations
-   No recurring transactions (future version)

------------------------------------------------------------------------

## 3. Functional Requirements

### 3.1 Authentication

-   Mandatory login screen
-   Username + password authentication
-   No anonymous access
-   Sessions secured via HTTP-only cookies
-   Logout functionality available via user menu

------------------------------------------------------------------------

### 3.2 Actions (Income / Expense)

Each **Action** represents a financial event.

Fields: - Type: `income` or `expense` - Date (user selectable, defaults
to today) - Description (free text) - Amount (positive decimal number)

Rules: - Expense is selected by default - Amount is always stored as
positive; type determines semantics - Users can only create actions for
themselves

------------------------------------------------------------------------

### 3.3 Main Screen (Dashboard)

Default view after login.

#### Content

-   List **20 most recent actions**, sorted by `date DESC`
-   Includes actions from **all users**

#### Filters

-   User (by `display_name`)
-   Type: Income / Expense
-   Date range (from -- to) via date pickers

Filters should be combinable.

------------------------------------------------------------------------

### 3.4 Add Action Flow

-   Floating **"+" button**
-   Opens action creation screen
-   Fields:
    -   Type (dropdown: Income / Expense)
    -   Date (date picker, today preselected)
    -   Description
    -   Amount
-   User id is added automatically from session
-   Submit validates all fields
-   Redirect back to main screen

------------------------------------------------------------------------

### 3.5 User Menu (Top Right)

-   Display logged-in user name
-   Logout option
-   Placeholder for future features (settings, reports, exports)

------------------------------------------------------------------------

## 4. Admin CLI Requirements

The backend must expose a **CLI tool** for administrative tasks.

### 4.1 User Management Commands

#### Add User

-   Parameters:
    -   `-username`
    -   `-name`
-   Password handling:
    -   Minimum 16 characters
    -   Prompt interactively for password
    -   If left empty → generate random password
    -   Display generated password once

#### Edit User

-   Edit display name and/or password
-   Password prompt behavior same as add

#### Delete User

-   Requires username
-   Confirmation prompt

#### List Users

-   Output in **table format**
-   Columns:
    -   ID
    -   Username
    -   Name
    -   Created At
    -   Updated At

------------------------------------------------------------------------

### 4.2 Actions Query Command

Command to display actions for a user.

Parameters: - `-username` (mandatory) - `-type` (optional:
income\|expense) - `-date-range` (optional)

Date range format:

    YYYYMMDD-YYYYMMDD

Output: - Table format - Columns: - Date - Type - Description - Amount -
Created At

------------------------------------------------------------------------

## 5. Technical Requirements

## 5.1 Backend

### Language & Framework

-   Go (latest stable)
-   Chi router (latest version)
-   Middleware-based architecture

### Configuration

-   All configuration via environment variables
-   Use: https://github.com/caarlos0/env

### Authentication & Security

-   Passwords stored using **bcrypt**
-   Auth middleware on all routes
-   Secure cookies
-   No unauthenticated endpoints except login

### Logging

-   Use `slog`
-   JSON format only
-   Access logs for **all HTTP paths**
-   Each access log must include:
    -   HTTP method
    -   Path
    -   Status code
    -   Duration
    -   Username (if authenticated)

### Database

-   SQLite3
-   Foreign keys enabled
-   Auto-migration on startup

### Database Schema

#### Users

  Field          Type
  -------------- -------------
  id             INTEGER PK
  username       TEXT UNIQUE
  password       TEXT
  name           TEXT
  created_at     DATETIME
  updated_at     DATETIME

#### Categories

  Field         Type
  ------------- -------------------------
  id            INTEGER PK
  description   TEXT
  action_type   ENUM(income, expense)
  created_at    DATETIME
  updated_at    DATETIME

#### Actions

  Field         Type
  ------------- ---------------------------
  id            INTEGER PK
  user_id       INTEGER FK(users.id)
  type          ENUM(income, expense)
  date          DATE
  description   TEXT
  amount        REAL
  category_id   INTEGER FK(categories.id)
  created_at    DATETIME
  updated_at    DATETIME

------------------------------------------------------------------------

## 5.2 Frontend

### Stack

-   Served as static assets from Go backend
-   Shadcn UI components
-   Responsive design (mobile-first)
-   Dark / Light theme toggle

### Progressive Web App

-   Installable on Android
-   App manifest + service worker
-   Offline shell (no offline writes in v1)

------------------------------------------------------------------------

## 6. Performance & UX

-   App must load under 1s on modern devices
-   Avoid unnecessary API calls
-   Optimistic UI updates where applicable
-   Clear empty states (no actions yet)

------------------------------------------------------------------------

## 7. Deployment

-   Docker-based deployment
-   Single container:
    -   Go backend
    -   Embedded frontend assets
-   SQLite database stored via Docker volume
-   Makefile for build & run commands

------------------------------------------------------------------------

## 8. Testing

-   Unit tests for backend logic
-   Integration tests for API endpoints
-   Manual testing for PWA functionality
-   CLI commands tested via integration tests

------------------------------------------------------------------------

## 9. Future Enhancements (Out of Scope)

-   ~~Monthly summaries & charts~~ ✅ Implemented
-   ~~Categories for expenses~~ ✅ Implemented (income & expense categories)
-   CSV / PDF exports
-   Role-based permissions
-   Recurring transactions
-   Family budget limits & alerts

------------------------------------------------------------------------

## 10. Success Criteria

-   Users can log in and add actions without confusion
-   Admin can fully manage users without UI
-   Logs are structured, searchable, and user-attributed
-   App is installable and usable on mobile devices
-   No unauthenticated access possible

------------------------------------------------------------------------

## 11. Notes for LLM Implementation

-   Favor clarity and maintainability over cleverness
-   Use latest stable dependencies
-   Follow idiomatic Go patterns
-   Ensure deterministic CLI behavior
-   Validate all user inputs server-side
