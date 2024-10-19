import { createApp, h } from 'vue';
import Router from './router.js';
import MainLayout from './layouts/MainLayout.js';
import Notification, { useNotification } from './components/Notification.js';

console.log("Executing app.js");
const app = createApp({
    setup() {
        useNotification();
        return () => h(MainLayout);
    },
});

app.use(Router);
app.component('Notification', Notification);
app.mount('#app');
