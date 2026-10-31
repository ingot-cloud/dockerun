// 刷新时仅保留当前页仍存在且可操作的记录，并采用最新的行数据。
export function retainPageSelection<T>(selected: readonly T[], page: readonly T[], key: (item: T) => unknown, selectable: (item: T) => boolean = () => true): T[] {
  const selectedKeys = new Set(selected.map(key)), retained = new Set<unknown>()
  return page.filter(item => {
    const id = key(item)
    if (id == null || !selectedKeys.has(id) || retained.has(id) || !selectable(item)) return false
    retained.add(id)
    return true
  })
}
