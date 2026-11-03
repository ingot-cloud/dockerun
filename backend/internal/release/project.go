package release

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
	dockerTypes "github.com/docker/docker/api/types"
)

// 自定义项目名不能接管目标主机上已有的同名项目；旧自动命名容器兼容无归属标签。
func checkProjectOwnership(project *types.Project, containers []dockerTypes.Container) error {
	for _, service := range project.Services {
		application, environment := service.Labels["io.dockerun.application"], service.Labels["io.dockerun.environment"]
		if application == "" || environment == "" {
			continue
		}
		legacyName := fmt.Sprintf("dockerun-a%s-e%s", application, environment)
		for _, existing := range containers {
			if existing.Labels["com.docker.compose.project"] != project.Name {
				continue
			}
			ownerApp, ownerEnv := existing.Labels["io.dockerun.application"], existing.Labels["io.dockerun.environment"]
			if ownerApp == application && ownerEnv == environment {
				continue
			}
			if project.Name == legacyName && ownerApp == "" && ownerEnv == "" {
				continue
			}
			return fmt.Errorf("目标主机上的 Compose 项目 %s 已有不属于此环境的容器，请使用其他项目名称", project.Name)
		}
		break // Render 为全部服务写入相同的归属标签。
	}
	return nil
}
