import { ref, provide, inject } from 'vue';

export const NotificationSymbol = Symbol();

export function useNotification() {
    const message = ref('');
    const show = ref(false);
    const type = ref('info');

    const showNotification = (msg, notificationType = 'info') => {
        message.value = msg;
        type.value = notificationType;
        show.value = true;
        setTimeout(() => {
            show.value = false;
        }, 3000);
    };

    provide(NotificationSymbol, showNotification);

    return {
        message,
        show,
        type,
        showNotification,
    };
}

export default {
    setup() {
        const { message, show, type } = useNotification();

        return {
            message,
            show,
            type,
        };
    },
    template: `
        <div v-if="show" :class="['fixed top-4 right-4 p-4 rounded-lg shadow-lg', {
            'bg-green-500': type === 'success',
            'bg-red-500': type === 'error',
            'bg-blue-500': type === 'info'
        }]">
            {{ message }}
        </div>
    `,
};
