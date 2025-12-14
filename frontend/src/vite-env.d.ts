/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  readonly VITE_AES_KEY: string
  readonly VITE_AES_IV: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

