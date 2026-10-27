import { createRouter, createWebHistory } from 'vue-router'
import TripPage from '../components/trips/TripPage.vue'
import TripEditor from '../components/trips/TripEditor.vue'
import AuthPage from '../components/account/AuthPage.vue'
import ProfilePage from '../components/account/ProfilePage.vue'
import PasswordPage from '../components/account/PasswordPage.vue'
import AdminPage from '../components/admin/AdminPage.vue'

export const router = createRouter({
    history: createWebHistory(),
    routes: [
        // The map is always rendered by App; the home route adds nothing on top
        { path: '/', name: 'home', component: { render: () => null } },
        // ?country=<alpha-3>: the country the trip is started from
        { path: '/trips/new', name: 'trip-new', component: TripEditor },
        { path: '/trips/:id', name: 'trip', component: TripPage },
        { path: '/trips/:id/edit', name: 'trip-edit', component: TripEditor },
        { path: '/login', name: 'login', component: AuthPage },
        { path: '/account', name: 'account', component: ProfilePage },
        { path: '/profile', redirect: { name: 'account' } },
        { path: '/password', name: 'password', component: PasswordPage },
        { path: '/admin', name: 'admin', component: AdminPage },
        { path: '/:pathMatch(.*)*', redirect: { name: 'home' } },
    ],
})
