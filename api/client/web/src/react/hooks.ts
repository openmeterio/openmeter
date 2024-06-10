'use client'

import { useContext, useMemo } from 'react'
import { OpenMeterClient } from '../client/index.js'
import { OpenMeterContext } from './context.js'

export function useOpenMeter() {
  const context = useContext(OpenMeterContext)

  if (!context) {
    throw new Error('useOpenMeter must be used within an OpenMeterProvider')
  }

  if (!context.url) {
    throw new Error('OpenMeterProvider must be initialized with a url')
  }

  return useMemo(
    () =>
      context.token ? new OpenMeterClient(context.url, context.token) : null,
    [context.url, context.token]
  )
}
