// A message with one form per plural category of the language; `other` is
// the fallback for categories a language does not list
export type Plural = Partial<Record<Intl.LDMLPluralRule, string>> & {
    other: string
}

export function plural<T extends Plural>(forms: T): Plural {
    return forms
}
