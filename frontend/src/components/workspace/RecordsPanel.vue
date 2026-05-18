<script setup lang="ts">
import { ref } from 'vue';
import { Download, FileText, RefreshCw, X } from 'lucide-vue-next';
import { useWorkspaceContext } from '../../composables/useWorkspaceContext';

const { records, recordDetail, recordDetailOpen, typeLabel, loadRecords, openRecordDetail, closeRecordDetail } = useWorkspaceContext();

function downloadRecordContent() {
  if (!recordDetail.value?.converted_content) return;
  const blob = new Blob([recordDetail.value.converted_content], { type: 'text/markdown;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = recordDetail.value.original_filename || 'converted.md';
  link.click();
  URL.revokeObjectURL(url);
}

const urlTooltip = ref({
  visible: false,
  text: '',
  x: 0,
  y: 0,
});

function updateUrlTooltipPosition(event: MouseEvent | FocusEvent) {
  const maxWidth = Math.min(560, Math.max(260, window.innerWidth - 32));
  const maxHeight = 220;
  let x = 16;
  let y = 16;

  if (event instanceof MouseEvent) {
    x = event.clientX + 14;
    y = event.clientY + 16;
    if (y + maxHeight > window.innerHeight - 16) y = event.clientY - maxHeight - 16;
  } else if (event.currentTarget instanceof HTMLElement) {
    const rect = event.currentTarget.getBoundingClientRect();
    x = rect.left;
    y = rect.bottom + 8;
    if (y + maxHeight > window.innerHeight - 16) y = rect.top - maxHeight - 8;
  }

  urlTooltip.value.x = Math.max(16, Math.min(x, window.innerWidth - maxWidth - 16));
  urlTooltip.value.y = Math.max(16, Math.min(y, window.innerHeight - 80));
}

function showUrlTooltip(text: string, event: MouseEvent | FocusEvent) {
  if (!text) return;
  urlTooltip.value.visible = true;
  urlTooltip.value.text = text;
  updateUrlTooltipPosition(event);
}

function hideUrlTooltip() {
  urlTooltip.value.visible = false;
}
</script>

<template>
<section class="panel stack">
  <div class="section-title">
    <div>
      <p class="section-kicker">Timeline</p>
      <h2>转换历史</h2>
    </div>
    <button class="secondary" type="button" @click="loadRecords">
      <RefreshCw :size="18" />刷新
    </button>
  </div>
  <div class="table-wrap records-table-wrap">
    <table>
      <thead>
        <tr>
          <th>文件</th>
          <th>源图床</th>
          <th>目标图床</th>
          <th>状态</th>
          <th>图片数</th>
          <th>时间</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="record in records" :key="record.id" class="clickable-row" @click="openRecordDetail(record)">
          <td>{{ record.original_filename }}</td>
          <td>{{ typeLabel(record.source_picbed) }}</td>
          <td>{{ typeLabel(record.target_picbed) }}</td>
          <td>
            <span :class="['status', record.status]">{{
              record.status === 'success' ? '成功' : '失败'
            }}</span>
          </td>
          <td>{{ record.image_count }}</td>
          <td>{{ new Date(record.created_at).toLocaleString() }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <p v-if="records.length === 0" class="empty">暂无转换记录</p>
</section>

<div v-if="recordDetailOpen && recordDetail" class="modal-backdrop" role="presentation" @click.self="closeRecordDetail">
  <section class="confirm-dialog record-detail-dialog" role="dialog" aria-modal="true">
    <header class="record-detail-header">
      <div class="record-detail-title">
        <div class="dialog-icon record-detail-icon"><FileText :size="22" /></div>
        <div>
          <p class="section-kicker">Record</p>
          <h2>{{ recordDetail.original_filename }}</h2>
        </div>
      </div>
      <button class="ghost icon-only" type="button" aria-label="关闭详情" @click="closeRecordDetail"><X :size="18" /></button>
    </header>

    <div class="record-detail-summary">
      <div class="record-stat">
        <span>源图床</span>
        <strong>{{ typeLabel(recordDetail.source_picbed) }}</strong>
      </div>
      <div class="record-stat">
        <span>目标图床</span>
        <strong>{{ typeLabel(recordDetail.target_picbed) }}</strong>
      </div>
      <div class="record-stat">
        <span>状态</span>
        <strong><span :class="['status', recordDetail.status]">{{ recordDetail.status === 'success' ? '成功' : '失败' }}</span></strong>
      </div>
      <div class="record-stat">
        <span>图片数</span>
        <strong>{{ recordDetail.image_count }}</strong>
      </div>
    </div>

    <p v-if="recordDetail.error_message" class="notice error record-detail-error">{{ recordDetail.error_message }}</p>

    <section class="record-detail-section">
      <div class="record-detail-section-head">
        <div>
          <p class="section-kicker">Images</p>
          <h3>替换明细</h3>
        </div>
        <div class="record-detail-head-actions">
          <span>{{ recordDetail.details?.length || 0 }} 条</span>
          <button class="secondary" type="button" :disabled="!recordDetail.converted_content" @click="downloadRecordContent">
            <Download :size="18" />下载转换结果
          </button>
        </div>
      </div>
      <div v-if="recordDetail.details?.length" class="record-detail-table-wrap">
        <table class="record-detail-table">
          <thead>
            <tr>
              <th>序号</th>
              <th>源地址</th>
              <th>目标地址</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(detail, index) in recordDetail.details" :key="detail.id">
              <td><span class="record-index">{{ index + 1 }}</span></td>
              <td>
                <code
                  class="url-preview"
                  tabindex="0"
                  :aria-label="detail.original_url"
                  @mouseenter="showUrlTooltip(detail.original_url, $event)"
                  @mousemove="updateUrlTooltipPosition($event)"
                  @mouseleave="hideUrlTooltip"
                  @focus="showUrlTooltip(detail.original_url, $event)"
                  @blur="hideUrlTooltip"
                >{{ detail.original_url }}</code>
              </td>
              <td>
                <a
                  v-if="detail.target_url"
                  class="url-preview"
                  :href="detail.target_url"
                  target="_blank"
                  rel="noopener noreferrer"
                  @mouseenter="showUrlTooltip(detail.target_url, $event)"
                  @mousemove="updateUrlTooltipPosition($event)"
                  @mouseleave="hideUrlTooltip"
                  @focus="showUrlTooltip(detail.target_url, $event)"
                  @blur="hideUrlTooltip"
                >
                  {{ detail.target_url }}
                </a>
                <span v-else>-</span>
              </td>
              <td><span :class="['status', detail.status]">{{ detail.status === 'success' ? '成功' : '失败' }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="empty record-detail-empty">暂无替换明细</p>
    </section>
  </section>
  <div
    v-if="urlTooltip.visible"
    class="url-tooltip"
    :style="{ left: `${urlTooltip.x}px`, top: `${urlTooltip.y}px` }"
    role="tooltip"
  >{{ urlTooltip.text }}</div>
</div>

</template>
