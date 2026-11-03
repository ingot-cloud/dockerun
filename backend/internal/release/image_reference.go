package release

import (
	"dockerun/internal/models"
	"strings"

	"github.com/distribution/reference"
)

const ImageReferenceLabel = "io.dockerun.image-reference"
const ImageIDLabel = "io.dockerun.image-id"

// VerifiedImageReference 只在实际镜像身份一致时补充名称，不用当前 tag 推测运行镜像。
func VerifiedImageReference(name, expectedID, actualID string) string {
	if actualID == "" || expectedID != actualID || name == "" || strings.HasPrefix(name, "sha256:") {
		return ""
	}
	if _, err := reference.ParseNormalizedNamed(name); err != nil {
		return ""
	}
	return name
}

func preparedImageReference(r models.Release, image Image) string {
	name := image.Reference
	if r.TriggerImage != "" && r.TriggerDigest != "" && image.Digest == r.TriggerDigest {
		trigger, triggerErr := reference.ParseNormalizedNamed(r.TriggerImage)
		source, sourceErr := reference.ParseNormalizedNamed(image.Reference)
		if triggerErr == nil && sourceErr == nil && trigger.Name() == source.Name() {
			name = r.TriggerImage
		}
	}
	return VerifiedImageReference(name, image.ID, image.ID)
}
