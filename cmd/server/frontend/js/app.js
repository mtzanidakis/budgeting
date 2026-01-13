// State management
const state = {
    user: null,
    language: localStorage.getItem('language') || 'en',
    actions: [],
    users: [],
    categories: [],
    theme: localStorage.getItem('theme') || 'light',
    currency: '€',
    currentPage: 'dashboard',
    allActionsPage: {
        actions: [],
        filters: {
            mode: 'month',
            month: new Date().getMonth(),
            year: new Date().getFullYear(),
            username: '',
            type: '',
            date_from: '',
            date_to: '',
            search: ''
        },
        pagination: {
            currentPage: 1,
            perPage: 20,
            totalActions: 0,
            totalPages: 0
        }
    },
    chartsPage: {
        year: new Date().getFullYear(),
        chartData: null,
        chartInstance: null
    },
    profilePage: {
        message: null,
        isSubmitting: false
    },
    datePicker: {
        visible: false,
        targetInput: null,
        currentMonth: new Date().getMonth(),
        currentYear: new Date().getFullYear()
    },
    editActionModal: {
        visible: false,
        action: null
    },
    categoriesPage: {
        categories: [],
        editCategoryModal: {
            visible: false,
            category: null
        }
    },
    mobileMenuOpen: false
};

// Date formatting helpers
function formatDateForDisplay(isoDate) {
    // Convert YYYY-MM-DD or ISO timestamp to DD/MM/YYYY
    if (!isoDate) return '';
    // Handle ISO timestamps by taking only the date part
    const datePart = isoDate.split('T')[0];
    const [year, month, day] = datePart.split('-');
    return `${day}/${month}/${year}`;
}

function formatDateToISO(displayDate) {
    // Convert DD/MM/YYYY to YYYY-MM-DD for API
    if (!displayDate) return '';
    const [day, month, year] = displayDate.split('/');
    return `${year}-${month}-${day}`;
}

function getTodayFormatted() {
    // Get today's date in DD/MM/YYYY format
    const today = new Date();
    const year = today.getFullYear();
    const month = String(today.getMonth() + 1).padStart(2, '0');
    const day = String(today.getDate()).padStart(2, '0');
    return `${day}/${month}/${year}`;
}

// Date picker helpers
function showDatePicker(inputId) {
    state.datePicker.targetInput = inputId;
    state.datePicker.visible = true;
    const input = document.getElementById(inputId);
    if (input && input.value) {
        const [day, month, year] = input.value.split('/');
        if (year && month) {
            state.datePicker.currentYear = parseInt(year);
            state.datePicker.currentMonth = parseInt(month) - 1;
        }
    }
    renderDatePicker();
}

function hideDatePicker(event) {
    if (event) {
        event.stopPropagation();
    }
    state.datePicker.visible = false;
    state.datePicker.targetInput = null;
    const existingPicker = document.getElementById('date-picker-root');
    if (existingPicker) {
        existingPicker.remove();
    }
}

function selectDate(day) {
    const month = String(state.datePicker.currentMonth + 1).padStart(2, '0');
    const dayStr = String(day).padStart(2, '0');
    const dateStr = `${dayStr}/${month}/${state.datePicker.currentYear}`;

    const input = document.getElementById(state.datePicker.targetInput);
    if (input) {
        input.value = dateStr;
        // Trigger change event
        const event = new Event('change', { bubbles: true });
        input.dispatchEvent(event);
    }
    hideDatePicker();
}

function changeMonth(offset) {
    state.datePicker.currentMonth += offset;
    if (state.datePicker.currentMonth < 0) {
        state.datePicker.currentMonth = 11;
        state.datePicker.currentYear--;
    } else if (state.datePicker.currentMonth > 11) {
        state.datePicker.currentMonth = 0;
        state.datePicker.currentYear++;
    }
    renderDatePicker();
}

function renderDatePicker() {
    // Remove existing picker if any
    const existingPicker = document.getElementById('date-picker-root');
    if (existingPicker) {
        existingPicker.remove();
    }

    // Add new picker if visible
    if (state.datePicker.visible) {
        const pickerDiv = document.createElement('div');
        pickerDiv.id = 'date-picker-root';
        pickerDiv.innerHTML = DatePicker();
        document.body.appendChild(pickerDiv);
    }
}

function DatePicker() {
    if (!state.datePicker.visible) return '';

    const monthNames = ta('months.full');
    const weekdays = ta('weekdays');

    const firstDay = new Date(state.datePicker.currentYear, state.datePicker.currentMonth, 1).getDay();
    const daysInMonth = new Date(state.datePicker.currentYear, state.datePicker.currentMonth + 1, 0).getDate();

    const today = new Date();
    const isCurrentMonth = today.getMonth() === state.datePicker.currentMonth &&
                          today.getFullYear() === state.datePicker.currentYear;

    let calendarDays = '';

    // Empty cells for days before the first day of month
    for (let i = 0; i < (firstDay === 0 ? 6 : firstDay - 1); i++) {
        calendarDays += '<div class="calendar-day empty"></div>';
    }

    // Days of the month
    for (let day = 1; day <= daysInMonth; day++) {
        const isToday = isCurrentMonth && today.getDate() === day;
        const classes = `calendar-day${isToday ? ' today' : ''}`;
        calendarDays += `<div class="${classes}" onclick="selectDate(${day}); event.stopPropagation();">${day}</div>`;
    }

    return `
        <div class="date-picker-overlay" onclick="hideDatePicker(event)">
            <div class="date-picker" onclick="event.stopPropagation()">
                <div class="date-picker-header">
                    <button onclick="changeMonth(-1); event.stopPropagation();" class="date-picker-nav">&lt;</button>
                    <span class="date-picker-title">${monthNames[state.datePicker.currentMonth]} ${state.datePicker.currentYear}</span>
                    <button onclick="changeMonth(1); event.stopPropagation();" class="date-picker-nav">&gt;</button>
                </div>
                <div class="calendar-weekdays">
                    ${weekdays.map(day => `<div>${day}</div>`).join('')}
                </div>
                <div class="calendar-days">
                    ${calendarDays}
                </div>
            </div>
        </div>
    `;
}

// API helpers
async function api(endpoint, options = {}) {
    try {
        const response = await fetch(endpoint, {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            credentials: 'same-origin'
        });

        if (response.status === 401) {
            state.user = null;
            render();
            return null;
        }

        const data = await response.json();
        return data;
    } catch (error) {
        console.error('API Error:', error);
        return null;
    }
}

// Theme management
function applyTheme() {
    if (state.theme === 'dark') {
        document.documentElement.classList.add('dark');
    } else {
        document.documentElement.classList.remove('dark');
    }
}

function toggleTheme() {
    state.theme = state.theme === 'light' ? 'dark' : 'light';
    localStorage.setItem('theme', state.theme);
    applyTheme();
    render();
}

function toggleLanguage() {
    const languages = getAvailableLanguages();
    const currentIndex = languages.indexOf(state.language);
    const nextIndex = (currentIndex + 1) % languages.length;
    state.language = languages[nextIndex];
    localStorage.setItem('language', state.language);
    render();
}

// Auth
async function login(username, password) {
    const data = await api('/api/login', {
        method: 'POST',
        body: JSON.stringify({ username, password })
    });

    console.log('Login response:', data);

    if (data && data.success) {
        state.user = data.user;
        await loadActions();
        await loadUsers();
        render();
    } else {
        const errorMsg = data?.message || t('login.error');
        console.error('Login failed:', errorMsg, data);
        alert(errorMsg);
    }
}

async function logout() {
    await api('/api/logout', { method: 'POST' });
    state.user = null;
    state.actions = [];
    state.users = [];
    render();
}

async function checkSession() {
    const data = await api('/api/me');
    if (data && data.success && data.user) {
        state.user = data.user;
        await loadActions();
        await loadUsers();
        await loadCategories();
        setMonthDateRange();
    }
    render();
}

// Data loading
async function loadActions() {
    const params = new URLSearchParams();
    params.append('limit', '10');

    const data = await api(`/api/actions?${params}`);
    if (data) {
        state.actions = data;
        render();
    }
}

async function loadUsers() {
    const data = await api('/api/users');
    if (data) {
        state.users = data;
    }
}

async function loadCategories(actionType = '') {
    const params = new URLSearchParams();
    if (actionType) params.append('action_type', actionType);

    const data = await api(`/api/categories?${params}`);
    if (data) {
        state.categories = data;
        return data;
    }
    return [];
}

async function createCategory(categoryData) {
    const data = await api('/api/categories', {
        method: 'POST',
        body: JSON.stringify(categoryData)
    });

    if (data && data.id) {
        await loadAllCategories();
        return true;
    }
    return false;
}

async function updateCategory(categoryId, categoryData) {
    const data = await api(`/api/categories/${categoryId}`, {
        method: 'PUT',
        body: JSON.stringify(categoryData)
    });

    if (data && data.id) {
        await loadAllCategories();
        return true;
    }
    return false;
}

async function deleteCategory(categoryId) {
    const data = await api(`/api/categories/${categoryId}`, {
        method: 'DELETE'
    });

    if (data) {
        await loadAllCategories();
        return true;
    }
    return false;
}

async function loadAllCategories() {
    const data = await api('/api/categories');
    if (data) {
        state.categoriesPage.categories = data;
        render();
    }
}

async function loadConfig() {
    const data = await api('/api/config');
    if (data && data.currency) {
        state.currency = data.currency;
    }
}

async function createAction(actionData) {
    const data = await api('/api/actions', {
        method: 'POST',
        body: JSON.stringify(actionData)
    });

    if (data && data.id) {
        await loadActions();
        return true;
    }
    return false;
}

async function updateAction(actionId, actionData) {
    const data = await api(`/api/actions/${actionId}`, {
        method: 'PUT',
        body: JSON.stringify(actionData)
    });

    if (data && data.id) {
        await loadActions();
        if (state.currentPage === 'all-actions') {
            await loadAllActions();
        }
        return true;
    }
    return false;
}

async function deleteAction(actionId) {
    const data = await api(`/api/actions/${actionId}`, {
        method: 'DELETE'
    });

    if (data) {
        await loadActions();
        if (state.currentPage === 'all-actions') {
            await loadAllActions();
        }
        return true;
    }
    return false;
}

// Components
function LoginPage() {
    return `
        <div class="login-container">
            <div class="card login-card">
                <div class="login-header">
                    <h1>${t('login.title')}</h1>
                    <button onclick="toggleTheme()" class="icon-btn">
                        ${state.theme === 'light' ? '🌙' : '☀️'}
                    </button>
                    <button onclick="toggleLanguage()"
                            class="icon-btn"
                            title="${getLangMeta(state.language).name}">
                        ${getLangMeta(state.language).code}
                    </button>
                </div>
                <form onsubmit="handleLogin(event)">
                    <div class="form-group">
                        <label>${t('login.username')}</label>
                        <input type="text" id="username" class="input" required>
                    </div>
                    <div class="form-group">
                        <label>${t('login.password')}</label>
                        <input type="password" id="password" class="input" required>
                    </div>
                    <div class="form-group mt-4">
                        <button type="submit" class="btn btn-primary w-full">${t('login.button')}</button>
                    </div>
                </form>
            </div>
        </div>
    `;
}

function Header() {
    return `
        <header>
            <div class="header-content">
                <h1><a href="#" onclick="navigateTo('dashboard'); return false;" style="text-decoration: none; color: inherit;">${t('nav.budgeting')}</a></h1>

                <!-- Desktop Navigation -->
                <div class="header-right desktop-nav">
                    <a href="#" onclick="navigateTo('all-actions'); return false;"
                       class="nav-link ${state.currentPage === 'all-actions' ? 'active' : ''}">
                        ${t('nav.all_actions')}
                    </a>
                    <a href="#" onclick="navigateTo('categories'); return false;"
                       class="nav-link ${state.currentPage === 'categories' ? 'active' : ''}">
                        ${t('nav.categories')}
                    </a>
                    <a href="#" onclick="navigateTo('charts'); return false;"
                       class="nav-link ${state.currentPage === 'charts' ? 'active' : ''}">
                        ${t('nav.charts')}
                    </a>
                    <button onclick="toggleTheme()" class="icon-btn">
                        ${state.theme === 'light' ? '🌙' : '☀️'}
                    </button>
                    <button onclick="toggleLanguage()"
                            class="icon-btn"
                            title="${getLangMeta(state.language).name}">
                        ${getLangMeta(state.language).code}
                    </button>
                    <div class="user-menu">
                        <button onclick="toggleUserMenu()" class="user-menu-trigger">
                            <span>${state.user?.name || 'User'}</span>
                            <span>▼</span>
                        </button>
                        <div id="user-menu" class="user-menu-dropdown hidden">
                            <button onclick="navigateTo('profile'); toggleUserMenu();">${t('nav.profile')}</button>
                            <button onclick="logout()">${t('nav.logout')}</button>
                        </div>
                    </div>
                </div>

                <!-- Mobile Burger Menu -->
                <button class="burger-menu-btn" onclick="toggleMobileMenu()">
                    <span class="burger-line"></span>
                    <span class="burger-line"></span>
                    <span class="burger-line"></span>
                </button>
            </div>

            <!-- Mobile Menu Overlay -->
            <div class="mobile-menu ${state.mobileMenuOpen ? 'open' : ''}" onclick="closeMobileMenuOnOverlay(event)">
                <div class="mobile-menu-content" onclick="event.stopPropagation()">
                    <div class="mobile-menu-header">
                        <h2>${state.user?.name || 'User'}</h2>
                        <button class="mobile-menu-close" onclick="toggleMobileMenu()">&times;</button>
                    </div>
                    <nav class="mobile-menu-nav">
                        <a href="#" onclick="navigateTo('all-actions'); toggleMobileMenu(); return false;"
                           class="mobile-nav-link ${state.currentPage === 'all-actions' ? 'active' : ''}">
                            ${t('nav.all_actions')}
                        </a>
                        <a href="#" onclick="navigateTo('categories'); toggleMobileMenu(); return false;"
                           class="mobile-nav-link ${state.currentPage === 'categories' ? 'active' : ''}">
                            ${t('nav.categories')}
                        </a>
                        <a href="#" onclick="navigateTo('charts'); toggleMobileMenu(); return false;"
                           class="mobile-nav-link ${state.currentPage === 'charts' ? 'active' : ''}">
                            ${t('nav.charts')}
                        </a>
                        <a href="#" onclick="navigateTo('profile'); toggleMobileMenu(); return false;"
                           class="mobile-nav-link ${state.currentPage === 'profile' ? 'active' : ''}">
                            ${t('nav.profile')}
                        </a>
                        <div class="mobile-menu-divider"></div>
                        <button onclick="toggleTheme()" class="mobile-menu-btn">
                            <span>${state.theme === 'light' ? '🌙' : '☀️'}</span>
                            <span>${state.theme === 'light' ? 'Dark Mode' : 'Light Mode'}</span>
                        </button>
                        <button onclick="toggleLanguage()" class="mobile-menu-btn">
                            <span>🌐</span>
                            <span>${getLangMeta(state.language).name}</span>
                        </button>
                        <div class="mobile-menu-divider"></div>
                        <button onclick="logout()" class="mobile-menu-btn logout-btn">
                            ${t('nav.logout')}
                        </button>
                    </nav>
                </div>
            </div>
        </header>
    `;
}

function Dashboard() {
    return `
        <div>
            ${Header()}
            <main class="dashboard-main">
                <div class="container">
                    ${ActionsList()}
                </div>
            </main>

            <button class="floating-button" onclick="openAddActionModal()">+</button>
            <div id="modal-container"></div>
        </div>
    `;
}

function AllActionsPage() {
    return `
        <div>
            ${Header()}
            <main class="dashboard-main">
                <div class="container">
                    ${AllActionsFilters()}
                    ${AllActionsTable()}
                    ${Pagination()}
                </div>
            </main>
            <div id="modal-container"></div>
        </div>
    `;
}

function AllActionsFilters() {
    const mode = state.allActionsPage.filters.mode;
    return `
        <div class="card filters-card mb-6">
            <div class="filter-mode-toggle mb-4">
                <button class="btn ${mode === 'month' ? 'btn-primary' : 'btn-secondary'}"
                        onclick="setFilterMode('month')">${t('filters.month_view')}</button>
                <button class="btn ${mode === 'custom' ? 'btn-primary' : 'btn-secondary'}"
                        onclick="setFilterMode('custom')">${t('filters.custom_range')}</button>
            </div>
            ${mode === 'month' ? MonthFilter() : CustomRangeFilters()}
        </div>
    `;
}

function MonthFilter() {
    const months = ta('months.full');
    const currentYear = new Date().getFullYear();
    const years = Array.from({length: 10}, (_, i) => currentYear - i);

    return `
        <div class="form-group mb-4">
            <label>${t('filters.search')}</label>
            <input type="text"
                   value="${state.allActionsPage.filters.search}"
                   oninput="updateAllActionsSearch(this.value)"
                   class="input"
                   placeholder="${t('filters.search_placeholder')}" />
        </div>

        <div class="filter-grid">
            <div class="form-group">
                <label>${t('filters.month')}</label>
                <select onchange="updateAllActionsFilter('month', parseInt(this.value))" class="input">
                    ${months.map((m, i) => `
                        <option value="${i}" ${state.allActionsPage.filters.month === i ? 'selected' : ''}>${m}</option>
                    `).join('')}
                </select>
            </div>
            <div class="form-group">
                <label>${t('filters.year')}</label>
                <select onchange="updateAllActionsFilter('year', parseInt(this.value))" class="input">
                    ${years.map(y => `
                        <option value="${y}" ${state.allActionsPage.filters.year === y ? 'selected' : ''}>${y}</option>
                    `).join('')}
                </select>
            </div>
            ${UserAndTypeFilters('all-actions')}
            <div class="form-group" style="display: flex; align-items: flex-end;">
                <button onclick="clearAllActionsFilters()" class="btn btn-secondary" style="width: 100%;">
                    ${t('filters.clear')}
                </button>
            </div>
        </div>
    `;
}

function CustomRangeFilters() {
    return `
        <div class="form-group mb-4">
            <label>${t('filters.search')}</label>
            <input type="text"
                   value="${state.allActionsPage.filters.search}"
                   oninput="updateAllActionsSearch(this.value)"
                   class="input"
                   placeholder="${t('filters.search_placeholder')}" />
        </div>

        <div class="filter-grid">
            ${UserAndTypeFilters('all-actions')}
            <div class="form-group">
                <label>${t('filters.from_date')}</label>
                <input type="text" id="all-actions-date-from"
                       value="${state.allActionsPage.filters.date_from}"
                       onclick="showDatePicker('all-actions-date-from')"
                       class="input" placeholder="${t('date_format')}" readonly>
            </div>
            <div class="form-group">
                <label>${t('filters.to_date')}</label>
                <input type="text" id="all-actions-date-to"
                       value="${state.allActionsPage.filters.date_to}"
                       onclick="showDatePicker('all-actions-date-to')"
                       class="input" placeholder="${t('date_format')}" readonly>
            </div>
            <div class="form-group" style="display: flex; align-items: flex-end;">
                <button onclick="clearAllActionsFilters()" class="btn btn-secondary" style="width: 100%;">
                    ${t('filters.clear')}
                </button>
            </div>
        </div>
    `;
}

function UserAndTypeFilters(context) {
    const filters = context === 'all-actions' ? state.allActionsPage.filters : state.filters;
    const updateFn = context === 'all-actions' ? 'updateAllActionsFilter' : 'updateFilter';

    return `
        <div class="form-group">
            <label>${t('filters.user')}</label>
            <select onchange="${updateFn}('username', this.value)" class="input">
                <option value="">${t('filters.all_users')}</option>
                ${state.users.map(u => `
                    <option value="${u.username}" ${filters.username === u.username ? 'selected' : ''}>${u.name}</option>
                `).join('')}
            </select>
        </div>
        <div class="form-group">
            <label>${t('filters.type')}</label>
            <select onchange="${updateFn}('type', this.value)" class="input">
                <option value="">${t('filters.all')}</option>
                <option value="income" ${filters.type === 'income' ? 'selected' : ''}>${t('filters.income')}</option>
                <option value="expense" ${filters.type === 'expense' ? 'selected' : ''}>${t('filters.expense')}</option>
            </select>
        </div>
    `;
}

function AllActionsTable() {
    const actions = state.allActionsPage.actions;

    if (actions.length === 0) {
        return `
            <div class="card empty-state">
                <p class="empty-state-title">${t('empty.no_actions_found')}</p>
                <p class="empty-state-text">${t('empty.adjust_filters')}</p>
            </div>
        `;
    }

    return `
        <div class="card mb-6">
            <div class="overflow-x-auto">
                <table>
                    <thead>
                        <tr>
                            <th>${t('table.date')}</th>
                            <th>${t('table.user')}</th>
                            <th>${t('table.type')}</th>
                            <th>${t('table.description')}</th>
                            <th>${t('table.category')}</th>
                            <th class="text-right">${t('table.amount')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${actions.map(action => {
                            const category = action.category_id ? state.categories.find(c => c.id === action.category_id) : null;
                            return `
                            <tr class="${action.user_id === state.user.id ? 'action-row-editable' : ''}" ${action.user_id === state.user.id ? `onclick="openEditActionModal(${action.id})"` : ''}>
                                <td>${formatDateForDisplay(action.date)}</td>
                                <td>${action.name}</td>
                                <td><span class="badge ${action.type === 'income' ? 'badge-success' : 'badge-danger'}">${action.type === 'income' ? t('filters.income') : t('filters.expense')}</span></td>
                                <td>${action.description}</td>
                                <td>${category ? category.description : '-'}</td>
                                <td class="text-right ${action.type === 'income' ? 'amount-success' : 'amount-danger'}">
                                    ${action.type === 'income' ? '+' : '-'}${state.currency}${action.amount.toFixed(2)}
                                </td>
                            </tr>
                        `}).join('')}
                    </tbody>
                </table>
            </div>
        </div>
    `;
}

function Pagination() {
    const { currentPage, totalPages, totalActions, perPage } = state.allActionsPage.pagination;

    if (totalPages <= 1) return '';

    const pageNumbers = calculatePageNumbers(currentPage, totalPages);

    return `
        <div class="card pagination-card">
            <div class="pagination-container">
                <button class="btn btn-secondary" onclick="goToPage(${currentPage - 1})"
                        ${currentPage === 1 ? 'disabled' : ''}>${t('pagination.previous')}</button>

                <div class="pagination-numbers">
                    ${pageNumbers.map(page => {
                        if (page === '...') return '<span class="pagination-ellipsis">...</span>';
                        return `<button class="btn ${page === currentPage ? 'btn-primary' : 'btn-secondary'}"
                                        onclick="goToPage(${page})">${page}</button>`;
                    }).join('')}
                </div>

                <button class="btn btn-secondary" onclick="goToPage(${currentPage + 1})"
                        ${currentPage === totalPages ? 'disabled' : ''}>${t('pagination.next')}</button>
            </div>
            <div class="pagination-info">
                ${t('pagination.showing', {
                    from: (currentPage - 1) * perPage + 1,
                    to: Math.min(currentPage * perPage, totalActions),
                    total: totalActions
                })}
            </div>
        </div>
    `;
}

function ChartsPage() {
    return `
        <div>
            ${Header()}
            <main class="dashboard-main">
                <div class="container">
                    ${ChartsFilters()}
                    ${ChartsDisplay()}
                </div>
            </main>
            <div id="modal-container"></div>
        </div>
    `;
}

function ChartsFilters() {
    const currentYear = new Date().getFullYear();
    const years = Array.from({length: 10}, (_, i) => currentYear - i);

    return `
        <div class="card filters-card mb-6">
            <h2 style="margin-bottom: 1rem;">${t('charts.title')}</h2>
            <div class="filter-grid">
                <div class="form-group">
                    <label>${t('filters.year')}</label>
                    <select onchange="updateChartYear(parseInt(this.value))" class="input">
                        ${years.map(y => `
                            <option value="${y}" ${state.chartsPage.year === y ? 'selected' : ''}>${y}</option>
                        `).join('')}
                    </select>
                </div>
            </div>
        </div>
    `;
}

function ChartsDisplay() {
    if (!state.chartsPage.chartData) {
        return `
            <div class="card empty-state">
                <p class="empty-state-title">${t('empty.loading_chart')}</p>
            </div>
        `;
    }

    return `
        <div class="card" style="padding: 2rem;">
            <canvas id="monthly-chart"></canvas>
        </div>
    `;
}

function CategoriesPage() {
    return `
        <div>
            ${Header()}
            <main class="dashboard-main">
                <div class="container">
                    ${CategoriesTable()}
                </div>
            </main>

            <button class="floating-button" onclick="openAddCategoryModal()">+</button>
            <div id="modal-container"></div>
        </div>
    `;
}

function CategoriesTable() {
    const categories = state.categoriesPage.categories;

    if (categories.length === 0) {
        return `
            <div class="card empty-state">
                <p class="empty-state-title">${t('categories.empty_title')}</p>
                <p class="empty-state-text">${t('categories.empty_text')}</p>
            </div>
        `;
    }

    return `
        <div class="card">
            <div class="overflow-x-auto">
                <table>
                    <thead>
                        <tr>
                            <th>${t('categories.description')}</th>
                            <th>${t('categories.type')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${categories.map(category => `
                            <tr class="action-row-editable" onclick="openEditCategoryModal(${category.id})">
                                <td>${category.description}</td>
                                <td><span class="badge ${category.action_type === 'income' ? 'badge-success' : 'badge-danger'}">${category.action_type === 'income' ? t('filters.income') : t('filters.expense')}</span></td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            </div>
        </div>
    `;
}

function AddCategoryModal() {
    return `
        <div class="modal-overlay" onclick="closeCategoryModal(event)">
            <div class="modal-content" onclick="event.stopPropagation()">
                <div class="modal-header">
                    <h2>${t('categories.add')}</h2>
                    <button onclick="closeCategoryModal()" class="modal-close">&times;</button>
                </div>
                <form onsubmit="handleAddCategory(event)">
                    <div class="form-group">
                        <label>${t('categories.type')}</label>
                        <select id="category-action-type" class="input">
                            <option value="expense">${t('filters.expense')}</option>
                            <option value="income">${t('filters.income')}</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>${t('categories.description')}</label>
                        <input type="text" id="category-description" class="input" required>
                    </div>
                    <div class="modal-actions">
                        <button type="submit" class="btn btn-primary">${t('modal.submit')}</button>
                        <button type="button" onclick="closeCategoryModal()" class="btn btn-secondary">${t('modal.cancel')}</button>
                    </div>
                </form>
            </div>
        </div>
    `;
}

function EditCategoryModal() {
    const category = state.categoriesPage.editCategoryModal.category;
    if (!category) return '';

    return `
        <div class="modal-overlay" onclick="closeEditCategoryModal(event)">
            <div class="modal-content" onclick="event.stopPropagation()">
                <div class="modal-header">
                    <h2>${t('categories.edit')}</h2>
                    <button onclick="closeEditCategoryModal()" class="modal-close">&times;</button>
                </div>
                <form onsubmit="handleEditCategory(event)">
                    <div class="form-group">
                        <label>${t('categories.type')}</label>
                        <select id="edit-category-action-type" class="input">
                            <option value="expense" ${category.action_type === 'expense' ? 'selected' : ''}>${t('filters.expense')}</option>
                            <option value="income" ${category.action_type === 'income' ? 'selected' : ''}>${t('filters.income')}</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>${t('categories.description')}</label>
                        <input type="text" id="edit-category-description" value="${category.description}" class="input" required>
                    </div>
                    <div class="modal-actions">
                        <button type="submit" class="btn btn-primary">${t('modal.save')}</button>
                        <button type="button" onclick="handleDeleteCategory(${category.id})" class="btn btn-danger">${t('modal.delete')}</button>
                        <button type="button" onclick="closeEditCategoryModal()" class="btn btn-secondary">${t('modal.cancel')}</button>
                    </div>
                </form>
            </div>
        </div>
    `;
}

function ProfilePage() {
    return `
        <div>
            ${Header()}
            <main class="dashboard-main">
                <div class="container">
                    <div class="profile-container">
                        <div class="card profile-card">
                            <h2 class="profile-title">${t('profile.title')}</h2>

                            ${state.profilePage.message ? InlineMessage(state.profilePage.message) : ''}

                            <form onsubmit="handleUpdateProfile(event)">
                                <div class="form-group">
                                    <label>${t('profile.username')}</label>
                                    <input type="text"
                                           value="${state.user?.username || ''}"
                                           class="input"
                                           disabled
                                           style="opacity: 0.6; cursor: not-allowed;">
                                    <small style="color: var(--text-secondary); font-size: 0.875rem;">${t('profile.username_note')}</small>
                                </div>

                                <div class="form-group">
                                    <label>${t('profile.name')} <span style="color: var(--danger);">*</span></label>
                                    <input type="text"
                                           id="profile-name"
                                           value="${state.user?.name || ''}"
                                           class="input"
                                           required
                                           maxlength="100">
                                </div>

                                <div class="profile-section-divider">
                                    <h3>${t('profile.change_password')}</h3>
                                    <p style="color: var(--text-secondary); font-size: 0.875rem; margin-top: 0.25rem;">
                                        ${t('profile.password_note')}
                                    </p>
                                </div>

                                <div class="form-group">
                                    <label>${t('profile.current_password')}</label>
                                    <input type="password"
                                           id="profile-current-password"
                                           class="input"
                                           autocomplete="current-password">
                                    <small style="color: var(--text-secondary); font-size: 0.875rem;">
                                        ${t('profile.password_required_note')}
                                    </small>
                                </div>

                                <div class="form-group">
                                    <label>${t('profile.new_password')}</label>
                                    <input type="password"
                                           id="profile-new-password"
                                           class="input"
                                           minlength="6"
                                           autocomplete="new-password">
                                    <small style="color: var(--text-secondary); font-size: 0.875rem;">
                                        ${t('profile.password_min_note')}
                                    </small>
                                </div>

                                <div class="form-group">
                                    <label>${t('profile.confirm_password')}</label>
                                    <input type="password"
                                           id="profile-confirm-password"
                                           class="input"
                                           autocomplete="new-password">
                                </div>

                                <div class="profile-actions">
                                    <button type="submit"
                                            class="btn btn-primary"
                                            ${state.profilePage.isSubmitting ? 'disabled' : ''}>
                                        ${state.profilePage.isSubmitting ? t('profile.saving') : t('profile.save')}
                                    </button>
                                    <button type="button"
                                            onclick="navigateTo('dashboard')"
                                            class="btn btn-secondary"
                                            ${state.profilePage.isSubmitting ? 'disabled' : ''}>
                                        ${t('modal.cancel')}
                                    </button>
                                </div>
                            </form>
                        </div>
                    </div>
                </div>
            </main>
            <div id="modal-container"></div>
        </div>
    `;
}

function InlineMessage(message) {
    if (!message) return '';

    const typeClass = message.type === 'success' ? 'message-success' : 'message-error';
    const icon = message.type === 'success' ? '✓' : '✕';

    return `
        <div class="inline-message ${typeClass}">
            <span class="message-icon">${icon}</span>
            <span class="message-text">${message.text}</span>
            <button onclick="dismissMessage()" class="message-dismiss" type="button">×</button>
        </div>
    `;
}

function dismissMessage() {
    state.profilePage.message = null;
    render();
}

function renderChart() {
    if (state.currentPage !== 'charts' || !state.chartsPage.chartData) {
        return;
    }

    // Destroy existing chart to prevent duplicates
    if (state.chartsPage.chartInstance) {
        state.chartsPage.chartInstance.destroy();
        state.chartsPage.chartInstance = null;
    }

    // Wait for DOM to be ready
    setTimeout(() => {
        const canvas = document.getElementById('monthly-chart');
        if (!canvas) return;

        const ctx = canvas.getContext('2d');
        const chartData = state.chartsPage.chartData;

        const labels = chartData.data.map(d => d.month);
        const incomeData = chartData.data.map(d => d.income);
        const expenseData = chartData.data.map(d => d.expense);

        state.chartsPage.chartInstance = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: labels,
                datasets: [
                    {
                        label: t('charts.income'),
                        data: incomeData,
                        backgroundColor: '#10b981',
                        borderColor: '#10b981',
                        borderWidth: 1
                    },
                    {
                        label: t('charts.expenses'),
                        data: expenseData,
                        backgroundColor: '#ef4444',
                        borderColor: '#ef4444',
                        borderWidth: 1
                    }
                ]
            },
            options: {
                responsive: true,
                maintainAspectRatio: true,
                plugins: {
                    legend: {
                        display: true,
                        position: 'top',
                        labels: {
                            color: getComputedStyle(document.documentElement)
                                .getPropertyValue('--text-primary').trim()
                        }
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                let label = context.dataset.label || '';
                                if (label) label += ': ';
                                label += state.currency + context.parsed.y.toFixed(2);
                                return label;
                            }
                        }
                    }
                },
                scales: {
                    x: {
                        grid: {
                            color: getComputedStyle(document.documentElement)
                                .getPropertyValue('--border-color').trim()
                        },
                        ticks: {
                            color: getComputedStyle(document.documentElement)
                                .getPropertyValue('--text-primary').trim()
                        }
                    },
                    y: {
                        beginAtZero: true,
                        grid: {
                            color: getComputedStyle(document.documentElement)
                                .getPropertyValue('--border-color').trim()
                        },
                        ticks: {
                            color: getComputedStyle(document.documentElement)
                                .getPropertyValue('--text-primary').trim(),
                            callback: function(value) {
                                return state.currency + value;
                            }
                        }
                    }
                }
            }
        });
    }, 0);
}

function ActionsList() {
    if (state.actions.length === 0) {
        return `
            <div class="card empty-state">
                <p class="empty-state-title">${t('empty.no_actions')}</p>
                <p class="empty-state-text">${t('empty.click_add')}</p>
            </div>
        `;
    }

    return `
        <div class="card">
            <div class="overflow-x-auto">
                <table>
                    <thead>
                        <tr>
                            <th>${t('table.date')}</th>
                            <th>${t('table.user')}</th>
                            <th>${t('table.type')}</th>
                            <th>${t('table.description')}</th>
                            <th>${t('table.category')}</th>
                            <th class="text-right">${t('table.amount')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${state.actions.map(action => {
                            const category = action.category_id ? state.categories.find(c => c.id === action.category_id) : null;
                            return `
                            <tr class="${action.user_id === state.user.id ? 'action-row-editable' : ''}" ${action.user_id === state.user.id ? `onclick="openEditActionModal(${action.id})"` : ''}>
                                <td>${formatDateForDisplay(action.date)}</td>
                                <td>${action.name}</td>
                                <td>
                                    <span class="badge ${action.type === 'income' ? 'badge-success' : 'badge-danger'}">
                                        ${action.type === 'income' ? t('filters.income') : t('filters.expense')}
                                    </span>
                                </td>
                                <td>${action.description}</td>
                                <td>${category ? category.description : '-'}</td>
                                <td class="text-right ${action.type === 'income' ? 'amount-success' : 'amount-danger'}">
                                    ${action.type === 'income' ? '+' : '-'}${state.currency}${action.amount.toFixed(2)}
                                </td>
                            </tr>
                        `}).join('')}
                    </tbody>
                </table>
            </div>
            <div class="card-footer">
                <button onclick="navigateTo('all-actions')" class="btn btn-secondary w-full">
                    ${t('dashboard.view_all_actions')}
                </button>
            </div>
        </div>
    `;
}

function AddActionModal() {
    const today = getTodayFormatted();
    const expenseCategories = state.categories.filter(c => c.action_type === 'expense');

    return `
        <div class="modal-overlay" onclick="closeModal(event)">
            <div class="modal-content" onclick="event.stopPropagation()">
                <div class="modal-header">
                    <h2>${t('modal.add_action')}</h2>
                    <button onclick="closeModal()" class="modal-close">&times;</button>
                </div>
                <form onsubmit="handleAddAction(event)">
                    <div class="form-group">
                        <label>${t('modal.type')}</label>
                        <select id="action-type" class="input" onchange="updateActionTypeCategories()">
                            <option value="expense">${t('filters.expense')}</option>
                            <option value="income">${t('filters.income')}</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>${t('categories.category_optional')}</label>
                        <select id="action-category" class="input">
                            <option value="">${t('categories.none')}</option>
                            ${expenseCategories.map(c => `<option value="${c.id}">${c.description}</option>`).join('')}
                        </select>
                    </div>
                    <div class="form-group">
                        <label>${t('modal.date')}</label>
                        <input type="text" id="action-date" value="${today}" onclick="showDatePicker('action-date')" class="input" placeholder="${t('date_format')}" pattern="\\d{2}/\\d{2}/\\d{4}" readonly required>
                    </div>
                    <div class="form-group">
                        <label>${t('modal.description')}</label>
                        <input type="text" id="action-description" class="input" required>
                    </div>
                    <div class="form-group">
                        <label>${t('modal.amount')}</label>
                        <input type="number" id="action-amount" step="0.01" min="0.01" class="input" required>
                    </div>
                    <div class="modal-actions">
                        <button type="submit" class="btn btn-primary">${t('modal.submit')}</button>
                        <button type="button" onclick="closeModal()" class="btn btn-secondary">${t('modal.cancel')}</button>
                    </div>
                </form>
            </div>
        </div>
    `;
}

function EditActionModal() {
    const action = state.editActionModal.action;
    if (!action) return '';

    const displayDate = formatDateForDisplay(action.date);
    const actionCategories = state.categories.filter(c => c.action_type === action.type);

    return `
        <div class="modal-overlay" onclick="closeEditModal(event)">
            <div class="modal-content" onclick="event.stopPropagation()">
                <div class="modal-header">
                    <h2>${t('modal.edit_action')}</h2>
                    <button onclick="closeEditModal()" class="modal-close">&times;</button>
                </div>
                <form onsubmit="handleEditAction(event)">
                    <div class="form-group">
                        <label>${t('modal.type')}</label>
                        <select id="edit-action-type" class="input" onchange="updateEditActionTypeCategories()">
                            <option value="expense" ${action.type === 'expense' ? 'selected' : ''}>${t('filters.expense')}</option>
                            <option value="income" ${action.type === 'income' ? 'selected' : ''}>${t('filters.income')}</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>${t('categories.category_optional')}</label>
                        <select id="edit-action-category" class="input">
                            <option value="">${t('categories.none')}</option>
                            ${actionCategories.map(c => `<option value="${c.id}" ${action.category_id === c.id ? 'selected' : ''}>${c.description}</option>`).join('')}
                        </select>
                    </div>
                    <div class="form-group">
                        <label>${t('modal.date')}</label>
                        <input type="text" id="edit-action-date" value="${displayDate}" onclick="showDatePicker('edit-action-date')" class="input" placeholder="${t('date_format')}" pattern="\\d{2}/\\d{2}/\\d{4}" readonly required>
                    </div>
                    <div class="form-group">
                        <label>${t('modal.description')}</label>
                        <input type="text" id="edit-action-description" value="${action.description}" class="input" required>
                    </div>
                    <div class="form-group">
                        <label>${t('modal.amount')}</label>
                        <input type="number" id="edit-action-amount" step="0.01" min="0.01" value="${action.amount}" class="input" required>
                    </div>
                    <div class="modal-actions">
                        <button type="submit" class="btn btn-primary">${t('modal.save')}</button>
                        <button type="button" onclick="handleDeleteAction(${action.id})" class="btn btn-danger">${t('modal.delete')}</button>
                        <button type="button" onclick="closeEditModal()" class="btn btn-secondary">${t('modal.cancel')}</button>
                    </div>
                </form>
            </div>
        </div>
    `;
}

// Event handlers
function handleLogin(event) {
    event.preventDefault();
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    login(username, password);
}

function updateActionTypeCategories() {
    const actionType = document.getElementById('action-type').value;
    const categorySelect = document.getElementById('action-category');
    const categories = state.categories.filter(c => c.action_type === actionType);

    categorySelect.innerHTML = `<option value="">${t('categories.none')}</option>` +
        categories.map(c => `<option value="${c.id}">${c.description}</option>`).join('');
}

function updateEditActionTypeCategories() {
    const actionType = document.getElementById('edit-action-type').value;
    const categorySelect = document.getElementById('edit-action-category');
    const categories = state.categories.filter(c => c.action_type === actionType);

    categorySelect.innerHTML = `<option value="">${t('categories.none')}</option>` +
        categories.map(c => `<option value="${c.id}">${c.description}</option>`).join('');
}

async function handleAddAction(event) {
    event.preventDefault();
    const categoryValue = document.getElementById('action-category').value;
    const actionData = {
        type: document.getElementById('action-type').value,
        date: formatDateToISO(document.getElementById('action-date').value),
        description: document.getElementById('action-description').value,
        amount: parseFloat(document.getElementById('action-amount').value),
        category_id: categoryValue ? parseInt(categoryValue) : null
    };

    if (await createAction(actionData)) {
        closeModal();
    } else {
        alert(t('validation.failed_create'));
    }
}

function openEditActionModal(actionId) {
    // Find the action in state
    let action = state.actions.find(a => a.id === actionId);
    if (!action) {
        action = state.allActionsPage.actions.find(a => a.id === actionId);
    }

    if (action) {
        state.editActionModal = {
            visible: true,
            action: action
        };
        const modalContainer = document.getElementById('modal-container');
        if (modalContainer) {
            modalContainer.innerHTML = EditActionModal();
        } else {
            console.error('Modal container not found - cannot display edit modal');
        }
    }
}

function closeEditModal(event) {
    if (event && event.target !== event.currentTarget) {
        return;
    }
    state.editActionModal = {
        visible: false,
        action: null
    };
    document.getElementById('modal-container').innerHTML = '';
}

async function handleEditAction(event) {
    event.preventDefault();

    const categoryValue = document.getElementById('edit-action-category').value;
    const actionData = {
        type: document.getElementById('edit-action-type').value,
        date: formatDateToISO(document.getElementById('edit-action-date').value),
        description: document.getElementById('edit-action-description').value,
        amount: parseFloat(document.getElementById('edit-action-amount').value),
        category_id: categoryValue ? parseInt(categoryValue) : null
    };

    if (await updateAction(state.editActionModal.action.id, actionData)) {
        closeEditModal();
    } else {
        alert(t('validation.failed_update'));
    }
}

async function handleDeleteAction(actionId) {
    if (!confirm(t('modal.delete_confirm'))) {
        return;
    }

    if (await deleteAction(actionId)) {
        closeEditModal();
    } else {
        alert(t('validation.failed_delete'));
    }
}

async function handleUpdateProfile(event) {
    event.preventDefault();

    // Get form values
    const name = document.getElementById('profile-name').value.trim();
    const currentPassword = document.getElementById('profile-current-password').value;
    const newPassword = document.getElementById('profile-new-password').value;
    const confirmPassword = document.getElementById('profile-confirm-password').value;

    // Clear previous message
    state.profilePage.message = null;

    // Frontend validation
    if (!name) {
        state.profilePage.message = {
            type: 'error',
            text: t('validation.name_required')
        };
        render();
        return;
    }

    // If new password is provided, validate
    if (newPassword) {
        if (!currentPassword) {
            state.profilePage.message = {
                type: 'error',
                text: t('validation.password_required')
            };
            render();
            return;
        }

        if (newPassword.length < 6) {
            state.profilePage.message = {
                type: 'error',
                text: t('validation.password_min')
            };
            render();
            return;
        }

        if (newPassword !== confirmPassword) {
            state.profilePage.message = {
                type: 'error',
                text: t('validation.passwords_match')
            };
            render();
            return;
        }
    }

    // Set submitting state
    state.profilePage.isSubmitting = true;
    render();

    // Prepare request data
    const profileData = {
        name: name,
        current_password: currentPassword,
        new_password: newPassword
    };

    // Call API
    const response = await updateProfile(profileData);

    // Reset submitting state
    state.profilePage.isSubmitting = false;

    if (response && response.success) {
        // Update user in state with new name
        state.user.name = response.user.name;

        // Show success message
        state.profilePage.message = {
            type: 'success',
            text: response.message || t('validation.success')
        };

        // Clear password fields
        document.getElementById('profile-current-password').value = '';
        document.getElementById('profile-new-password').value = '';
        document.getElementById('profile-confirm-password').value = '';

        // Auto-dismiss success message after 5 seconds
        setTimeout(() => {
            if (state.profilePage.message?.type === 'success') {
                state.profilePage.message = null;
                render();
            }
        }, 5000);
    } else {
        // Show error message
        state.profilePage.message = {
            type: 'error',
            text: response?.message || t('validation.error')
        };
    }

    render();
}

// Category event handlers
async function handleAddCategory(event) {
    event.preventDefault();
    const categoryData = {
        action_type: document.getElementById('category-action-type').value,
        description: document.getElementById('category-description').value
    };

    if (await createCategory(categoryData)) {
        closeCategoryModal();
    } else {
        alert(t('categories.failed_create'));
    }
}

function openEditCategoryModal(categoryId) {
    const category = state.categoriesPage.categories.find(c => c.id === categoryId);
    if (category) {
        state.categoriesPage.editCategoryModal = {
            visible: true,
            category: category
        };
        const modalContainer = document.getElementById('modal-container');
        if (modalContainer) {
            modalContainer.innerHTML = EditCategoryModal();
        }
    }
}

function closeEditCategoryModal(event) {
    if (event && event.target !== event.currentTarget) {
        return;
    }
    state.categoriesPage.editCategoryModal = {
        visible: false,
        category: null
    };
    document.getElementById('modal-container').innerHTML = '';
}

async function handleEditCategory(event) {
    event.preventDefault();
    const categoryData = {
        action_type: document.getElementById('edit-category-action-type').value,
        description: document.getElementById('edit-category-description').value
    };

    if (await updateCategory(state.categoriesPage.editCategoryModal.category.id, categoryData)) {
        closeEditCategoryModal();
    } else {
        alert(t('categories.failed_update'));
    }
}

async function handleDeleteCategory(categoryId) {
    if (!confirm(t('categories.delete_confirm'))) {
        return;
    }

    if (await deleteCategory(categoryId)) {
        closeEditCategoryModal();
    } else {
        alert(t('categories.failed_delete'));
    }
}

function openAddCategoryModal() {
    document.getElementById('modal-container').innerHTML = AddCategoryModal();
}

function closeCategoryModal(event) {
    if (!event || event.target.classList.contains('modal-overlay')) {
        document.getElementById('modal-container').innerHTML = '';
    }
}

// All Actions Page Navigation and Filters
function navigateTo(page) {
    state.currentPage = page;

    // Clear profile message when navigating away
    if (page !== 'profile') {
        state.profilePage.message = null;
    }

    if (page === 'all-actions') {
        loadAllActions();
    } else if (page === 'dashboard') {
        loadActions();
    } else if (page === 'charts') {
        loadChartData();
    } else if (page === 'profile') {
        // No data loading needed - user info already in state
    } else if (page === 'categories') {
        loadAllCategories();
    }

    render();
}

function setFilterMode(mode) {
    state.allActionsPage.filters.mode = mode;
    if (mode === 'month') {
        setMonthDateRange();
    }
    state.allActionsPage.pagination.currentPage = 1;
    loadAllActions();
}

function setMonthDateRange() {
    const { month, year } = state.allActionsPage.filters;
    const firstDay = `${year}-${String(month + 1).padStart(2, '0')}-01`;
    const lastDay = new Date(year, month + 1, 0).getDate();
    const lastDayStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(lastDay).padStart(2, '0')}`;
    state.allActionsPage.filters.date_from = formatDateForDisplay(firstDay);
    state.allActionsPage.filters.date_to = formatDateForDisplay(lastDayStr);
}

function updateAllActionsFilter(key, value) {
    state.allActionsPage.filters[key] = value;
    if (state.allActionsPage.filters.mode === 'month' && (key === 'month' || key === 'year')) {
        setMonthDateRange();
    }
    state.allActionsPage.pagination.currentPage = 1;
    loadAllActions();
}

// Debounced search to avoid excessive API calls
let searchTimeout;
function updateAllActionsSearch(value) {
    state.allActionsPage.filters.search = value;
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
        state.allActionsPage.pagination.currentPage = 1;
        loadAllActions();
    }, 500);
}

function clearAllActionsFilters() {
    clearTimeout(searchTimeout);
    const now = new Date();
    state.allActionsPage.filters = {
        mode: 'month',
        month: now.getMonth(),
        year: now.getFullYear(),
        username: '',
        type: '',
        date_from: '',
        date_to: '',
        search: ''
    };
    state.allActionsPage.pagination.currentPage = 1;
    setMonthDateRange();
    loadAllActions();
    render();
}

function goToPage(page) {
    if (page < 1 || page > state.allActionsPage.pagination.totalPages) return;
    state.allActionsPage.pagination.currentPage = page;
    loadAllActions();
}

async function loadAllActions() {
    const { filters, pagination } = state.allActionsPage;
    const params = new URLSearchParams();

    const offset = (pagination.currentPage - 1) * pagination.perPage;
    params.append('offset', offset);
    params.append('limit', pagination.perPage);

    if (filters.date_from) params.append('date_from', formatDateToISO(filters.date_from));
    if (filters.date_to) params.append('date_to', formatDateToISO(filters.date_to));
    if (filters.username) params.append('username', filters.username);
    if (filters.type) params.append('type', filters.type);
    if (filters.search) params.append('search', filters.search);

    const data = await api(`/api/actions?${params}`);
    if (data) {
        state.allActionsPage.actions = data.actions || [];
        state.allActionsPage.pagination.totalActions = data.total || 0;
        state.allActionsPage.pagination.totalPages = Math.ceil(data.total / pagination.perPage);
        render();
    }
}

async function loadChartData() {
    const params = new URLSearchParams();
    params.append('year', state.chartsPage.year);

    const data = await api(`/api/charts/monthly?${params}`);
    if (data) {
        state.chartsPage.chartData = data;
        render();
    }
}

function updateChartYear(year) {
    state.chartsPage.year = year;
    loadChartData();
}

async function updateProfile(profileData) {
    const data = await api('/api/profile', {
        method: 'PUT',
        body: JSON.stringify(profileData)
    });
    return data;
}

function calculatePageNumbers(current, total) {
    const pages = new Set();
    pages.add(1);
    pages.add(total);
    for (let i = Math.max(1, current - 1); i <= Math.min(total, current + 1); i++) {
        pages.add(i);
    }

    const sorted = Array.from(pages).sort((a, b) => a - b);
    const result = [];
    for (let i = 0; i < sorted.length; i++) {
        if (i > 0 && sorted[i] - sorted[i-1] > 1) result.push('...');
        result.push(sorted[i]);
    }
    return result;
}

function toggleUserMenu() {
    const menu = document.getElementById('user-menu');
    menu.classList.toggle('hidden');
}

function toggleMobileMenu() {
    state.mobileMenuOpen = !state.mobileMenuOpen;
    render();
}

function closeMobileMenuOnOverlay(event) {
    if (event.target.classList.contains('mobile-menu')) {
        toggleMobileMenu();
    }
}

function openAddActionModal() {
    document.getElementById('modal-container').innerHTML = AddActionModal();
}

function closeModal(event) {
    if (!event || event.target.classList.contains('modal-overlay')) {
        document.getElementById('modal-container').innerHTML = '';
    }
}

// Render
function render() {
    const app = document.getElementById('app');
    if (!state.user) {
        app.innerHTML = LoginPage();
    } else {
        if (state.currentPage === 'all-actions') {
            app.innerHTML = AllActionsPage();
        } else if (state.currentPage === 'charts') {
            app.innerHTML = ChartsPage();
            renderChart();
        } else if (state.currentPage === 'profile') {
            app.innerHTML = ProfilePage();
        } else if (state.currentPage === 'categories') {
            app.innerHTML = CategoriesPage();
        } else {
            app.innerHTML = Dashboard();
        }
    }
    applyTheme();
}

// Close user menu when clicking outside
document.addEventListener('click', (e) => {
    const userMenu = document.getElementById('user-menu');
    if (userMenu && !e.target.closest('.user-menu')) {
        userMenu.classList.add('hidden');
    }
});

// Initial render
applyTheme();
loadConfig().then(() => checkSession());

// Register service worker
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker.register('/sw.js').catch(() => {});
    });
}
