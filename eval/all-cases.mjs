// Single source of truth for ALL eval cases: original 44 (cases.mjs) + the 15
// per-theme files (themes/index.mjs). run-cases.mjs consumes ALL_CASES.
import { buildCases } from './cases.mjs';
import { THEME_CASES } from './themes/index.mjs';

export const ALL_CASES = [...buildCases(), ...THEME_CASES];

// Categories derived from the cases themselves (original 10 + 15 themes).
export const CATEGORIES = [...new Set(ALL_CASES.map((c) => c.cat))];
