/**
 * Composable for wizard steps that have a single select control.
 * When the user selects a value from the only select in the step,
 * automatically advance to the next step.
 *
 * Usage:
 *   const { onSelectAutoNext } = useWizardAutoAdvance(step, () => { step.value++ })
 *   // In template: @change="onSelectAutoNext"
 */
import { type Ref } from 'vue'

export function useWizardAutoAdvance(
  currentStep: Ref<number>,
  advance: () => void,
) {
  /**
   * Call this from a select @change handler.
   * Advances to the next step if a value was selected.
   */
  function onSelectAutoNext(value: any) {
    if (value !== undefined && value !== null && value !== '') {
      advance()
    }
  }

  return { onSelectAutoNext }
}
