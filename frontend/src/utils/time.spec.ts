import { describe, it, expect } from 'vitest'
import { formatTime, formatDuration, formatDate, formatDateTime, getRemainingTime } from '@/utils/time'

describe('Time Utilities', () => {
  describe('formatTime', () => {
    it('formats 0 seconds', () => {
      expect(formatTime(0)).toBe('00:00')
    })

    it('formats seconds less than a minute', () => {
      expect(formatTime(30)).toBe('00:30')
    })

    it('formats exactly one minute', () => {
      expect(formatTime(60)).toBe('01:00')
    })

    it('formats minutes and seconds', () => {
      expect(formatTime(90)).toBe('01:30')
    })

    it('formats 25 minutes (pomodoro)', () => {
      expect(formatTime(1500)).toBe('25:00')
    })

    it('pads single digit seconds', () => {
      expect(formatTime(61)).toBe('01:01')
    })
  })

  describe('formatDuration', () => {
    it('formats seconds less than 60', () => {
      expect(formatDuration(30)).toBe('30秒')
    })

    it('formats 1 minute', () => {
      expect(formatDuration(60)).toBe('1分钟')
    })

    it('formats multiple minutes', () => {
      expect(formatDuration(1500)).toBe('25分钟')
    })

    it('formats 1 hour', () => {
      expect(formatDuration(3600)).toBe('1小时')
    })

    it('formats hours and minutes', () => {
      expect(formatDuration(5400)).toBe('1小时30分钟')
    })

    it('formats multiple hours', () => {
      expect(formatDuration(7200)).toBe('2小时')
    })
  })

  describe('formatDate', () => {
    it('formats ISO date string', () => {
      const result = formatDate('2024-03-10T08:00:00Z')
      expect(result).toBeTruthy()
    })
  })

  describe('formatDateTime', () => {
    it('formats ISO datetime string', () => {
      const result = formatDateTime('2024-03-10T08:30:00Z')
      expect(result).toBeTruthy()
    })
  })

  describe('getRemainingTime', () => {
    it('returns 0 when time has elapsed', () => {
      const pastTime = new Date(Date.now() - 10000).toISOString()
      expect(getRemainingTime(pastTime, 5)).toBe(0)
    })

    it('returns remaining time for active session', () => {
      const now = new Date().toISOString()
      // This test might have slight timing variations
      const remaining = getRemainingTime(now, 1500)
      expect(remaining).toBeLessThanOrEqual(1500)
      expect(remaining).toBeGreaterThanOrEqual(1498)
    })
  })
})