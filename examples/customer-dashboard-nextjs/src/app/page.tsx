import { OpenMeterQuery } from './Query.client'

export default function Home() {
  return (
    <main className="flex h-screen space-x-12 p-24">
      <div className="rounded bg-white p-9 shadow-xl overflow-auto w-full h-full space-y-6">
        <h2 className="text-2xl font-bold text-sky-700 border-b border-slate-300 pb-1.5">
          Customer dashboard demo
        </h2>
        <div className="grid grid-cols-2 gap-y-1 gap-x-3 w-fit items-center">
          <div className="uppercase font-bold text-sm text-slate-800">
            Subject
          </div>
          <div>{process.env.NEXT_PUBLIC_OPENMETER_SUBJECT}</div>
          <div className="uppercase font-bold text-sm text-slate-800">
            Meter
          </div>
          <div>{process.env.NEXT_PUBLIC_OPENMETER_METER_SLUG}</div>
        </div>
        <OpenMeterQuery />
      </div>
    </main>
  )
}
