// Aggregates all per-theme case files into one flat array.
import { CASES as toolMatrix } from './tool-matrix.mjs';
import { CASES as argFidelity } from './arg-fidelity.mjs';
import { CASES as entityResolution } from './entity-resolution.mjs';
import { CASES as confirmationLifecycle } from './confirmation-lifecycle.mjs';
import { CASES as multiTurn } from './multi-turn.mjs';
import { CASES as idempotency } from './idempotency.mjs';
import { CASES as inputBoundaries } from './input-boundaries.mjs';
import { CASES as safety } from './safety.mjs';
import { CASES as outputQuality } from './output-quality.mjs';
import { CASES as i18nTimezone } from './i18n-timezone.mjs';
import { CASES as domainLogic } from './domain-logic.mjs';
import { CASES as postActionVerify } from './post-action-verify.mjs';
import { CASES as determinism } from './determinism.mjs';
import { CASES as performance } from './performance.mjs';
import { CASES as resilience } from './resilience.mjs';

export const THEME_CASES = [
  ...toolMatrix,
  ...argFidelity,
  ...entityResolution,
  ...confirmationLifecycle,
  ...multiTurn,
  ...idempotency,
  ...inputBoundaries,
  ...safety,
  ...outputQuality,
  ...i18nTimezone,
  ...domainLogic,
  ...postActionVerify,
  ...determinism,
  ...performance,
  ...resilience,
];
