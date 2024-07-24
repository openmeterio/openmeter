export interface Paginated<T> {
  items: T[]
  totalCount: number
  page: number
  pageSize: number
}
