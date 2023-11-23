'use client'

import { createContext } from 'react'

export type OpenMeterConfig = {
  url: string
  token?: string
}

export const OpenMeterContext = createContext<OpenMeterConfig | null>(null)
