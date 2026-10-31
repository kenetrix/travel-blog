<script setup lang="ts">
import { computed } from 'vue'
import { useTravelStore } from '../stores/store'
import { t } from '../../i18n'

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
        emit(
            'say',
            isPublic.value ? t('profile.isPublic') : t('profile.isPrivate')
        )
}

async function setColor(value: string) {
    const failed = await store.updateProfile({ color: value })
    if (failed) emit('say', failed, true)
    else emit('say', t('profile.colourSaved'))
}
</script>

<template>
    <section class="panel">
        <!-- The same switch as the administration's settings -->
        <label class="check toggle">
            <span class="title">{{ t('profile.publicProfile') }} 🌐</span>
            <input
                type="checkbox"
                role="switch"
                class="switch"
                :checked="isPublic"
                :disabled="!store.canWrite"
                @change="toggle"
            />
        </label>
        <div class="head divider">
            <h2 id="color-title">{{ t('profile.colour') }}</h2>
            <label
                class="avatar"
                :style="{ background: color }"
                :title="t('profile.changeColour')"
            >
                {{ initial }}
                <input
                    type="color"
                    :disabled="!store.canWrite"
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

/* The panel's muted label colour would otherwise win */
.panel label.avatar {
    margin-top: 0;
    color: #fff;
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

.title {
    font-size: 1.05rem;
    font-weight: 600;
    color: var(--text);
}
</style>
