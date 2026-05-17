<script setup lang="ts">
import { ImageUp } from 'lucide-vue-next';
</script>

<template>
  <section class="boot-screen" aria-label="页面加载中">
    <div class="boot-loader">
      <div class="brand-mark auth-logo boot-logo"><ImageUp :size="32" /></div>
      <span class="boot-shadow"></span>
    </div>
  </section>
</template>
