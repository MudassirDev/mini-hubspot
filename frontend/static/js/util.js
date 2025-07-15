import { postJSON } from "./api.js";

export function ContactFormSetup(onSuccess) {
    const modal = document.querySelector("#contact-modal");
    const openBtn = document.querySelector("#add-contact");
    const closeBtn = document.querySelector("#cancel-modal");
    const form = document.querySelector("#contact-form");

    openBtn?.addEventListener("click", (e) => {
        e.preventDefault();
        modal.showModal();
    });

    closeBtn?.addEventListener("click", () => modal.close());

    form?.addEventListener("submit", async (e) => {
        e.preventDefault();

        const data = {
            name: form.name.value,
            email: form.email.value,
            phone: form.phone.value,
            company: form.company.value,
            position: form.position.value,
            notes: form.notes.value,
        };

        const isEdit = form.id && form.id.value;
        const endpoint = isEdit
            ? `/contacts/${form.id.value}`
            : `/contacts/new`;

        try {
            if (isEdit) {
                const res = await fetch(endpoint, {
                    method: "PATCH",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(data),
                });
                if (!res.ok) throw new Error(await res.text());
            } else {
                await postJSON(endpoint, data);
            }
            modal.close();
            form.reset();

            if (isEdit) {
                window.location.reload();
            } else if (typeof onSuccess === "function") {
                onSuccess();
            }
        } catch (err) {
            alert("Error saving contact: " + err.message);
        }
    });
}
