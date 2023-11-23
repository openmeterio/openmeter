'use client'

import {
  QueryClient,
  QueryClientProvider,
  useQuery,
} from '@tanstack/react-query'
import { experimental_createPersister } from '@tanstack/query-persist-client-core'
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import type { MeterQueryRow, WindowSize } from '@openmeter/web'
import { OpenMeterProvider, useOpenMeter } from '@openmeter/web/react'
import {
  Chart as ChartJS,
  Colors,
  BarElement,
  TimeScale,
  TimeSeriesScale,
  LinearScale,
  Tooltip,
  PointElement,
} from 'chart.js'
import 'chartjs-adapter-luxon'
import { Bar } from 'react-chartjs-2'
import { useMemo } from 'react'

ChartJS.register(
  Colors,
  BarElement,
  LinearScale,
  TimeScale,
  TimeSeriesScale,
  Tooltip,
  PointElement
)

const queryClient = new QueryClient()

export function OpenMeterQuery() {
  return (
    <QueryClientProvider client={queryClient}>
      <OpenMeterPortal>
        <div className="flex flex-col space-y-12">
          <div className="flex-1">
            <OpenMeterQueryChart
              meterSlug={process.env.NEXT_PUBLIC_OPENMETER_METER_SLUG}
              windowSize="DAY"
            />
          </div>
          <OpenMeterQueryTable
            meterSlug={process.env.NEXT_PUBLIC_OPENMETER_METER_SLUG}
            windowSize="DAY"
          />
        </div>
      </OpenMeterPortal>
    </QueryClientProvider>
  )
}

export function OpenMeterPortal({ children }: { children?: React.ReactNode }) {
  // get portal token from server
  const { data } = useQuery<{ token: string }>({
    queryKey: ['openmeter', 'token'],
    queryFn: async () => fetch('/api/token').then((res) => res.json()),
    // enable token caching (optional)
    staleTime: 12 * 60 * 60 * 1000,
    refetchInterval: 12 * 60 * 60 * 1000,
    // only cache token in browser
    persister:
      typeof window !== 'undefined'
        ? experimental_createPersister({
            prefix: 'openmeter',
            storage: window.localStorage,
            maxAge: 30 * 60 * 1000, // 30 minutes in ms
          })
        : undefined,
  })

  return (
    <OpenMeterProvider
      url={process.env.NEXT_PUBLIC_OPENMETER_URL}
      token={data?.token}
    >
      {children}
    </OpenMeterProvider>
  )
}

type Params = {
  meterSlug: string
  from?: string
  to?: string
  windowSize?: WindowSize
  windowTimeZone?: string
}

function useOpenMeterQuery(params: Params) {
  const openmeter = useOpenMeter()
  return useQuery({
    queryKey: ['openmeter', 'queryPortalMeter', params],
    queryFn: async () => {
      const { data } = await openmeter!.queryPortalMeter(params)
      return data
    },
    // disable query when openmeter client is not initialized (token is missing)
    enabled: !!openmeter,
  })
}

const columnHelper = createColumnHelper<MeterQueryRow>()
const columns = [
  columnHelper.accessor('windowStart', {
    id: 'windowStart',
    header: 'Window Start',
    cell: ({ getValue }) => (
      <time
        className="whitespace-nowrap text-gray-600 underline outline-none"
        dateTime={getValue()}
      >
        {getValue()}
      </time>
    ),
  }),
  columnHelper.accessor('windowEnd', {
    id: 'windowEnd',
    header: 'Window End',
    cell: ({ getValue }) => (
      <time
        className="whitespace-nowrap text-gray-600 underline outline-none"
        dateTime={getValue()}
      >
        {getValue()}
      </time>
    ),
  }),
  columnHelper.accessor('value', {
    id: 'value',
    header: 'Value',
    cell: ({ getValue }) =>
      getValue().toLocaleString(undefined, {
        maximumFractionDigits: 2,
      }),
  }),
]

export function OpenMeterQueryTable(params: Params) {
  // NOTE: error and loading states aren't handled here for brevity
  const { data } = useOpenMeterQuery(params)
  const table = useReactTable({
    columns,
    data: data?.data ?? [],
    getCoreRowModel: getCoreRowModel(),
  })

  return (
    <table className="w-full caption-bottom border-separate border-spacing-0 bg-slate-100">
      <thead>
        {table.getHeaderGroups().map((headerGroup) => (
          <tr key={headerGroup.id}>
            {headerGroup.headers.map((header) => (
              <th
                key={header.id}
                className="h-10 whitespace-nowrap bg-slate-200 px-2 text-left align-middle font-medium uppercase text-sm"
              >
                {flexRender(
                  header.column.columnDef.header,
                  header.getContext()
                )}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody>
        {table.getRowModel().rows.map((row) => (
          <tr key={row.id}>
            {row.getVisibleCells().map((cell) => (
              <td
                key={cell.id}
                className="whitespace-nowrap border-b border-zinc-200 p-2 align-middle"
              >
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  )
}

export function OpenMeterQueryChart(params: Params) {
  // NOTE: error and loading states aren't handled here for brevity
  const { data } = useOpenMeterQuery(params)
  const chartData = useMemo(
    () => ({
      label: 'Values',
      datasets: [
        {
          data:
            data?.data.map(({ windowStart, value }) => ({
              x: windowStart,
              y: value,
            })) ?? [],
        },
      ],
    }),
    [data]
  )

  return (
    <div className="bg-slate-100 rounded h-96">
      <Bar
        options={{
          responsive: true,
          maintainAspectRatio: false,
          interaction: {
            mode: 'nearest',
            axis: 'x',
            intersect: false,
          },
          scales: {
            x: {
              type: 'time',
              distribution: 'series',
              time: {
                unit: 'day',
              },
              adapters: {
                date: {
                  zone: 'UTC',
                },
              },
            },
            y: {
              min: 0,
            },
          },
        }}
        data={chartData}
      />
    </div>
  )
}
