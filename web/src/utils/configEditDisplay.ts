export type ConfigEditViewLevel = 'normal' | 'advanced' | 'developer'
export type Translate = (key: string) => string

export function configEditViewLevelOptions(t: Translate): Array<{ label: string, value: ConfigEditViewLevel }> {
  return [
    { label: t('configEdit.levels.normal'), value: 'normal' },
    { label: t('configEdit.levels.advanced'), value: 'advanced' },
    { label: t('configEdit.levels.developer'), value: 'developer' },
  ]
}

export function configEditViewLevelHelp(t: Translate): string {
  return t('configEdit.levels.help')
}
