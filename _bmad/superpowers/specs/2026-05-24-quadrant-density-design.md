# Quadrant View Task Density Optimization

**Date:** 2026-05-24
**Status:** approved

## Problem

Each quadrant currently shows only ~3 tasks due to large TaskCard (~118px/card). Users with more tasks must scroll.

## Solution

Add `mode` prop to TaskCard: `'card'` (existing, ListView) and `'row'` (new, QuadrantView). Row mode shows compact horizontal rows with hover popover for full details.

## Design

### TaskCard.vue

- Add `mode` prop: `'card' | 'row'`, default `'card'`
- Row mode: horizontal flex row — checkbox | title (flex:1) | estimated time pill | deadline pill | menu (⋮)
- ~30px/row → 8-10 visible tasks per quadrant
- Checkbox click: toggle complete/reopen
- Row click: open edit dialog
- Hover: el-popover shows description + all tags + status
- Card mode: unchanged

### QuadrantView.vue

- Pass `mode="row"` to TaskCard
- Reduce gap: 10px → 2px
- Reduce padding: 20px → 16px

### Backward compatibility

- ListView uses default `mode="card"`, unchanged
- All TaskCard events unchanged

## Files

- `frontend/src/components/tasks/TaskCard.vue`
- `frontend/src/components/tasks/QuadrantView.vue`

## Not in scope

- ListView changes
- Server/database changes
