import type { Directive } from 'vue'

// Fields Enter can move between; checkboxes, radios and buttons keep their
// own behaviour. Hidden, disabled and read-only fields are skipped.
const FIELDS =
    'input:not([type=hidden]):not([type=checkbox]):not([type=radio]):not([type=submit]):not([type=button]), select, textarea'

function fieldsOf(form: HTMLElement): HTMLElement[] {
    return [...form.querySelectorAll<HTMLElement>(FIELDS)].filter(
        (el) =>
            !(el as HTMLInputElement).disabled &&
            !(el as HTMLInputElement).readOnly &&
            el.offsetParent !== null
    )
}

// Focuses the field after `from` in its form. False if it was the last one.
export function focusNext(from: HTMLElement): boolean {
    const form = from.closest('form')
    if (!form) return false
    const fields = fieldsOf(form)
    const next = fields[fields.indexOf(from) + 1]
    if (!next) return false
    next.focus()
    return true
}

function onKeydown(e: KeyboardEvent) {
    if (e.key !== 'Enter' || e.defaultPrevented || e.isComposing) return
    const target = e.target
    // A textarea needs Enter for new lines
    if (
        !(
            target instanceof HTMLInputElement ||
            target instanceof HTMLSelectElement
        ) ||
        !target.matches(FIELDS)
    ) {
        return
    }
    // Enter in the last field submits the form as usual
    if (focusNext(target)) e.preventDefault()
}

// v-enter-next on a <form>: Enter moves to the next field, and only the
// last field submits.
export const vEnterNext: Directive<HTMLElement> = {
    mounted: (form) => form.addEventListener('keydown', onKeydown),
    beforeUnmount: (form) => form.removeEventListener('keydown', onKeydown),
}
