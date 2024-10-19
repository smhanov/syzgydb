import { inject } from 'vue';
import { NotificationSymbol } from './components/Notification.js';

const API_BASE_URL = '';

async function fetchJson(url, options = {}) {
    const showNotification = inject(NotificationSymbol);
    try {
        const response = await fetch(url, options);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
    } catch (error) {
        showNotification(error.message, 'error');
        throw error;
    }
}

export async function fetchCollections() {
    return fetchJson(`${API_BASE_URL}/api/v1/collections`);
}

export async function fetchCollectionInfo(collectionName) {
    return fetchJson(`${API_BASE_URL}/api/v1/collections/${collectionName}`);
}

export async function createCollection(collectionData) {
    return fetchJson(`${API_BASE_URL}/api/v1/collections`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(collectionData),
    });
}

export async function searchRecords(collectionName, searchParams) {
    return fetchJson(`${API_BASE_URL}/api/v1/collections/${collectionName}/search`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(searchParams),
    });
}
