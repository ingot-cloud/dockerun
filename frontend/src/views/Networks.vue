<template>
  <div class="networks-view">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>网络列表</span>
          <div>
            <el-button type="primary" @click="createDialogVisible = true">
              <el-icon><Plus /></el-icon>
              创建网络
            </el-button>
            <el-button type="primary" @click="loadNetworks">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>
      
      <el-table :data="networks" v-loading="loading" border>
        <el-table-column prop="Name" label="名称" width="200" />
        <el-table-column prop="Id" label="网络ID" width="150">
          <template #default="{ row }">
            {{ row.Id.substring(0, 12) }}
          </template>
        </el-table-column>
        <el-table-column prop="Driver" label="驱动" width="100" />
        <el-table-column prop="Scope" label="范围" width="100" />
        <el-table-column label="子网" width="200">
          <template #default="{ row }">
            <div v-for="(config, index) in row.IPAM?.Config" :key="index">
              {{ config.Subnet }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button 
              v-if="!['bridge', 'host', 'none'].includes(row.Name)"
              type="danger" 
              size="small" 
              @click="handleRemove(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    
    <!-- 创建网络对话框 -->
    <el-dialog v-model="createDialogVisible" title="创建网络" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="网络名称">
          <el-input v-model="form.name" placeholder="请输入网络名称" />
        </el-form-item>
        <el-form-item label="驱动">
          <el-select v-model="form.driver" style="width: 100%">
            <el-option label="bridge" value="bridge" />
            <el-option label="overlay" value="overlay" />
            <el-option label="host" value="host" />
            <el-option label="macvlan" value="macvlan" />
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

const networks = ref<any[]>([])
const loading = ref(false)
const createDialogVisible = ref(false)

const form = reactive({
  name: '',
  driver: 'bridge'
})

const loadNetworks = async () => {
  loading.value = true
  try {
    networks.value = await dockerApi.networks.list() as any
  } catch (error) {
    ElMessage.error('加载网络列表失败')
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  if (!form.name) {
    ElMessage.warning('请输入网络名称')
    return
  }
  
  try {
    await dockerApi.networks.create(form.name, form.driver)
    ElMessage.success('网络创建成功')
    createDialogVisible.value = false
    form.name = ''
    form.driver = 'bridge'
    loadNetworks()
  } catch (error) {
    ElMessage.error('网络创建失败')
  }
}

const handleRemove = async (network: any) => {
  ElMessageBox.confirm('确定要删除这个网络吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await dockerApi.networks.remove(network.Id)
      ElMessage.success('网络删除成功')
      loadNetworks()
    } catch (error) {
      ElMessage.error('网络删除失败')
    }
  })
}

onMounted(() => {
  loadNetworks()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

