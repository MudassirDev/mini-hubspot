import { setupLogin } from './login.js';
import { setupSignup } from './signup.js';
import { setupContacts } from './contacts.js';
import { setupContact } from './contact.js';

document.addEventListener('DOMContentLoaded', () => {
    const page = document.body.querySelector("#content")?.dataset.page;

    if (page === 'login') setupLogin();
    if (page === 'signup') setupSignup();
    if (page === 'contacts') setupContacts();
    if (page === 'contact') setupContact();
});
