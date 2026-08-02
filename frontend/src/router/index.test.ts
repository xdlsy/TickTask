import { describe, it, expect } from 'vitest'
import router from './index'

describe('Router', () => {
  it('should create router with history mode', () => {
    expect(router).toBeDefined()
  })

  it('should have all routes defined', () => {
    const routes = router.getRoutes()
    const names = routes.map(r => r.name)
    expect(names).toContain('Dashboard')
    expect(names).toContain('Timer')
    expect(names).toContain('Tasks')
    expect(names).toContain('Schedule')
    expect(names).toContain('Analytics')
    expect(names).toContain('Settings')
  })

  it('should redirect / to /dashboard', () => {
    const routes = router.getRoutes()
    const root = routes.find(r => r.path === '/')
    expect(root).toBeDefined()
    expect(root?.redirect).toBe('/dashboard')
  })

  it('should resolve dashboard route', () => {
    const resolved = router.resolve('/dashboard')
    expect(resolved.name).toBe('Dashboard')
  })

  it('should resolve timer route', () => {
    const resolved = router.resolve('/timer')
    expect(resolved.name).toBe('Timer')
  })

  it('should resolve schedule route', () => {
    const resolved = router.resolve('/schedule')
    expect(resolved.name).toBe('Schedule')
  })
})
