import { onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import { api, errorText } from '@/api/workspace'

// 列表和详情共用加载、取消与有界轮询，刷新失败时保留上次数据。
export function useResource<T>(path: string | (() => string), initial: T, polling?: { interval: number; active: (data:T)=>boolean }) {
  const data=shallowRef<T>(initial), loading=ref(false), error=ref(''), loaded=ref(false)
  const controller=new AbortController()
  let disposed=false, fetching=false, timer: ReturnType<typeof setTimeout> | undefined
  async function load(background=false) {
    if(disposed || fetching) return
    clearTimeout(timer); fetching=true
    if(!background || !loaded.value) loading.value=true
    error.value=''
    try { const result=await api<T>('GET',typeof path==='function'?path():path,undefined,controller.signal); if(!disposed){data.value=result ?? initial;loaded.value=true} }
    catch(cause){if(!disposed) error.value=errorText(cause)}
    finally {fetching=false;if(!disposed){loading.value=false;if(polling && (error.value || polling.active(data.value))) timer=setTimeout(()=>void load(true),polling.interval)}}
  }
  onMounted(()=>void load())
  onBeforeUnmount(()=>{disposed=true;clearTimeout(timer);controller.abort()})
  return {data,loading,error,loaded,load}
}
