import { ref, onMounted, computed, watch } from 'vue';
import { useRoute } from 'vue-router';
import { fetchCollectionInfo, searchRecords } from '../api.js';
import { useNotification } from './Notification.js';

export default {
    name: 'CollectionView',
    setup() {
        const route = useRoute();
        const collection = ref(null);
        const records = ref([]);
        const searchQuery = ref('');
        const loading = ref(false);
        const searching = ref(false);
        const page = ref(1);
        const pageSize = ref(20);
        const searchStats = ref(null);
        const tempRecords = ref([]);
        const exactPrecision = ref(false);

        const collectionName = computed(() => route.params.id);

        const loadCollection = async () => {
            collection.value = await fetchCollectionInfo(collectionName.value);
        };

        const loadRecords = async () => {
            loading.value = true;
            searching.value = true;
            const searchParams = {
                text: searchQuery.value,
                limit: pageSize.value,
                offset: (page.value - 1) * pageSize.value,
            };
            if (searchQuery.value.trim() !== '') {
                searchParams.k = 100;
            }
            if (exactPrecision.value) {
                searchParams.precision = "exact";
            }
            console.log("Searching records");
            const searchResult = await searchRecords(collectionName.value, searchParams);
            console.log("Fetched new records: ", searchResult);
            tempRecords.value = searchResult.results;
            searchStats.value = {
                percentSearched: searchResult.percent_searched,
                searchTime: searchResult.search_time,
            };
            loading.value = false;
            searching.value = false;
            records.value = tempRecords.value;
        };

        const debounce = (fn, delay) => {
            let timeoutId;
            return (...args) => {
                clearTimeout(timeoutId);
                timeoutId = setTimeout(() => fn(...args), delay);
            };
        };

        const debouncedSearch = debounce(() => {
            tempRecords.value = [];
            page.value = 1;
            loadRecords();
        }, 500);

        const handleSearch = () => {
            debouncedSearch();
        };

        const loadMore = () => {
            page.value++;
            pageSize.value *= 2;
            loadRecords();
        };

        onMounted(() => {
            loadCollection();
            loadRecords();
        });

        watch(collectionName, () => {
            loadCollection();
            records.value = [];
            tempRecords.value = [];
            page.value = 1;
            loadRecords();
        });

        watch(exactPrecision, () => {
            if (searchQuery.value.trim() !== '') {
                debouncedSearch();
            }
        });

        const copyToClipboard = (record) => {
            const jsonContent = JSON.stringify(record.metadata, null, 2);
            navigator.clipboard.writeText(jsonContent).then(() => {
                showNotification('JSON content copied to clipboard', 'success');
            }).catch((err) => {
                console.error('Failed to copy text: ', err);
                showNotification('Failed to copy JSON content', 'error');
            });
        };

        return {
            collection,
            records,
            searchQuery,
            loading,
            searching,
            handleSearch,
            loadMore,
            searchStats,
            exactPrecision,
            copyToClipboard,
        };
    },
    template: `
        <div v-if="collection">
            <h2 class="text-xl font-bold mb-4">{{ collection.name }}</h2>
            <div class="mb-4 relative">
                <input v-model="searchQuery" @input="handleSearch" placeholder="Search records..." class="bg-gray-700 p-2 rounded w-full">
                <div v-if="searching" class="absolute right-3 top-1/2 transform -translate-y-1/2">
                    <div class="animate-spin rounded-full h-5 w-5 border-t-2 border-b-2 border-accent-violet"></div>
                </div>
            </div>
            <div class="mb-4 flex items-center">
                <input type="checkbox" id="exactPrecision" v-model="exactPrecision" class="mr-2">
                <label for="exactPrecision">Use exact precision</label>
            </div>
            <div v-if="searchStats" class="mb-4 text-sm text-gray-400">
                <p>Percent searched: {{ searchStats.percentSearched.toFixed(2) }}%</p>
                <p>Search time: {{ searchStats.searchTime }}ms</p>
            </div>
            <div class="space-y-4">
                <div v-for="record in records" :key="record.id" class="bg-gray-800 p-4 rounded-lg relative">
                    <button @click="copyToClipboard(record)" class="absolute top-2 right-2 text-accent-violet hover:text-accent-blue">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                        </svg>
                    </button>
                    <p class="font-semibold">ID: {{ record.id }}</p>
                    <p class="text-sm text-gray-400 mb-2">Distance: {{ record.distance.toFixed(4) }}</p>
                    <pre class="mt-2 overflow-x-auto text-xs">{{ JSON.stringify(record.metadata, null, 2) }}</pre>
                </div>
            </div>
            <div v-if="loading" class="text-center mt-4">Loading...</div>
            <button v-if="!loading" @click="loadMore" class="mt-4 bg-accent-violet hover:bg-accent-blue text-white font-bold py-2 px-4 rounded w-full">
                Load More
            </button>
        </div>
    `,
};
