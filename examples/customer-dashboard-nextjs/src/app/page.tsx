import { OpenMeterQuery } from './Query.client';

export default function Home() {
  return (
    <main className="flex h-screen space-x-12 p-24">
      <div className="size-full space-y-6 overflow-auto rounded bg-white p-9 shadow-xl">
        <h2 className="border-b border-slate-300 pb-1.5 text-2xl font-bold text-sky-700">
          Customer dashboard demo
        </h2>
        <div className="grid w-fit grid-cols-2 items-center gap-x-3 gap-y-1">
          <div className="text-sm font-bold uppercase text-slate-800">
            Subject
          </div>
          <div>{process.env.NEXT_PUBLIC_OPENMETER_SUBJECT}</div>
          <div className="text-sm font-bold uppercase text-slate-800">
            Meter
          </div>
          <div>{process.env.NEXT_PUBLIC_OPENMETER_METER_SLUG}</div>
        </div>
        <OpenMeterQuery />
      </div>
    </main>
  );
}
