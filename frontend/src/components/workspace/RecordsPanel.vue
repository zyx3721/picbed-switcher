<script setup lang="ts">
import { RefreshCw } from 'lucide-vue-next';
import { useWorkspaceContext } from '../../composables/useWorkspaceContext';

const { records, typeLabel, loadRecords } = useWorkspaceContext();
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
        <tr v-for="record in records" :key="record.id">
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

</template>
