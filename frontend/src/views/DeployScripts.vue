<template>
  <div class="deploy-scripts-view">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>部署脚本</span>
          <div>
            <el-button type="primary" @click="handleCreate">
              <el-icon><Plus /></el-icon>
              新建脚本
            </el-button>
            <el-button type="primary" @click="loadScripts">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </div>
        </div>
      </template>
      
      <el-table :data="scripts" v-loading="loading" border>
        <el-table-column prop="name" label="名称" width="200" />
        <el-table-column prop="description" label="描述" />
        <el-table-column prop="script_type" label="类型" width="150">
          <template #default="{ row }">
            <el-tag>{{ row.script_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="关联容器" width="150">
          <template #default="{ row }">
            {{ row.container_id ? row.container_id.substring(0, 12) : '-' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button type="success" size="small" @click="handleExecute(row)">
              执行
            </el-button>
            <el-button type="primary" size="small" @click="handleEdit(row)">
              编辑
            </el-button>
            <el-button type="danger" size="small" @click="handleDelete(row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    
    <!-- 编辑脚本对话框 -->
    <el-dialog 
      v-model="dialogVisible" 
      :title="isEdit ? '编辑脚本' : '新建脚本'" 
      width="70%"
      :close-on-click-modal="false"
    >
      <el-form :model="form" label-width="100px">
        <el-form-item label="脚本名称">
          <el-input v-model="form.name" placeholder="请输入脚本名称" />
        </el-form-item>
        
        <el-form-item label="描述">
          <el-input v-model="form.description" placeholder="请输入脚本描述" />
        </el-form-item>
        
        <el-form-item label="脚本类型">
          <el-radio-group v-model="form.script_type">
            <el-radio label="shell">Shell脚本</el-radio>
            <el-radio label="docker-compose">Docker Compose</el-radio>
          </el-radio-group>
        </el-form-item>
        
        <el-form-item label="脚本内容">
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="15"
            placeholder="请输入脚本内容"
            style="font-family: monospace;"
          />
        </el-form-item>
        
        <el-form-item 
          v-if="form.script_type === 'docker-compose'" 
          label="Compose文件"
        >
          <el-input
            v-model="form.compose_file"
            type="textarea"
            :rows="15"
            placeholder="请输入docker-compose.yml内容"
            style="font-family: monospace;"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
    
    <!-- 执行输出对话框 -->
    <el-dialog v-model="outputVisible" title="执行输出" width="70%">
      <el-input
        v-model="output"
        type="textarea"
        :rows="20"
        readonly
        style="font-family: monospace;"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { deployApi, type DeployScript } from '@/api/deploy'

const scripts = ref<DeployScript[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const outputVisible = ref(false)
const output = ref('')
const isEdit = ref(false)
const currentId = ref(0)

const form = reactive<DeployScript>({
  name: '',
  description: '',
  script_type: 'shell',
  content: '',
  compose_file: ''
})

const exampleScript = `#!/usr/bin/env bash

VERSION=latest
IMAGE_NAME=nginx
SERVICE_NAME=my-nginx

# 停止运行当前容器
docker ps -q --filter name="\${SERVICE_NAME}" | xargs -r docker rm -f

# 拉取最新镜像
docker pull \${IMAGE_NAME}:\${VERSION}

# 启动容器
docker run -d --name \${SERVICE_NAME} \\
 --restart always \\
 -p 8080:80 \\
 \${IMAGE_NAME}:\${VERSION}
`

const loadScripts = async () => {
  loading.value = true
  try {
    scripts.value = await deployApi.list() as any
  } catch (error) {
    ElMessage.error('加载脚本列表失败')
  } finally {
    loading.value = false
  }
}

const getStatusType = (status: string) => {
  const map: Record<string, any> = {
    idle: 'info',
    running: 'warning',
    stopped: 'success',
    failed: 'danger'
  }
  return map[status] || 'info'
}

const handleCreate = () => {
  isEdit.value = false
  form.name = ''
  form.description = ''
  form.script_type = 'shell'
  form.content = exampleScript
  form.compose_file = ''
  dialogVisible.value = true
}

const handleEdit = async (script: DeployScript) => {
  isEdit.value = true
  currentId.value = script.id!
  
  try {
    const data = await deployApi.get(script.id!) as any
    form.name = data.name
    form.description = data.description
    form.script_type = data.script_type
    form.content = data.content
    form.compose_file = data.compose_file || ''
    dialogVisible.value = true
  } catch (error) {
    ElMessage.error('加载脚本详情失败')
  }
}

const handleSave = async () => {
  if (!form.name || !form.content) {
    ElMessage.warning('请填写完整信息')
    return
  }
  
  try {
    if (isEdit.value) {
      await deployApi.update(currentId.value, form)
      ElMessage.success('更新成功')
    } else {
      await deployApi.create(form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    loadScripts()
  } catch (error) {
    ElMessage.error('保存失败')
  }
}

const handleExecute = async (script: DeployScript) => {
  ElMessageBox.confirm('确定要执行这个脚本吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    const loading = ElMessage({
      message: '正在执行脚本...',
      type: 'info',
      duration: 0
    })
    
    try {
      const res: any = await deployApi.execute(script.id!)
      loading.close()
      output.value = res.output || '执行成功'
      outputVisible.value = true
      loadScripts()
    } catch (error: any) {
      loading.close()
      output.value = error.response?.data?.output || '执行失败'
      outputVisible.value = true
    }
  })
}

const handleDelete = async (script: DeployScript) => {
  ElMessageBox.confirm('确定要删除这个脚本吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deployApi.delete(script.id!)
      ElMessage.success('删除成功')
      loadScripts()
    } catch (error) {
      ElMessage.error('删除失败')
    }
  })
}

onMounted(() => {
  loadScripts()
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

