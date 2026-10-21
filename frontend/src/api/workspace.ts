import request from "./request";
export async function api<T = any>(
  method: string,
  path: string,
  data?: any,
): Promise<T> {
  return request({ method, url: path, data }) as unknown as Promise<T>;
}
export function errorText(e: any) {
  return e.response?.data?.error || e.message || "操作失败";
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
