// SFC 模块声明：补齐 tsc --noEmit 对 .vue 导入的类型解析
//（本仓库 CI 门控为 vitest + vite build，此文件只影响编辑器与手动 tsc）。
declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<object, object, unknown>;
  export default component;
}
