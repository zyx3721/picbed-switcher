<script setup lang="ts">
import { ImageUp, LogOut, MailCheck, UserCog } from 'lucide-vue-next';
import { useWorkspaceContext } from '../../composables/useWorkspaceContext';

const { user, loading, openProfileDialog, logout, resendEmailVerification } = useWorkspaceContext();
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
      <button v-if="user && !user.email_verified" class="ghost icon-button" type="button" :disabled="loading" @click="resendEmailVerification">
        <MailCheck :size="18" />验证邮箱
      </button>
      <button class="ghost icon-button" type="button" @click="openProfileDialog">
        <UserCog :size="18" />修改信息
      </button>
      <button class="ghost icon-button" type="button" @click="logout">
        <LogOut :size="18" />退出
      </button>
    </div>
  </header>
</template>
