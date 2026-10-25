<script setup lang="ts">
import { computed } from 'vue'
import { useTravelStore } from '../stores/store'

const emit = defineEmits<{ say: [text: string, error?: boolean] }>()
const store = useTravelStore()
const DEFAULT_COLOR = '#3a7d5c'

const isPublic = computed(() => store.settings.profileVisibility === 'public')
const color = computed(() => store.profile?.color || DEFAULT_COLOR)
const initial = computed(() =>
    (store.profile?.displayName || store.profile?.username || '?')
        .charAt(0)
        .toUpperCase()
)

// The store rolls back if the server rejects the change
async function toggle() {
    const failed = await store.saveSettings({
        ...store.settings,
        profileVisibility: isPublic.value ? 'private' : 'public',
    })
    if (failed) emit('say', failed, true)
    else
        emit('say', isPublic.value ? 'Profile is public' : 'Profile is private')
}

async function setColor(value: string) {
    const failed = await store.updateProfile({ color: value })
    if (failed) emit('say', failed, true)
    else emit('say', 'Colour saved')
}
</script>

<template>
    <section class="panel">
        <div class="head">
            <h2 id="visibility-title">Public profile</h2>
            <button
                type="button"
                class="switch"
                :class="{ on: isPublic }"
                role="switch"
                :aria-checked="isPublic"
                aria-labelledby="visibility-title"
                @click="toggle"
            >
                <span class="knob" />
            </button>
        </div>
        <p class="note">
            <template v-if="isPublic">
                Anyone can see your display name, bio, home country and
                everything marked 🌐 Public.
            </template>
            <template v-else>
                Nobody else can see anything, even items marked 🌐 Public.
            </template>
        </p>

        <div class="head divider">
            <h2 id="color-title">Profile colour</h2>
            <label
                class="avatar"
                :style="{ background: color }"
                title="Change colour"
            >
                {{ initial }}
                <input
                    type="color"
                    aria-labelledby="color-title"
                    :value="color"
                    @change="
                        setColor(($event.target as HTMLInputElement).value)
                    "
                />
            </label>
        </div>
    </section>
</template>

<style src="../ui/panel.css"></style>
<style scoped>
.head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
}

.divider {
    margin-top: 1.1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--line);
}

.panel .head h2 {
    margin: 0;
}

.avatar {
    position: relative;
    display: grid;
    flex: none;
    place-items: center;
    width: 2.4rem;
    height: 2.4rem;
    cursor: pointer;
    border-radius: 50%;
    color: #fff;
    font-size: 1.2rem;
    font-weight: 500;
}

.avatar:hover {
    filter: brightness(0.9);
}

/* The native picker covers the circle, invisible, so a click opens it */
.panel .avatar input {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    padding: 0;
    border: none;
    opacity: 0;
    cursor: pointer;
}

.panel .note {
    margin-top: 0.5rem;
}

.switch {
    position: relative;
    flex: none;
    width: 2.6rem;
    height: 1.5rem;
    padding: 0;
    border: none;
    border-radius: 1rem;
    background: var(--line-strong);
    cursor: pointer;
    transition: background 0.15s;
}

.switch.on {
    background: #3a7d5c;
}

.knob {
    position: absolute;
    top: 0.2rem;
    left: 0.2rem;
    width: 1.1rem;
    height: 1.1rem;
    border-radius: 50%;
    background: #fff;
    transition: transform 0.15s;
}

.switch.on .knob {
    transform: translateX(1.1rem);
}
</style>
