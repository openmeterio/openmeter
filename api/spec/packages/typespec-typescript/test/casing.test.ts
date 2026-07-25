import { describe, expect, it } from 'vitest'
import { isCasingDerivable, toCamelCase, toSnakeCase } from '../src/casing.js'

describe('toCamelCase', () => {
  it('camelizes single and multi-word snake names', () => {
    expect(toCamelCase('id')).toBe('id')
    expect(toCamelCase('meter_id')).toBe('meterId')
    expect(toCamelCase('amount_before_proration')).toBe('amountBeforeProration')
  })

  it('handles digits after an underscore', () => {
    expect(toCamelCase('cache_read_per_token')).toBe('cacheReadPerToken')
    expect(toCamelCase('int_64')).toBe('int64')
  })

  it('leaves already-camel names unchanged', () => {
    expect(toCamelCase('createdAt')).toBe('createdAt')
  })
})

describe('toSnakeCase', () => {
  it('snakeifies camel names', () => {
    expect(toSnakeCase('meterId')).toBe('meter_id')
    expect(toSnakeCase('amountBeforeProration')).toBe('amount_before_proration')
  })

  it('leaves single-word names unchanged', () => {
    expect(toSnakeCase('id')).toBe('id')
    expect(toSnakeCase('eq')).toBe('eq')
  })
})

describe('round-trip', () => {
  const snakeNames = [
    'id',
    'meter_id',
    'amount_before_proration',
    'has_access',
    'cache_read_per_token',
    'created_at',
    'include_deleted',
  ]

  it('snake → camel → snake is identity for derivable names', () => {
    for (const name of snakeNames) {
      expect(toSnakeCase(toCamelCase(name))).toBe(name)
      expect(isCasingDerivable(name)).toBe(true)
    }
  })

  it('flags names that do not round-trip', () => {
    expect(isCasingDerivable('weird_APIKey')).toBe(false)
    expect(isCasingDerivable('alreadyCamel')).toBe(false)
  })
})
