import { ContactFormSetup } from "./util.js";

export function setupContact() {
    ContactFormSetup();

    const deleteBtn = document.querySelector("#delete-contact");

    deleteBtn?.addEventListener("click", async (e) => {
        e.preventDefault();

        const id = deleteBtn.dataset.id;

        const confirmed = confirm("Are you sure you want to delete this contact?");
        if (!confirmed) return;

        try {
            const res = await fetch(`/contacts/${id}`, {
                method: "DELETE",
            });

            if (!res.ok) {
                const text = await res.text();
                throw new Error(text);
            }

            window.location.href = "/contacts";
        } catch (err) {
            alert("Failed to delete contact: " + err.message);
        }
    });
}