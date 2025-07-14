export async function postJSON(url, data) {
    const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });

    if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Something went wrong');
    }

    return res.json();
}