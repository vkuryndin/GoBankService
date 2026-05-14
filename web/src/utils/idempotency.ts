import { createPrefixedRandomID } from './random'

export function createIdempotencyKey(prefix = 'web'): string {
  return createPrefixedRandomID(prefix)
}
