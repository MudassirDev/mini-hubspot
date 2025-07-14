import { setupLogin } from './login.js';
import { setupSignup } from './signup.js';

document.addEventListener('DOMContentLoaded', () => {
    const page = document.body.querySelector("#content")?.dataset.page;

    if (page === 'login') setupLogin();
    if (page === 'signup') setupSignup();
});
