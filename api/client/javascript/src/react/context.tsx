'use client'

import { createContext, useContext } from 'react'
import type { OpenMeter } from '../portal/index.js'

export * from '../portal/index.js'

export const OpenMeterContext = createContext<OpenMeter | null>(null)

export type OpenMeterProviderProps = {
  children?: React.ReactNode
  value: OpenMeter | null
}

export function OpenMeterProvider({ children, value }: OpenMeterProviderProps) {
  return (
    <OpenMeterContext.Provider value={value}>
      {children}
    </OpenMeterContext.Provider>
  )
}

export function useOpenMeter() {
  const context = useContext(OpenMeterContext)
  if (typeof context === 'undefined') {
    throw new Error('useOpenMeter must be used within a OpenMeterProvider')
  }

  return context
}
