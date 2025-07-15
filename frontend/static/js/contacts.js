import { ContactFormSetup } from './util.js';

export function setupContacts() {
    let page = 1;
    let afterCursor = null;
    let searchTerm = '';
    let filterField = '';

    const tableBody = document.querySelector("#contacts-table tbody");
    const searchInput = document.querySelector("#contact-search");
    const filterSelect = document.querySelector("#filter-field");

    const prevBtn = document.querySelector("#prev-btn");
    const nextBtn = document.querySelector("#next-btn");
    const pageNumberEl = document.querySelector("#page-number");

    const cursors = [null];

    ContactFormSetup(fetchContacts);

    searchInput?.addEventListener("input", debounce(() => {
        searchTerm = searchInput.value.trim();
        page = 1;
        cursors.length = 1;
        cursors[0] = null;
        fetchContacts();
    }, 400));

    filterSelect?.addEventListener("change", () => {
        filterField = filterSelect.value;
        page = 1;
        cursors.length = 1;
        cursors[0] = null;
        fetchContacts();
    });

    prevBtn.addEventListener("click", () => {
        if (page <= 1) return;
        page--;
        fetchContacts(cursors[page - 1]);
    });

    nextBtn.addEventListener("click", () => {
        if (!cursors[page]) return;
        page++;
        fetchContacts(cursors[page - 1]);
    });

    async function fetchContacts(after = null) {
        let url = `/contacts/all?limit=10`;

        if (searchTerm) url += `&search=${encodeURIComponent(searchTerm)}`;

        const fieldMap = {
            email: "require_non_empty_email",
            phone: "require_non_empty_phone",
            company: "require_non_empty_company",
            position: "require_non_empty_position",
        };

        if (filterField && fieldMap[filterField]) {
            url += `&${fieldMap[filterField]}=true`;
        }

        if (after) url += `&after=${after}`;

        try {
            const res = await fetch(url);
            const { contacts, next_cursor } = await res.json();

            tableBody.innerHTML = "";

            for (const contact of contacts) {
                const tr = document.createElement("tr");
                tr.innerHTML = `
          <td>${contact.name}</td>
          <td>${contact.email || ""}</td>
          <td>${contact.company || ""}</td>
          <td><a href="/contacts/${contact['contact_id']}" class="secondary">Details</a></td>
        `;
                tableBody.appendChild(tr);
            }

            // Pagination state
            if (page >= cursors.length) cursors.push(next_cursor || null);
            pageNumberEl.textContent = `Page ${page}`;
            prevBtn.disabled = page <= 1;
            nextBtn.disabled = !next_cursor;
        } catch (err) {
            tableBody.innerHTML = `<tr><td colspan="4">Failed to load contacts.</td></tr>`;
        }
    }

    fetchContacts();
}

// Utility debounce
function debounce(fn, delay) {
    let timeout;
    return (...args) => {
        clearTimeout(timeout);
        timeout = setTimeout(() => fn(...args), delay);
    };
}
