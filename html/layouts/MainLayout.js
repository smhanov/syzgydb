export default {
    name: 'MainLayout',
    template: `
        <div class="min-h-screen flex flex-col">
            <header class="bg-gray-800 p-4">
                <div class="container mx-auto flex justify-between items-center">
                    <h1 class="text-2xl font-bold text-accent-blue">SyzgyDB</h1>
                    <nav>
                        <router-link to="/" class="text-accent-violet hover:text-accent-blue">Collections</router-link>
                    </nav>
                </div>
            </header>
            <main class="flex-grow container mx-auto p-4">
                <router-view></router-view>
            </main>
            <Notification />
        </div>
    `,
};
