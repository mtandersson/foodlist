import { describe, it, expect } from 'vitest'
import { categoryColorSlot, categoryColorVar, CATEGORY_COLOR_SLOTS } from './categoryColor'

describe('categoryColorSlot', () => {
  it('gives the design colours to the common grocery categories', () => {
    expect(categoryColorSlot('Frukt & grönt')).toBe(0)
    expect(categoryColorSlot('Mejeri')).toBe(1)
    expect(categoryColorSlot('Kött')).toBe(5)
  })

  it('ignores case, spacing and emoji in the name', () => {
    expect(categoryColorSlot('  mejeri ')).toBe(categoryColorSlot('Mejeri'))
    expect(categoryColorSlot('Frukt & Grönt 🍅🥬')).toBe(0)
    expect(categoryColorSlot('Asiatiskt & Taco🌮')).toBe(4)
  })

  it('gives any other category the same valid slot every time', () => {
    const slot = categoryColorSlot('Bageri')
    expect(slot).toBeGreaterThanOrEqual(0)
    expect(slot).toBeLessThan(CATEGORY_COLOR_SLOTS)
    expect(categoryColorSlot('Bageri')).toBe(slot)
  })

  it('resolves to a themed CSS token', () => {
    expect(categoryColorVar('Mejeri')).toBe('var(--cat-1)')
  })
})
