<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import PageWrapper from '../ui/PageWrapper.vue'
import { useTravelStore } from '../stores/store'
import { vEnterNext } from '../ui/enterNext'

const store = useTravelStore()
const router = useRouter()

const currentPassword = ref('')
const newPassword = ref('')
const repeat = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
    error.value = ''
    if (newPassword.value !== repeat.value) {
        error.value = 'The new passwords do not match'
        return
    }
    busy.value = true
    const failed = await store.changePassword(
        currentPassword.value,
        newPassword.value
    )
    if (failed) {
        busy.value = false
        error.value = failed
        return
    }
    // The server no longer requires a change; load the account again
    await store.load()
    busy.value = false
    router.push({ name: 'home' })
}
</script>

<template>
    <PageWrapper
        title="Choose a new password"
        subtitle="Your password was generated or reset. Pick your own to continue."
        :closable="false"
    >
        <form v-enter-next class="card" @submit.prevent="submit">
            <label for="current">Current password</label>
            <input
                id="current"
                v-model="currentPassword"
                type="password"
                autocomplete="current-password"
                required
            />
            <label for="new">New password</label>
            <input
                id="new"
                v-model="newPassword"
                type="password"
                autocomplete="new-password"
                minlength="10"
                required
            />
            <p class="note">At least 10 characters.</p>
            <label for="repeat">Repeat the new password</label>
            <input
                id="repeat"
                v-model="repeat"
                type="password"
                autocomplete="new-password"
                minlength="10"
                required
            />

            <p v-if="error" class="error" role="alert">{{ error }}</p>

            <button class="btn" type="submit" :disabled="busy">
                Change password
            </button>
            <button class="btn secondary" type="button" @click="store.logout()">
                Log out
            </button>
        </form>
    </PageWrapper>
</template>

<style scoped>
.card {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    max-width: 360px;
    margin: 1rem auto;
}

label {
    margin-top: 0.4rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--muted);
}

input {
    box-sizing: border-box;
    width: 100%;
    padding: 0.5rem 0.6rem;
    border: 1px solid var(--line-strong);
    border-radius: 0.5rem;
    background: var(--raised);
    color: var(--text);
    font: inherit;
    font-size: 0.85rem;
}

.note {
    margin: 0;
    font-size: 0.72rem;
    color: var(--muted);
}

.error {
    margin: 0.4rem 0 0;
    font-size: 0.78rem;
    color: #c0392b;
}

.btn {
    margin-top: 0.9rem;
    padding: 0.6rem;
    border: none;
    border-radius: 0.5rem;
    background: #3a7d5c;
    color: #fff;
    font: inherit;
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
}

.btn.secondary {
    margin-top: 0.4rem;
    border: 1px solid var(--line-strong);
    background: var(--raised);
    color: var(--text);
}

.btn:disabled {
    opacity: 0.6;
    cursor: wait;
}
</style>
