import { createRouter, createWebHistory } from 'vue-router'
import TripPage from '../components/trips/TripPage.vue'
import AuthPage from '../components/account/AuthPage.vue'
import ProfilePage from '../components/account/ProfilePage.vue'
import PasswordPage from '../components/account/PasswordPage.vue'
import AdminPage from '../components/admin/AdminPage.vue'

export const router = createRouter({
    history: createWebHistory(),
    routes: [
        // The map is always rendered by App; the home route adds nothing on top
        { path: '/', name: 'home', component: { render: () => null } },
        {
            path: '/countries/:iso/trips/new',
            name: 'trip-new',
            component: TripPage,
        },
        { path: '/login', name: 'login', component: AuthPage },
        { path: '/account', name: 'account', component: ProfilePage },
        { path: '/profile', redirect: { name: 'account' } },
        { path: '/password', name: 'password', component: PasswordPage },
        { path: '/admin', name: 'admin', component: AdminPage },
        { path: '/:pathMatch(.*)*', redirect: { name: 'home' } },
    ],
})
