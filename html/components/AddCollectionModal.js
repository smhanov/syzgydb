import { ref } from 'vue';

export default {
    name: 'AddCollectionModal',
    emits: ['close', 'add'],
    setup(props, { emit }) {
        const name = ref('');
        const vectorSize = ref(128);
        const quantization = ref(64);
        const distanceFunction = ref('cosine');

        const handleSubmit = () => {
            emit('add', {
                name: name.value,
                vector_size: vectorSize.value,
                quantization: quantization.value,
                distance_function: distanceFunction.value,
            });
        };

        return {
            name,
            vectorSize,
            quantization,
            distanceFunction,
            handleSubmit,
        };
    },
    template: `
        <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
            <div class="bg-gray-800 p-6 rounded-lg">
                <h3 class="text-xl font-bold mb-4">Add New Collection</h3>
                <form @submit.prevent="handleSubmit">
                    <div class="mb-4">
                        <label class="block mb-2">Name</label>
                        <input v-model="name" required class="w-full bg-gray-700 p-2 rounded">
                    </div>
                    <div class="mb-4">
                        <label class="block mb-2">Vector Size</label>
                        <input v-model.number="vectorSize" type="number" required class="w-full bg-gray-700 p-2 rounded">
                    </div>
                    <div class="mb-4">
                        <label class="block mb-2">Quantization</label>
                        <select v-model.number="quantization" required class="w-full bg-gray-700 p-2 rounded">
                            <option value="4">4</option>
                            <option value="8">8</option>
                            <option value="16">16</option>
                            <option value="32">32</option>
                            <option value="64">64</option>
                        </select>
                    </div>
                    <div class="mb-4">
                        <label class="block mb-2">Distance Function</label>
                        <select v-model="distanceFunction" required class="w-full bg-gray-700 p-2 rounded">
                            <option value="cosine">Cosine</option>
                            <option value="euclidean">Euclidean</option>
                        </select>
                    </div>
                    <div class="flex justify-end">
                        <button type="button" @click="$emit('close')" class="bg-gray-600 hover:bg-gray-700 text-white font-bold py-2 px-4 rounded mr-2">
                            Cancel
                        </button>
                        <button type="submit" class="bg-accent-violet hover:bg-accent-blue text-white font-bold py-2 px-4 rounded">
                            Add Collection
                        </button>
                    </div>
                </form>
            </div>
        </div>
    `,
};
