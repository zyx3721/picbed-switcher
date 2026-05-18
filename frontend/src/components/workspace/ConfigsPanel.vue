<script setup lang="ts">
import { ChevronDown, Eye, EyeOff, KeyRound, PlugZap, Plus, RefreshCw, Trash2 } from 'lucide-vue-next';
import { useWorkspaceContext } from '../../composables/useWorkspaceContext';

const {
  loading,
  configForm,
  configErrors,
  configs,
  configTypeDropdownOpen,
  supportedTypes,
  selectedFields,
  typeLabel,
  fieldLabel,
  fieldPlaceholder,
  secretFieldVisible,
  toggleSecretField,
  selectConfigType,
  handleConfigTypeChange,
  resetConfigForm,
  saveConfig,
  testConfig,
  editConfig,
  setDefault,
  requestDeleteConfig,
} = useWorkspaceContext();
</script>

<template>
<section class="grid two-cols">
  <form class="panel stack" @submit.prevent="saveConfig">
    <div class="section-title">
      <div>
        <p class="section-kicker">Credentials</p>
        <h2>{{ configForm.id ? '编辑图床配置' : '新增图床配置' }}</h2>
      </div>
      <span class="secure-badge"><KeyRound :size="16" /> 按类型校验</span>
    </div>
    <div class="field-row">
      <label class="select-field config-type-field">
        <span>图床类型</span>
        <div class="custom-select" :class="{ open: configTypeDropdownOpen }">
          <button
            class="select-trigger"
            type="button"
            :aria-expanded="configTypeDropdownOpen"
            @click="configTypeDropdownOpen = !configTypeDropdownOpen"
          >
            <span>{{ typeLabel(configForm.picbed_type) }}</span>
            <ChevronDown :size="18" class="select-chevron" />
          </button>
          <div v-if="configTypeDropdownOpen" class="select-menu">
            <button
              v-for="item in supportedTypes"
              :key="item.value"
              class="select-option"
              type="button"
              :class="{ selected: item.value === configForm.picbed_type }"
              @click="selectConfigType(item.value)"
            >
              <span>{{ typeLabel(item.value) }}</span>
              <small>{{ item.description }}</small>
            </button>
          </div>
        </div>
      </label>
      <label
        >图床类型<select v-model="configForm.picbed_type" @change="handleConfigTypeChange">
          <option v-for="item in supportedTypes" :key="item.value" :value="item.value">
            {{ typeLabel(item.value) }}
          </option>
        </select></label
      ><label :class="{ invalid: configErrors.config_name }"
        ><span class="field-label-text">配置名称</span
        ><input
          v-model.trim="configForm.config_name"
          :class="{ invalid: configErrors.config_name }"
          placeholder="生产图床配置"
          @input="configErrors.config_name = ''"
      /></label>
    </div>
    <div class="dynamic-fields">
      <label
        v-for="field in selectedFields"
        :key="field.key"              :class="{ required: field.required, invalid: configErrors.fields[field.key] }"
        ><span class="field-label-text">{{ fieldLabel(field) }}</span>
        <div class="secret-input-wrap" :class="{ invalid: configErrors.fields[field.key] }">
          <input
            v-model.trim="configForm.values[field.key]"
            :class="{ invalid: configErrors.fields[field.key] }"
            :type="field.secret && !secretFieldVisible(field.key) ? 'password' : 'text'"
            :placeholder="fieldPlaceholder(field)"
            autocomplete="off"
            @input="delete configErrors.fields[field.key]"
          />
          <button
            v-if="field.secret"
            class="secret-toggle"
            type="button"
            :aria-label="secretFieldVisible(field.key) ? '隐藏密钥' : '显示密钥'"
            @click="toggleSecretField(field.key)"
          >
            <EyeOff v-if="secretFieldVisible(field.key)" :size="18" />
            <Eye v-else :size="18" />
          </button>
        </div></label>
    </div>
    <label class="checkbox"
      ><input v-model="configForm.is_default" type="checkbox" />设为默认配置</label
    >
    <div class="actions">
      <button class="secondary" type="button" @click="resetConfigForm">
        <RefreshCw :size="18" />清空</button
      ><button class="secondary" type="button" :disabled="loading" @click="testConfig()">
        <PlugZap :size="18" />测试配置</button
      ><button class="primary" :disabled="loading" type="submit">
        <Plus :size="18" />保存配置
      </button>
    </div>
  </form>
  <div class="panel stack">
    <div class="section-title">
      <div>
        <p class="section-kicker">Saved</p>
        <h2>已保存配置</h2>
      </div>
      <span class="count-badge">{{ configs.length }}</span>
    </div>
    <div class="config-list">
      <div
        v-for="item in configs"
        :key="item.id"
        class="config-row"
        :data-provider="item.picbed_type"
      >
        <div>
          <strong>{{ item.config_name }}</strong
          ><span
            >{{ typeLabel(item.picbed_type) }}{{ item.is_default ? ' · 默认' : '' }}</span
          >
        </div>
        <div class="row-actions">
          <button class="ghost" type="button" @click="editConfig(item)">编辑</button
          ><button class="ghost" type="button" :disabled="loading" @click="testConfig(item)">测试</button
          ><button class="ghost" type="button" @click="setDefault(item.id)">默认</button
          ><button
            class="danger icon-only"
            type="button"
            aria-label="删除配置"
            @click="requestDeleteConfig(item)"
          >
            <Trash2 :size="17" />
          </button>
        </div>
      </div>
      <p v-if="configs.length === 0" class="empty">暂无图床配置</p>
    </div>
  </div>
</section>
</template>
