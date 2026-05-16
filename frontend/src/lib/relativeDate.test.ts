import { describe, it, expect } from 'vitest';
import { formatShortRelative } from './relativeDate';

describe('formatShortRelative', () => {
  const NOW = new Date('2026-05-16T12:00:00Z');

  it('returns "nu" for less than a minute ago', () => {
    expect(formatShortRelative('2026-05-16T11:59:31Z', NOW)).toBe('nu');
    expect(formatShortRelative('2026-05-16T12:00:00Z', NOW)).toBe('nu');
  });

  it('returns minutes for less than an hour', () => {
    expect(formatShortRelative('2026-05-16T11:55:00Z', NOW)).toBe('5m');
    expect(formatShortRelative('2026-05-16T11:01:00Z', NOW)).toBe('59m');
  });

  it('returns hours for less than a day', () => {
    expect(formatShortRelative('2026-05-16T09:00:00Z', NOW)).toBe('3h');
    expect(formatShortRelative('2026-05-15T13:00:00Z', NOW)).toBe('23h');
  });

  it('returns days for less than 14 days', () => {
    expect(formatShortRelative('2026-05-15T12:00:00Z', NOW)).toBe('1d');
    expect(formatShortRelative('2026-05-03T12:00:00Z', NOW)).toBe('13d');
  });

  it('returns weeks for less than 8 weeks', () => {
    expect(formatShortRelative('2026-05-02T12:00:00Z', NOW)).toBe('2v');
    expect(formatShortRelative('2026-03-22T12:00:00Z', NOW)).toBe('7v');
  });

  it('returns months for less than a year', () => {
    expect(formatShortRelative('2026-02-15T12:00:00Z', NOW)).toBe('3mån');
    expect(formatShortRelative('2025-06-01T12:00:00Z', NOW)).toBe('11mån');
  });

  it('returns years for at least a year', () => {
    expect(formatShortRelative('2025-05-16T12:00:00Z', NOW)).toBe('1år');
    expect(formatShortRelative('2024-05-16T12:00:00Z', NOW)).toBe('2år');
  });

  it('clamps future timestamps to "nu"', () => {
    expect(formatShortRelative('2026-05-17T00:00:00Z', NOW)).toBe('nu');
  });

  it('returns empty string for invalid input', () => {
    expect(formatShortRelative('not-a-date', NOW)).toBe('');
  });
});
