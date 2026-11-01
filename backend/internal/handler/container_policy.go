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

func commandDeploymentNotice(env models.Environment) string {
	if env.ReplaceContainers {
		return "命令语法检查通过。确认后将停止并删除关联容器，清理可删除的同 tag 本地镜像，再按本次摘要拉取并执行脚本。数据卷保留；拉取或执行失败不会自动恢复旧容器。"
	}
	return "命令语法检查通过；预览不执行脚本。容器替换由脚本负责，未开启系统替换。命令部署不自动回滚。"
}
