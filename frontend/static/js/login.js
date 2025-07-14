import { postJSON } from './api.js';

export function setupLogin() {
    const form = document.querySelector('#login-form');
    if (!form) return;

    const showError = (input, message, id) => {
        clearError(input);

        input.setAttribute('aria-invalid', 'true');
        input.setAttribute('aria-describedby', id);

        const errorEl = document.createElement('small');
        errorEl.id = id;
        errorEl.textContent = message;

        input.insertAdjacentElement('afterend', errorEl);
    };

    const clearError = (input) => {
        input.removeAttribute('aria-invalid');
        input.removeAttribute('aria-describedby');

        const next = input.nextElementSibling;
        if (next && next.tagName === 'SMALL') {
            next.remove();
        }
    };

    const clearAllErrors = () => {
        [form.email, form.password].forEach(clearError);
    };

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        clearAllErrors();

        const payload = {
            email: form.email.value,
            password: form.password.value,
        };

        try {
            await postJSON('/login', payload);
            window.location.href = '/';
        } catch (err) {
            const msg = err.message.toLowerCase();

            if (msg.includes('User not found')) {
                showError(form.email, 'Email not found or invalid', 'email-error');
            } else if (msg.includes('Invalid password')) {
                showError(form.password, 'Incorrect password', 'password-error');
            } else {
                alert(`Login failed: ${err.message}`);
            }
        }
    });
}
