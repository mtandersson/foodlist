import { describe, it, expect } from 'vitest';
import { extractAutocompleteQuery } from './quantityParser';

describe('extractAutocompleteQuery', () => {
  it('strips leading quantity and unit', () => {
    expect(extractAutocompleteQuery('2l Mj')).toBe('Mj');
    expect(extractAutocompleteQuery('2l mjölk')).toBe('mjölk');
    expect(extractAutocompleteQuery('1 dl mjölk')).toBe('mjölk');
  });

  it('strips modifier + quantity + unit', () => {
    expect(extractAutocompleteQuery('ca 2 dl mj')).toBe('mj');
    expect(extractAutocompleteQuery('lite mjöl')).toBe('lite mjöl');
  });

  it('strips number + packaging unit', () => {
    expect(extractAutocompleteQuery('1 burk tomater')).toBe('tomater');
    expect(extractAutocompleteQuery('2 burkar tomater')).toBe('tomater');
  });

  it('returns full input when no quantity/unit prefix', () => {
    expect(extractAutocompleteQuery('mjölk')).toBe('mjölk');
    expect(extractAutocompleteQuery('Mj')).toBe('Mj');
  });

  it('handles empty input', () => {
    expect(extractAutocompleteQuery('')).toBe('');
    expect(extractAutocompleteQuery('   ')).toBe('   ');
  });

  it('handles decimal numbers', () => {
    expect(extractAutocompleteQuery('1,5 dl mjölk')).toBe('mjölk');
    expect(extractAutocompleteQuery('2.5 kg potatis')).toBe('potatis');
  });
});
