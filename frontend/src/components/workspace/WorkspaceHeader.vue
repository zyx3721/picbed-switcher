<script setup lang="ts">
import { ImageUp, KeyRound, LogOut } from 'lucide-vue-next';
import { useWorkspaceContext } from '../../composables/useWorkspaceContext';

const { user, openPasswordDialog, logout } = useWorkspaceContext();
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
      <button class="ghost icon-button" type="button" @click="openPasswordDialog">
        <KeyRound :size="18" />修改密码
      </button>
      <button class="ghost icon-button" type="button" @click="logout">
        <LogOut :size="18" />退出
      </button>
    </div>
  </header>
</template>
