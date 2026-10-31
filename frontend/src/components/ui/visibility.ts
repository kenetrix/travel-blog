import type { Visibility } from '../stores/store'
import { t } from '../../i18n'

// Options of the "Only me / Public" segmented control. Call inside a
// computed, so the labels follow the UI language.
export function visibilityOptions(withTitles = false): {
    value: Visibility
    label: string
    title?: string
}[] {
    return [
        {
            value: 'private',
            label: t('common.onlyMe'),
            title: withTitles ? t('country.privateTitle') : undefined,
        },
        {
            value: 'public',
            label: t('common.public'),
            title: withTitles ? t('country.publicTitle') : undefined,
        },
    ]
}
