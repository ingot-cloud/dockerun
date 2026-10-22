import request from "./request";
export async function api<T = any>(
  method: string,
  path: string,
  data?: any,
  signal?: AbortSignal,
): Promise<T> {
  return request({ method, url: path, data, signal, silentError: true }) as unknown as Promise<T>;
}
export function errorText(e: any) {
  if (e?.code === "ECONNABORTED") return "请求超时，请重试";
  if (e?.code === "ERR_NETWORK") return "无法连接服务，请检查网络后重试";
  return e?.response?.data?.error || e?.message || "操作失败，请重试";
}
export async function download(path: string, name: string) {
  const b: any = await request.get(path, { responseType: "blob" });
  const url = URL.createObjectURL(b);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}
