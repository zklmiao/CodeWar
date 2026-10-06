import { createApp } from "vue";
import { createPinia } from "pinia";
// 06 号票：设计 token 单一来源（13 号票占位规范）——组件一律 var(--cw-*)
import "../../design/tokens.css";
import App from "./App.vue";
import { router } from "./router";

createApp(App).use(createPinia()).use(router).mount("#app");
