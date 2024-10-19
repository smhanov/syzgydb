export default {
    name: 'CollectionCard',
    props: ['collection'],
    template: `
        <router-link :to="'/collection/' + collection.name" class="bg-gray-800 p-4 rounded-lg shadow-md hover:shadow-lg transition-shadow collection-card">
            <h3 class="text-lg font-semibold mb-2">{{ collection.name }}</h3>
            <p>Documents: {{ collection.document_count }}</p>
            <p>Size: {{ formatSize(collection.storage_size) }}</p>
        </router-link>
    `,
    methods: {
        formatSize(bytes) {
            const units = ['B', 'KB', 'MB', 'GB', 'TB'];
            let size = bytes;
            let unitIndex = 0;
            while (size >= 1024 && unitIndex < units.length - 1) {
                size /= 1024;
                unitIndex++;
            }
            return `${size.toFixed(2)} ${units[unitIndex]}`;
        },
    },
};
