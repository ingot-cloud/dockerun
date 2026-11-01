package handler

import (
	"dockerun/internal/models"
	"dockerun/internal/release"
	"github.com/gin-gonic/gin"
	"slices"
)

func containerPolicy(env models.Environment) release.ContainerPolicy {
	p, _ := release.ContainerSettings(env)
	return p
}

func checkExpectedContainerPolicy(c *gin.Context, expected *release.ContainerPolicy, env models.Environment) bool {
	if err := release.ValidateContainerBindings(env); err != nil {
		fail(c, err)
		return false
	}
	actual := containerPolicy(env)
	if expected != nil && (expected.Replace != actual.Replace || !slices.Equal(expected.Names, actual.Names)) {
		c.JSON(409, gin.H{"error": "关联容器或替换策略已变化，请重新预览后部署"})
		return false
	}
	return true
}

func commandDeploymentNotice(env models.Environment, r models.Release) string {
	if env.ReplaceContainers {
		if r.BundleID != 0 {
			return "命令语法检查通过。确认后先加载并核对离线镜像，再停止并删除目标环境已关联的旧容器，执行此版本脚本；容器不存在时直接执行。不删除镜像、不拉取仓库。数据卷保留；脚本失败不会自动恢复旧容器。"
		}
		return "命令语法检查通过。确认后先准备此版本镜像，再停止并删除目标环境已关联的旧容器并执行脚本；容器不存在时直接执行。手动发布不清理镜像，未声明的镜像仍由脚本准备。数据卷保留；脚本失败不会自动恢复旧容器。"
	}
	return "命令语法检查通过；预览不执行脚本。容器替换由脚本负责，未开启系统替换。命令部署不自动回滚。"
}
