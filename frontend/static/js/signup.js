import { postJSON } from './api.js';

export function setupSignup() {
    const form = document.querySelector('#signup-form');
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
        [form.username, form.email, form.first_name, form.last_name, form.password].forEach(clearError);
    };

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        clearAllErrors();

        const payload = {
            username: form.username.value,
            email: form.email.value,
            first_name: form.first_name.value,
            last_name: form.last_name.value,
            password: form.password.value,
        };

        try {
            await postJSON('/create-account', payload);
            window.location.href = '/contacts';
        } catch (err) {
            const msg = err.message;

            if (msg.includes('Username already taken')) {
                showError(form.username, 'Username is already taken', 'username-error');
            } else if (msg.includes('Email already in use')) {
                showError(form.email, 'Email is already taken', 'email-error');
            } else {
                alert(`Signup failed: ${msg}`);
            }
        }
    });
}
