import { describe, it, expect } from 'vitest';
import { smartSplit, extractVoiceInput } from './voiceInputParser';

describe('extractVoiceInput', () => {
  it('should extract text after "handla" prefix', () => {
    expect(extractVoiceInput('handla mjölk sirap potatis')).toBe('mjölk sirap potatis');
    expect(extractVoiceInput('handla mjölk')).toBe('mjölk');
  });

  it('should extract text after "köp" prefix', () => {
    expect(extractVoiceInput('köp mjölk sirap')).toBe('mjölk sirap');
  });

  it('should extract text after "lägg till" prefix', () => {
    expect(extractVoiceInput('lägg till mjölk')).toBe('mjölk');
  });

  it('should extract text after "add" prefix', () => {
    expect(extractVoiceInput('add milk bread')).toBe('milk bread');
  });

  it('should extract text after "buy" prefix', () => {
    expect(extractVoiceInput('buy milk')).toBe('milk');
  });

  it('should handle case-insensitive prefixes', () => {
    expect(extractVoiceInput('HANDLA mjölk')).toBe('mjölk');
    expect(extractVoiceInput('Köp mjölk')).toBe('mjölk');
  });

  it('should return text as-is if no prefix matches', () => {
    expect(extractVoiceInput('mjölk sirap potatis')).toBe('mjölk sirap potatis');
    expect(extractVoiceInput('just some text')).toBe('just some text');
  });

  it('should handle empty input', () => {
    expect(extractVoiceInput('')).toBe('');
  });

  it('should trim whitespace', () => {
    expect(extractVoiceInput('  handla mjölk  ')).toBe('mjölk');
    expect(extractVoiceInput('handla   mjölk   sirap')).toBe('mjölk   sirap');
  });
});

describe('smartSplit', () => {
  describe('basic splitting without known items', () => {
    it('should split by spaces when no known items', () => {
      expect(smartSplit('mjölk sirap potatis', [])).toEqual(['mjölk', 'sirap', 'potatis']);
    });

    it('should handle single word', () => {
      expect(smartSplit('mjölk', [])).toEqual(['mjölk']);
    });

    it('should handle multiple spaces', () => {
      expect(smartSplit('mjölk  sirap   potatis', [])).toEqual(['mjölk', 'sirap', 'potatis']);
    });

    it('should trim whitespace', () => {
      expect(smartSplit('  mjölk sirap  ', [])).toEqual(['mjölk', 'sirap']);
    });
  });

  describe('matching known items', () => {
    it('should match single-word known items', () => {
      const knownItems = ['mjölk', 'sirap', 'potatis'];
      expect(smartSplit('mjölk sirap potatis', knownItems)).toEqual(['mjölk', 'sirap', 'potatis']);
    });

    it('should match multi-word known items', () => {
      const knownItems = ['mjölk sirap', 'potatis'];
      expect(smartSplit('mjölk sirap potatis', knownItems)).toEqual(['mjölk sirap', 'potatis']);
    });

    it('should prioritize longer matches', () => {
      const knownItems = ['mjölk', 'mjölk sirap', 'potatis'];
      expect(smartSplit('mjölk sirap potatis', knownItems)).toEqual(['mjölk sirap', 'potatis']);
    });

    it('should handle case-insensitive matching', () => {
      const knownItems = ['Mjölk', 'Sirap'];
      expect(smartSplit('mjölk sirap', knownItems)).toEqual(['mjölk', 'sirap']);
    });

    it('should preserve original case from input', () => {
      const knownItems = ['mjölk'];
      expect(smartSplit('Mjölk', knownItems)).toEqual(['Mjölk']);
    });

    it('should match known items mixed with unknown words', () => {
      const knownItems = ['mjölk', 'potatis'];
      expect(smartSplit('mjölk sirap potatis', knownItems)).toEqual(['mjölk', 'sirap', 'potatis']);
    });

    it('should handle multiple multi-word items', () => {
      const knownItems = ['mjölk sirap', 'potatis kål'];
      expect(smartSplit('mjölk sirap potatis kål', knownItems)).toEqual(['mjölk sirap', 'potatis kål']);
    });
  });

  describe('edge cases', () => {
    it('should handle empty input', () => {
      expect(smartSplit('', [])).toEqual([]);
      expect(smartSplit('', ['mjölk'])).toEqual([]);
    });

    it('should handle empty known items array', () => {
      expect(smartSplit('mjölk sirap', [])).toEqual(['mjölk', 'sirap']);
    });

    it('should filter out empty strings', () => {
      expect(smartSplit('mjölk   sirap', [])).toEqual(['mjölk', 'sirap']);
    });

    it('should handle items that are substrings of other items', () => {
      const knownItems = ['mjölk', 'mjölk sirap', 'sirap'];
      expect(smartSplit('mjölk sirap potatis', knownItems)).toEqual(['mjölk sirap', 'potatis']);
    });

    it('should handle partial matches correctly', () => {
      const knownItems = ['mjölk sirap'];
      // "mjölk sirap potatis" should match "mjölk sirap" and leave "potatis"
      expect(smartSplit('mjölk sirap potatis', knownItems)).toEqual(['mjölk sirap', 'potatis']);
    });

    it('should not match items that are part of a larger word', () => {
      const knownItems = ['mjölk'];
      // "mjölks" should not match "mjölk" because it's not a complete word
      expect(smartSplit('mjölks sirap', knownItems)).toEqual(['mjölks', 'sirap']);
    });

    it('should handle items with special characters', () => {
      const knownItems = ['mjölk (1L)', 'sirap'];
      expect(smartSplit('mjölk (1L) sirap', knownItems)).toEqual(['mjölk (1L)', 'sirap']);
    });
  });

  describe('real-world scenarios', () => {
    it('should handle Swedish shopping list items', () => {
      const knownItems = ['mjölk', 'mjölk sirap', 'potatis', 'köttfärs'];
      const input = 'mjölk sirap potatis köttfärs';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk sirap', 'potatis', 'köttfärs']);
    });

    it('should handle mixed known and unknown items', () => {
      const knownItems = ['mjölk', 'potatis'];
      const input = 'mjölk okänd vara potatis';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk', 'okänd', 'vara', 'potatis']);
    });

    it('should handle items that appear multiple times', () => {
      const knownItems = ['mjölk'];
      const input = 'mjölk sirap mjölk';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk', 'sirap', 'mjölk']);
    });

    it('should prioritize longer matches even if shorter ones come first in array', () => {
      const knownItems = ['potatis', 'mjölk', 'mjölk sirap'];
      const input = 'mjölk sirap potatis';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk sirap', 'potatis']);
    });
  });

  describe('emoji handling', () => {
    it('should match items with emojis', () => {
      const knownItems = ['🍞 Bröd', 'mjölk', '🥛 Mjölk'];
      const input = '🍞 Bröd mjölk 🥛 Mjölk';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 Bröd', 'mjölk', '🥛 Mjölk']);
    });

    it('should match items with emojis case-insensitively', () => {
      const knownItems = ['🍞 Bröd', '🥛 Mjölk'];
      const input = '🍞 bröd 🥛 mjölk';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 bröd', '🥛 mjölk']);
    });

    it('should match multi-word items with emojis', () => {
      const knownItems = ['🍞 Bröd', '🥛 Mjölk sirap', 'potatis'];
      const input = '🍞 Bröd 🥛 Mjölk sirap potatis';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 Bröd', '🥛 Mjölk sirap', 'potatis']);
    });

    it('should handle emojis in the middle of items', () => {
      const knownItems = ['Mjölk 🥛', 'Bröd 🍞'];
      const input = 'Mjölk 🥛 Bröd 🍞';
      expect(smartSplit(input, knownItems)).toEqual(['Mjölk 🥛', 'Bröd 🍞']);
    });

    it('should match items with multiple emojis', () => {
      const knownItems = ['🍞🥖 Bröd', '🥛🥛 Mjölk'];
      const input = '🍞🥖 Bröd 🥛🥛 Mjölk';
      expect(smartSplit(input, knownItems)).toEqual(['🍞🥖 Bröd', '🥛🥛 Mjölk']);
    });

    it('should prioritize longer matches with emojis', () => {
      const knownItems = ['🍞 Bröd', '🍞 Bröd med smör'];
      const input = '🍞 Bröd med smör';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 Bröd med smör']);
    });

    it('should handle emojis at the start of input', () => {
      const knownItems = ['🍞 Bröd'];
      const input = '🍞 Bröd mjölk';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 Bröd', 'mjölk']);
    });

    it('should handle emojis-only items', () => {
      const knownItems = ['🍞', '🥛'];
      const input = '🍞 🥛 potatis';
      expect(smartSplit(input, knownItems)).toEqual(['🍞', '🥛', 'potatis']);
    });

    it('should match items with emojis when input has no emojis', () => {
      const knownItems = ['🍞 Bröd', '🥛 Mjölk'];
      const input = 'Bröd Mjölk';
      expect(smartSplit(input, knownItems)).toEqual(['Bröd', 'Mjölk']);
    });

    it('should handle mixed emoji and non-emoji items', () => {
      const knownItems = ['🍞 Bröd', 'mjölk', 'potatis'];
      const input = '🍞 Bröd mjölk potatis';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 Bröd', 'mjölk', 'potatis']);
    });
  });

  describe('mixed case handling', () => {
    it('should match items with mixed case', () => {
      const knownItems = ['Mjölk', 'Sirap', 'Potatis'];
      const input = 'mjölk sirap potatis';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk', 'sirap', 'potatis']);
    });

    it('should preserve original case from input', () => {
      const knownItems = ['mjölk'];
      const input = 'Mjölk';
      expect(smartSplit(input, knownItems)).toEqual(['Mjölk']);
    });

    it('should match uppercase known items with lowercase input', () => {
      const knownItems = ['MJÖLK', 'SIRAP'];
      const input = 'mjölk sirap';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk', 'sirap']);
    });

    it('should match lowercase known items with uppercase input', () => {
      const knownItems = ['mjölk', 'sirap'];
      const input = 'MJÖLK SIRAP';
      expect(smartSplit(input, knownItems)).toEqual(['MJÖLK', 'SIRAP']);
    });

    it('should handle Title Case items', () => {
      const knownItems = ['Mjölk Sirap', 'Potatis'];
      const input = 'mjölk sirap potatis';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk sirap', 'potatis']);
    });

    it('should handle camelCase items', () => {
      const knownItems = ['MjölkSirap', 'Potatis'];
      const input = 'mjölksirap potatis';
      // Note: camelCase won't match because of space requirement, but should still work for single word
      expect(smartSplit(input, knownItems)).toEqual(['mjölksirap', 'potatis']);
    });

    it('should match mixed case multi-word items', () => {
      const knownItems = ['Mjölk Sirap', 'Potatis Kål'];
      const input = 'mjölk sirap potatis kål';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk sirap', 'potatis kål']);
    });

    it('should handle items with inconsistent casing', () => {
      const knownItems = ['mJöLk', 'sIrAp'];
      const input = 'Mjölk Sirap';
      expect(smartSplit(input, knownItems)).toEqual(['Mjölk', 'Sirap']);
    });

    it('should prioritize longer matches regardless of case', () => {
      const knownItems = ['Mjölk', 'Mjölk Sirap'];
      const input = 'mjölk sirap potatis';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk sirap', 'potatis']);
    });

    it('should handle items with accented characters and mixed case', () => {
      const knownItems = ['Mjölk', 'Köttfärs', 'Potatis'];
      const input = 'mjölk köttfärs potatis';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk', 'köttfärs', 'potatis']);
    });

    it('should match items with all uppercase when input is lowercase', () => {
      const knownItems = ['MJÖLK SIRAP', 'POTATIS'];
      const input = 'mjölk sirap potatis';
      expect(smartSplit(input, knownItems)).toEqual(['mjölk sirap', 'potatis']);
    });
  });

  describe('combined emoji and mixed case', () => {
    it('should handle emojis with mixed case items', () => {
      const knownItems = ['🍞 Bröd', '🥛 Mjölk', 'Potatis'];
      const input = '🍞 bröd 🥛 mjölk potatis';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 bröd', '🥛 mjölk', 'potatis']);
    });

    it('should handle multi-word items with emojis and mixed case', () => {
      const knownItems = ['🍞 Bröd med Smör', '🥛 Mjölk Sirap'];
      const input = '🍞 bröd med smör 🥛 mjölk sirap';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 bröd med smör', '🥛 mjölk sirap']);
    });

    it('should preserve case from input when matching emoji items', () => {
      const knownItems = ['🍞 Bröd'];
      const input = '🍞 BRÖD';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 BRÖD']);
    });

    it('should handle emojis with all uppercase known items', () => {
      const knownItems = ['🍞 BRÖD', '🥛 MJÖLK'];
      const input = '🍞 bröd 🥛 mjölk';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 bröd', '🥛 mjölk']);
    });

    it('should prioritize longer matches with emojis and mixed case', () => {
      const knownItems = ['🍞 Bröd', '🍞 Bröd med Smör'];
      const input = '🍞 bröd med smör';
      expect(smartSplit(input, knownItems)).toEqual(['🍞 bröd med smör']);
    });
  });
});


