<template>
  <div class="images-view">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>镜像列表</span>
          <div>
            <el-button type="primary" @click="pullDialogVisible = true">
              <el-icon><Download /></el-icon>
              拉取镜像
            </el-button>
            <el-button type="primary" @click="loadImages">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>
      
      <el-table :data="images" v-loading="loading" border>
        <el-table-column label="仓库:标签" width="350">
          <template #default="{ row }">
            <div v-for="(tag, index) in row.RepoTags" :key="index">
              {{ tag }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="Id" label="镜像ID" width="150">
          <template #default="{ row }">
            {{ row.Id.replace('sha256:', '').substring(0, 12) }}
          </template>
        </el-table-column>
        <el-table-column label="大小" width="120">
          <template #default="{ row }">
            {{ formatSize(row.Size) }}
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.Created) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" @click="handleTag(row)">
              标签
            </el-button>
            <el-button type="danger" size="small" @click="handleRemove(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    
    <!-- 拉取镜像对话框 -->
    <el-dialog v-model="pullDialogVisible" title="拉取镜像" width="500px">
      <el-form>
        <el-form-item label="镜像名称">
          <el-input v-model="pullImageName" placeholder="例如: nginx:latest" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pullDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handlePull" :loading="pulling">拉取</el-button>
      </template>
    </el-dialog>
    
    <!-- 打标签对话框 -->
    <el-dialog v-model="tagDialogVisible" title="创建标签" width="500px">
      <el-form>
        <el-form-item label="源镜像">
          <el-input v-model="tagForm.source" readonly />
        </el-form-item>
        <el-form-item label="新标签">
          <el-input v-model="tagForm.target" placeholder="例如: myregistry/myimage:v1.0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="tagDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreateTag">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { dockerApi } from '@/api/docker'

const images = ref<any[]>([])
const loading = ref(false)
const pullDialogVisible = ref(false)
const tagDialogVisible = ref(false)
const pullImageName = ref('')
const pulling = ref(false)

const tagForm = reactive({
  source: '',
  target: ''
})

const loadImages = async () => {
  loading.value = true
  try {
    images.value = await dockerApi.images.list() as any
  } catch (error) {
    ElMessage.error('加载镜像列表失败')
  } finally {
    loading.value = false
  }
}

const formatSize = (size: number) => {
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(2) + ' KB'
  if (size < 1024 * 1024 * 1024) return (size / (1024 * 1024)).toFixed(2) + ' MB'
  return (size / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

const formatDate = (timestamp: number) => {
  return new Date(timestamp * 1000).toLocaleString('zh-CN')
}

const handlePull = async () => {
  if (!pullImageName.value) {
    ElMessage.warning('请输入镜像名称')
    return
  }
  
  pulling.value = true
  try {
    await dockerApi.images.pull(pullImageName.value)
    ElMessage.success('镜像拉取成功')
    pullDialogVisible.value = false
    pullImageName.value = ''
    loadImages()
  } catch (error) {
    ElMessage.error('镜像拉取失败')
  } finally {
    pulling.value = false
  }
}

const handleTag = (image: any) => {
  tagForm.source = image.RepoTags?.[0] || image.Id
  tagForm.target = ''
  tagDialogVisible.value = true
}

const handleCreateTag = async () => {
  if (!tagForm.target) {
    ElMessage.warning('请输入新标签')
    return
  }
  
  try {
    await dockerApi.images.tag(tagForm.source, tagForm.target)
    ElMessage.success('标签创建成功')
    tagDialogVisible.value = false
    loadImages()
  } catch (error) {
    ElMessage.error('标签创建失败')
  }
}

const handleRemove = async (image: any) => {
  ElMessageBox.confirm('确定要删除这个镜像吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await dockerApi.images.remove(image.Id, false)
      ElMessage.success('镜像删除成功')
      loadImages()
    } catch (error) {
      ElMessage.error('镜像删除失败')
    }
  })
}

onMounted(() => {
  loadImages()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

