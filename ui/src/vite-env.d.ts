/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_TASK_DEBUG_API: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
