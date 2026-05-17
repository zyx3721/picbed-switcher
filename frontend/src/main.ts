import { createApp } from 'vue';
import './style.css';
import App from './App.vue';

if (window.location.pathname !== '/') {
  window.history.replaceState(null, '', '/' + window.location.search + window.location.hash);
}

createApp(App).mount('#app');
