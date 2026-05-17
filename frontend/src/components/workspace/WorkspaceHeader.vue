<script setup lang="ts">
import { ImageUp, LogOut, UserCog } from 'lucide-vue-next';
import { useWorkspaceContext } from '../../composables/useWorkspaceContext';

const { user, openProfileDialog, logout } = useWorkspaceContext();
</script>

<template>
  <header class="topbar">
    <div class="brand-line">
      <div class="brand-mark"><ImageUp :size="24" /></div>
      <div>
        <p class="eyebrow">PicBed Switcher</p>
        <h1>图床转站助手</h1>
      </div>
    </div>
    <div class="account">
      <span>{{ user?.username }}</span>
      <button class="ghost icon-button" type="button" @click="openProfileDialog">
        <UserCog :size="18" />修改信息
      </button>
      <button class="ghost icon-button" type="button" @click="logout">
        <LogOut :size="18" />退出
      </button>
    </div>
  </header>
</template>
