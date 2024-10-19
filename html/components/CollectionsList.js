import { ref, onMounted } from 'vue';
import { fetchCollections, createCollection } from '../api.js';
import CollectionCard from './CollectionCard.js';
import AddCollectionModal from './AddCollectionModal.js';

export default {
    name: 'CollectionsList',
    components: {
        CollectionCard,
        AddCollectionModal,
    },
    setup() {
        const collections = ref([]);
        const showModal = ref(false);

        const loadCollections = async () => {
            collections.value = await fetchCollections();
            console.log("Loaded collections");
        };

        const handleAddCollection = async (collectionData) => {
            await createCollection(collectionData);
            await loadCollections();
            showModal.value = false;
        };

        onMounted(loadCollections);

        return {
            collections,
            showModal,
            handleAddCollection,
        };
    },
    template: `
        <div>
            <h2 class="text-xl font-bold mb-4">Collections</h2>
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                <CollectionCard v-for="collection in collections" :key="collection.name" :collection="collection" />
                <button @click="showModal = true" class="bg-accent-violet hover:bg-accent-blue text-white font-bold py-2 px-4 rounded flex items-center justify-center">
                    <span class="mr-2">&#10133;</span> Add Collection
                </button>
            </div>
            <AddCollectionModal v-if="showModal" @close="showModal = false" @add="handleAddCollection" />
        </div>
    `,
};
