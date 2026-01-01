// Translation data with metadata
const translations = {
    en: {
        _meta: {
            code: 'EN',
            name: 'English'
        },
        // Login
        'login.title': 'Budgeting App',
        'login.username': 'Username',
        'login.password': 'Password',
        'login.button': 'Login',
        'login.error': 'Login failed. Please check your credentials.',

        // Navigation
        'nav.budgeting': 'Budgeting',
        'nav.all_actions': 'All Actions',
        'nav.charts': 'Charts',
        'nav.profile': 'Profile',
        'nav.logout': 'Logout',

        // Filters
        'filters.user': 'User',
        'filters.all_users': 'All Users',
        'filters.type': 'Type',
        'filters.all': 'All',
        'filters.income': 'Income',
        'filters.expense': 'Expense',
        'filters.from_date': 'From Date',
        'filters.to_date': 'To Date',
        'filters.clear': 'Clear Filters',
        'filters.month_view': 'Month View',
        'filters.custom_range': 'Custom Range',
        'filters.month': 'Month',
        'filters.year': 'Year',
        'filters.search': 'Search',
        'filters.search_placeholder': 'Search in descriptions...',

        // Table Headers
        'table.date': 'Date',
        'table.user': 'User',
        'table.type': 'Type',
        'table.description': 'Description',
        'table.amount': 'Amount',

        // Dashboard
        'dashboard.view_all_actions': 'View All Actions',

        // Empty States
        'empty.no_actions': 'No actions yet',
        'empty.click_add': 'Click the + button to add your first action',
        'empty.no_actions_found': 'No actions found',
        'empty.adjust_filters': 'Try adjusting your filters',
        'empty.loading_chart': 'Loading chart data...',

        // Add Action Modal
        'modal.add_action': 'Add Action',
        'modal.edit_action': 'Edit Action',
        'modal.type': 'Type',
        'modal.date': 'Date',
        'modal.description': 'Description',
        'modal.amount': 'Amount',
        'modal.submit': 'Add Action',
        'modal.save': 'Save Changes',
        'modal.delete': 'Delete',
        'modal.cancel': 'Cancel',
        'modal.delete_confirm': 'Are you sure you want to delete this action? This cannot be undone.',

        // Charts
        'charts.title': 'Monthly Income & Expenses',
        'charts.income': 'Income',
        'charts.expenses': 'Expenses',

        // Profile
        'profile.title': 'User Profile',
        'profile.username': 'Username',
        'profile.username_note': 'Username cannot be changed',
        'profile.name': 'Name',
        'profile.change_password': 'Change Password',
        'profile.password_note': 'Leave blank to keep current password',
        'profile.current_password': 'Current Password',
        'profile.new_password': 'New Password',
        'profile.password_min_note': 'Minimum 6 characters',
        'profile.confirm_password': 'Confirm New Password',
        'profile.password_required_note': 'Required when changing password',
        'profile.saving': 'Saving...',
        'profile.save': 'Save Changes',

        // Validation Messages
        'validation.name_required': 'Name is required',
        'validation.password_required': 'Current password is required when changing password',
        'validation.password_min': 'New password must be at least 6 characters',
        'validation.passwords_match': 'New passwords do not match',
        'validation.success': 'Profile updated successfully',
        'validation.error': 'Failed to update profile',
        'validation.failed_create': 'Failed to create action',
        'validation.failed_update': 'Failed to update action',
        'validation.failed_delete': 'Failed to delete action',

        // Pagination
        'pagination.previous': 'Previous',
        'pagination.next': 'Next',
        'pagination.showing': 'Showing {{from}}-{{to}} of {{total}} actions',

        // Date Picker
        'months.full': ['January', 'February', 'March', 'April', 'May', 'June',
                       'July', 'August', 'September', 'October', 'November', 'December'],
        'months.short': ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
                        'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'],
        'weekdays': ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'],
        'date_format': 'DD/MM/YYYY'
    },
    el: {
        _meta: {
            code: 'EL',
            name: 'Ελληνικά'
        },
        // Login
        'login.title': 'Εφαρμογή Προϋπολογισμού',
        'login.username': 'Όνομα χρήστη',
        'login.password': 'Κωδικός πρόσβασης',
        'login.button': 'Σύνδεση',
        'login.error': 'Η σύνδεση απέτυχε. Ελέγξτε τα στοιχεία σας.',

        // Navigation
        'nav.budgeting': 'Προϋπολογισμός',
        'nav.all_actions': 'Όλες οι Κινήσεις',
        'nav.charts': 'Γραφήματα',
        'nav.profile': 'Προφίλ',
        'nav.logout': 'Αποσύνδεση',

        // Filters
        'filters.user': 'Χρήστης',
        'filters.all_users': 'Όλοι οι Χρήστες',
        'filters.type': 'Τύπος',
        'filters.all': 'Όλα',
        'filters.income': 'Έσοδο',
        'filters.expense': 'Έξοδο',
        'filters.from_date': 'Από Ημερομηνία',
        'filters.to_date': 'Έως Ημερομηνία',
        'filters.clear': 'Καθαρισμός Φίλτρων',
        'filters.month_view': 'Προβολή Μήνα',
        'filters.custom_range': 'Προσαρμοσμένο Εύρος',
        'filters.month': 'Μήνας',
        'filters.year': 'Έτος',
        'filters.search': 'Αναζήτηση',
        'filters.search_placeholder': 'Αναζήτηση στις περιγραφές...',

        // Table Headers
        'table.date': 'Ημερομηνία',
        'table.user': 'Χρήστης',
        'table.type': 'Τύπος',
        'table.description': 'Περιγραφή',
        'table.amount': 'Ποσό',

        // Dashboard
        'dashboard.view_all_actions': 'Προβολή Όλων των Κινήσεων',

        // Empty States
        'empty.no_actions': 'Δεν υπάρχουν κινήσεις ακόμα',
        'empty.click_add': 'Κάντε κλικ στο κουμπί + για να προσθέσετε την πρώτη σας κίνηση',
        'empty.no_actions_found': 'Δεν βρέθηκαν κινήσεις',
        'empty.adjust_filters': 'Δοκιμάστε να προσαρμόσετε τα φίλτρα σας',
        'empty.loading_chart': 'Φόρτωση δεδομένων γραφήματος...',

        // Add Action Modal
        'modal.add_action': 'Προσθήκη Κίνησης',
        'modal.edit_action': 'Επεξεργασία Κίνησης',
        'modal.type': 'Τύπος',
        'modal.date': 'Ημερομηνία',
        'modal.description': 'Περιγραφή',
        'modal.amount': 'Ποσό',
        'modal.submit': 'Προσθήκη Κίνησης',
        'modal.save': 'Αποθήκευση Αλλαγών',
        'modal.delete': 'Διαγραφή',
        'modal.cancel': 'Ακύρωση',
        'modal.delete_confirm': 'Είστε σίγουροι ότι θέλετε να διαγράψετε αυτή την κίνηση; Αυτό δεν μπορεί να αναιρεθεί.',

        // Charts
        'charts.title': 'Μηνιαία Έσοδα & Έξοδα',
        'charts.income': 'Έσοδα',
        'charts.expenses': 'Έξοδα',

        // Profile
        'profile.title': 'Προφίλ Χρήστη',
        'profile.username': 'Όνομα χρήστη',
        'profile.username_note': 'Το όνομα χρήστη δεν μπορεί να αλλάξει',
        'profile.name': 'Όνομα',
        'profile.change_password': 'Αλλαγή Κωδικού',
        'profile.password_note': 'Αφήστε κενό για να κρατήσετε τον τρέχοντα κωδικό',
        'profile.current_password': 'Τρέχων Κωδικός',
        'profile.new_password': 'Νέος Κωδικός',
        'profile.password_min_note': 'Ελάχιστο 6 χαρακτήρες',
        'profile.confirm_password': 'Επιβεβαίωση Νέου Κωδικού',
        'profile.password_required_note': 'Απαιτείται κατά την αλλαγή κωδικού',
        'profile.saving': 'Αποθήκευση...',
        'profile.save': 'Αποθήκευση Αλλαγών',

        // Validation Messages
        'validation.name_required': 'Το όνομα είναι υποχρεωτικό',
        'validation.password_required': 'Ο τρέχων κωδικός απαιτείται για αλλαγή κωδικού',
        'validation.password_min': 'Ο νέος κωδικός πρέπει να έχει τουλάχιστον 6 χαρακτήρες',
        'validation.passwords_match': 'Οι νέοι κωδικοί δεν ταιριάζουν',
        'validation.success': 'Το προφίλ ενημερώθηκε επιτυχώς',
        'validation.error': 'Αποτυχία ενημέρωσης προφίλ',
        'validation.failed_create': 'Αποτυχία δημιουργίας κίνησης',
        'validation.failed_update': 'Αποτυχία ενημέρωσης κίνησης',
        'validation.failed_delete': 'Αποτυχία διαγραφής κίνησης',

        // Pagination
        'pagination.previous': 'Προηγούμενο',
        'pagination.next': 'Επόμενο',
        'pagination.showing': 'Εμφάνιση {{from}}-{{to}} από {{total}} κινήσεις',

        // Date Picker
        'months.full': ['Ιανουάριος', 'Φεβρουάριος', 'Μάρτιος', 'Απρίλιος', 'Μάιος', 'Ιούνιος',
                       'Ιούλιος', 'Αύγουστος', 'Σεπτέμβριος', 'Οκτώβριος', 'Νοέμβριος', 'Δεκέμβριος'],
        'months.short': ['Ιαν', 'Φεβ', 'Μαρ', 'Απρ', 'Μαϊ', 'Ιουν',
                        'Ιουλ', 'Αυγ', 'Σεπ', 'Οκτ', 'Νοε', 'Δεκ'],
        'weekdays': ['Δε', 'Τρ', 'Τε', 'Πε', 'Πα', 'Σα', 'Κυ'],
        'date_format': 'ΗΗ/ΜΜ/ΕΕΕΕ'
    }
};

// Helper function to get translation
function t(key, replacements = {}) {
    const lang = state.language || 'en';
    let translation = translations[lang]?.[key] || translations['en'][key] || key;

    // Simple template replacement for {{placeholder}}
    Object.keys(replacements).forEach(placeholder => {
        translation = translation.replace(`{{${placeholder}}}`, replacements[placeholder]);
    });

    return translation;
}

// Helper for array translations (months, weekdays)
function ta(key) {
    const lang = state.language || 'en';
    return translations[lang]?.[key] || translations['en'][key] || [];
}

// Get language metadata
function getLangMeta(lang) {
    return translations[lang]?._meta || translations['en']._meta;
}

// Get list of available languages
function getAvailableLanguages() {
    return Object.keys(translations);
}
