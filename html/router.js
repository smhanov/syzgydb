import { createRouter, createWebHashHistory } from 'vue-router';
import CollectionsList from './components/CollectionsList.js';
import CollectionView from './components/CollectionView.js';

const routes = [
    { path: '/', component: CollectionsList },
    { path: '/collection/:id', component: CollectionView, props: true },
];

const router = createRouter({
    history: createWebHashHistory(),
    routes,
});

export default router;
