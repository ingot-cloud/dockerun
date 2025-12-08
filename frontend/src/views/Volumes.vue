<template>
  <div class="volumes-view">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>卷列表</span>
          <div>
            <el-button type="primary" @click="createDialogVisible = true">
              <el-icon><Plus /></el-icon>
              创建卷
            </el-button>
            <el-button type="primary" @click="loadVolumes">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>
      
      <el-table :data="volumes" v-loading="loading" border>
        <el-table-column prop="Name" label="名称" width="300" />
        <el-table-column prop="Driver" label="驱动" width="120" />
        <el-table-column prop="Mountpoint" label="挂载点" />
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.CreatedAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="danger" size="small" @click="handleRemove(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    
    <!-- 创建卷对话框 -->
    <el-dialog v-model="createDialogVisible" title="创建卷" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="卷名称">
          <el-input v-model="form.name" placeholder="请输入卷名称" />
        </el-form-item>
        <el-form-item label="驱动">
          <el-select v-model="form.driver" style="width: 100%">
            <el-option label="local" value="local" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { dockerApi } from '@/api/docker'

const volumes = ref<any[]>([])
const loading = ref(false)
const createDialogVisible = ref(false)

const form = reactive({
  name: '',
  driver: 'local'
})

const loadVolumes = async () => {
  loading.value = true
  try {
    const res: any = await dockerApi.volumes.list()
    volumes.value = res.Volumes || []
  } catch (error) {
    ElMessage.error('加载卷列表失败')
  } finally {
    loading.value = false
  }
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleString('zh-CN')
}

const handleCreate = async () => {
  if (!form.name) {
    ElMessage.warning('请输入卷名称')
    return
  }
  
  try {
    await dockerApi.volumes.create(form.name, form.driver)
    ElMessage.success('卷创建成功')
    createDialogVisible.value = false
    form.name = ''
    form.driver = 'local'
    loadVolumes()
  } catch (error) {
    ElMessage.error('卷创建失败')
  }
}

const handleRemove = async (volume: any) => {
  ElMessageBox.confirm('确定要删除这个卷吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await dockerApi.volumes.remove(volume.Name, false)
      ElMessage.success('卷删除成功')
      loadVolumes()
    } catch (error) {
      ElMessage.error('卷删除失败')
    }
  })
}

onMounted(() => {
  loadVolumes()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

